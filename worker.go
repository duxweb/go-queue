package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/samber/lo"
)

// HandlerFunc 处理器函数定义
// HandlerFunc handler function definition
type HandlerFunc func(ctx context.Context, params []byte) error

// WorkerStatus 工作线程状态
// WorkerStatus worker status
type WorkerStatus int

const (
	StatusStopped WorkerStatus = iota
	StatusRunning
	StatusPaused
)

// String 将工作状态转换为字符串表示
// String convert worker status to string representation
func (s WorkerStatus) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusRunning:
		return "running"
	case StatusPaused:
		return "paused"
	default:
		return "unknown"
	}
}

// WorkerConfig 工作线程配置
// WorkerConfig worker configuration
type WorkerConfig struct {
	DeviceName  string
	Num         int
	Interval    time.Duration
	Retry       int
	RetryDelay  time.Duration
	Timeout     time.Duration
	SuccessFunc func(item *QueueItem)
	FailFunc    func(item *QueueItem, err error)
}

// Worker 工作器
// Worker worker
type Worker struct {
	Name       string        // 工作器名称 | Worker name
	num        int           // 并发数量 | Concurrency count
	interval   time.Duration // 轮询间隔 | Polling interval
	retry      int           // 重试次数 | Retry count
	retryDelay time.Duration // 重试间隔 | Retry delay
	timeout    time.Duration // 每个任务超时时间 | Timeout for each task

	Driver      QueueDriver
	successFunc func(item *QueueItem)
	failFunc    func(item *QueueItem, err error)

	ctx        context.Context
	cancelFunc context.CancelFunc
	pool       *ants.Pool
	status     WorkerStatus
	lock       sync.Mutex
	ticker     *time.Ticker

	totalProcessed int64 // 处理的任务总数 | Total number of processed tasks
	totalSuccess   int64 // 处理成功的任务数 | Number of successfully processed tasks
	totalFailed    int64 // 处理失败的任务数 | Number of failed tasks
	totalRetry     int64 // 重试次数 | Number of retries
	totalTime      int64 // 处理任务总时间（纳秒） | Total task processing time (nanoseconds)
}

// NewWorker 创建新的工作线程
// NewWorker create a new worker
func NewWorker(config *WorkerConfig) *Worker {

	return &Worker{
		Name:       config.DeviceName,
		num:        lo.Ternary(config.Num == 0, 1, config.Num),
		interval:   lo.Ternary(config.Interval == 0, time.Second*10, config.Interval),
		retry:      lo.Ternary(config.Retry == 0, 3, config.Retry),
		retryDelay: lo.Ternary(config.RetryDelay == 0, time.Second*1, config.RetryDelay),
		timeout:    lo.Ternary(config.Timeout == 0, time.Second*300, config.Timeout),

		status:      StatusStopped,
		successFunc: config.SuccessFunc,
		failFunc:    config.FailFunc,
	}
}

// Start 启动工作线程
// Start the worker
func (w *Worker) Start(ctx context.Context, handlers map[string]HandlerFunc) error {
	w.lock.Lock()

	if w.status != StatusStopped {
		w.lock.Unlock()
		return nil
	}

	w.ctx, w.cancelFunc = context.WithCancel(ctx)

	var err error
	w.pool, err = ants.NewPool(w.num, ants.WithPreAlloc(true))
	if err != nil {
		w.lock.Unlock()
		return fmt.Errorf("创建协程池失败: %w", err)
	}

	w.status = StatusRunning
	// 使用更短的轮询间隔，以便更频繁地检查队列
	w.ticker = time.NewTicker(time.Millisecond * 10)
	w.lock.Unlock()

	defer func() {
		w.ticker.Stop()

		if w.pool != nil {
			w.pool.Release()
		}

		w.lock.Lock()
		w.status = StatusStopped
		w.lock.Unlock()
	}()

	for {
		select {
		case <-w.ctx.Done():
			return nil
		case <-w.ticker.C:
			w.lock.Lock()
			status := w.status
			w.lock.Unlock()

			if status == StatusPaused {
				continue
			}

			// 总是尝试获取尽可能多的任务
			items := w.Driver.Pop(w.Name, w.num)
			if len(items) == 0 {
				continue
			}

			for _, item := range items {
				// 创建任务副本，防止闭包中的指针问题
				taskItem := *item
				err := w.pool.Submit(func() {
					w.safeProcessTask(&taskItem, handlers)
				})

				if err != nil {
					w.handleTaskFailure(&taskItem, err)
				}
			}
		}
	}
}

// safeProcessTask 安全处理任务，包含panic恢复逻辑
// safeProcessTask safe task processing, including panic recovery logic
func (w *Worker) safeProcessTask(item *QueueItem, handlers map[string]HandlerFunc) {
	atomic.AddInt64(&w.totalProcessed, 1)
	startTime := time.Now()

	defer func() {
		processTime := time.Since(startTime).Nanoseconds()
		atomic.AddInt64(&w.totalTime, processTime)

		if r := recover(); r != nil {
			err := fmt.Errorf("panic: %v", r)
			w.handleTaskFailure(item, err)
		}
	}()

	handler, ok := handlers[item.HandlerName]
	if !ok {
		err := fmt.Errorf("handler not found for task[%s]", item.HandlerName)
		w.handleTaskFailure(item, err)
		return
	}

	err := w.executeWithRetry(handler, item)

	if err == nil {
		atomic.AddInt64(&w.totalSuccess, 1)
		if w.successFunc != nil {
			w.successFunc(item)
		}
	} else {
		w.handleTaskFailure(item, err)
	}
}

// handleTaskFailure 处理任务失败
// handleTaskFailure handle task failure
func (w *Worker) handleTaskFailure(item *QueueItem, err error) {
	// 每次失败都调用失败回调
	if w.failFunc != nil {
		w.failFunc(item, err)
	}

	if item.Retried >= w.retry {
		atomic.AddInt64(&w.totalFailed, 1)
		return
	}

	// 处理重试
	item.Retried++
	atomic.AddInt64(&w.totalRetry, 1)
	// 添加重试延迟
	item.CreatedAt = time.Now().Add(w.retryDelay)
	w.Driver.Add(w.Name, item)
}

// executeWithRetry 执行函数，带重试
// executeWithRetry execute function with retry
func (w *Worker) executeWithRetry(handler HandlerFunc, item *QueueItem) error {
	ctx, cancel := context.WithTimeout(w.ctx, w.timeout)
	defer cancel()

	err := handler(ctx, item.Params)
	return err
}

// Pause 暂停工作线程
// Pause the worker
func (w *Worker) Pause() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.status != StatusRunning {
		return nil
	}

	w.status = StatusPaused
	return nil
}

// Resume 恢复工作线程
// Resume the worker
func (w *Worker) Resume() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.status != StatusPaused {
		return nil
	}

	w.status = StatusRunning
	return nil
}

// Stop 停止工作线程
// Stop the worker
func (w *Worker) Stop() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.status == StatusStopped {
		return nil
	}

	if w.cancelFunc != nil {
		w.cancelFunc()
	}

	if w.pool != nil {
		w.pool.Release()
	}

	return nil
}

// GetStatus 获取工作线程状态
// GetStatus get worker status
func (w *Worker) GetStatus() WorkerStatus {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.status
}

// getAvgProcessTime 获取平均处理时间
// getAvgProcessTime get average processing time
func (w *Worker) getAvgProcessTime() time.Duration {
	processed := atomic.LoadInt64(&w.totalProcessed)
	if processed == 0 {
		return 0
	}

	totalTime := atomic.LoadInt64(&w.totalTime)
	return time.Duration(totalTime / processed)
}

// getSuccessRate 获取成功率
// getSuccessRate get success rate
func (w *Worker) getSuccessRate() float64 {
	processed := atomic.LoadInt64(&w.totalProcessed)
	if processed == 0 {
		return 0
	}

	success := atomic.LoadInt64(&w.totalSuccess)
	return float64(success) / float64(processed)
}

// GetTotal 获取工作线程统计数据
// GetTotal get worker statistics
func (w *Worker) GetTotal() map[string]any {
	return map[string]any{
		"processed":    atomic.LoadInt64(&w.totalProcessed),
		"success":      atomic.LoadInt64(&w.totalSuccess),
		"failed":       atomic.LoadInt64(&w.totalFailed),
		"retry":        atomic.LoadInt64(&w.totalRetry),
		"avg_time":     w.getAvgProcessTime().String(),
		"success_rate": w.getSuccessRate(),
		"workers":      w.num,
		"status":       w.GetStatus().String(),
	}
}

func (w *Worker) Service() QueueDriver {
	return w.Driver
}

// SetSuccessFunc 设置成功回调函数
// SetSuccessFunc set success callback function
func (w *Worker) SetSuccessFunc(f func(item *QueueItem)) {
	w.successFunc = f
}

// SetFailFunc 设置失败回调函数
// SetFailFunc set fail callback function
func (w *Worker) SetFailFunc(f func(item *QueueItem, err error)) {
	w.failFunc = f
}
