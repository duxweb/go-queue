package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/duxweb/go-queue/internal/queue"
	"github.com/duxweb/go-queue/internal/worker"
	"github.com/panjf2000/ants/v2"
)

// DefaultWorker 默认工作器实现
type DefaultWorker struct {
	serviceName string        // 服务名称 | Service name
	num         int           // 并发数量 | Concurrency count
	interval    time.Duration // 轮询间隔 | Polling interval
	retry       int           // 重试次数 | Retry count
	retryDelay  time.Duration // 重试间隔 | Retry delay
	timeout     time.Duration // 每个任务超时时间 | Timeout for each task

	successFunc func(item *queue.QueueItem)
	failFunc    func(item *queue.QueueItem, err error)

	ctx        context.Context
	cancelFunc context.CancelFunc
	pool       *ants.Pool
	status     worker.WorkerStatus
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
func NewWorker(config *worker.WorkerConfig) worker.Worker {
	return &DefaultWorker{
		serviceName: config.ServiceName,
		num:         config.Num,
		interval:    config.Interval,
		retry:       config.Retry,
		retryDelay:  config.RetryDelay,
		status:      worker.StatusStopped,
		timeout:     config.Timeout,
		successFunc: config.SuccessFunc,
		failFunc:    config.FailFunc,
	}
}

// Start 启动工作线程
// Start the worker
func (w *DefaultWorker) Start(ctx context.Context, service queue.QueueService, handlers map[string]worker.HandlerFunc) error {
	w.lock.Lock()

	if w.status != worker.StatusStopped {
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

	w.status = worker.StatusRunning
	w.ticker = time.NewTicker(w.interval)
	w.lock.Unlock()

	defer func() {
		w.ticker.Stop()

		if w.pool != nil {
			w.pool.Release()
		}

		w.lock.Lock()
		w.status = worker.StatusStopped
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

			if status == worker.StatusPaused {
				continue
			}

			freeWorkers := w.pool.Free()
			if freeWorkers == 0 {
				continue
			}

			fetchNum := w.num
			if fetchNum > freeWorkers {
				fetchNum = freeWorkers
			}

			items := service.Pop(w.serviceName, fetchNum)
			if len(items) == 0 {
				continue
			}

			for _, item := range items {
				taskItem := *item
				err := w.pool.Submit(func() {
					w.safeProcessTask(&taskItem, service, handlers)
				})

				if err != nil {
					atomic.AddInt64(&w.totalFailed, 1)
					if w.failFunc != nil {
						w.failFunc(&taskItem, err)
					}
				}
			}
		}
	}
}

// safeProcessTask 安全处理任务，包含panic恢复逻辑
// safeProcessTask safe task processing, including panic recovery logic
func (w *DefaultWorker) safeProcessTask(item *queue.QueueItem, service queue.QueueService, handlers map[string]worker.HandlerFunc) {
	atomic.AddInt64(&w.totalProcessed, 1)
	startTime := time.Now()

	defer func() {
		processTime := time.Since(startTime).Nanoseconds()
		atomic.AddInt64(&w.totalTime, processTime)

		if r := recover(); r != nil {
			atomic.AddInt64(&w.totalFailed, 1)
			if w.failFunc != nil {
				w.failFunc(item, fmt.Errorf("panic: %v", r))
			}
		}
	}()

	handler, ok := handlers[item.HandlerName]
	if !ok {
		atomic.AddInt64(&w.totalFailed, 1)
		if w.failFunc != nil {
			w.failFunc(item, fmt.Errorf("handler not found for task[%s]", item.HandlerName))
		}
		return
	}

	err := w.executeWithRetry(handler, item)

	if err == nil {
		atomic.AddInt64(&w.totalSuccess, 1)
		if w.successFunc != nil {
			w.successFunc(item)
		}
	} else if item.Retried >= w.retry {
		atomic.AddInt64(&w.totalFailed, 1)
		if w.failFunc != nil {
			w.failFunc(item, err)
		}
	} else {
		item.Retried++
		atomic.AddInt64(&w.totalRetry, 1)
		service.Add(w.serviceName, item)
	}
}

// executeWithRetry 执行函数，带重试
// executeWithRetry execute function with retry
func (w *DefaultWorker) executeWithRetry(handler worker.HandlerFunc, item *queue.QueueItem) error {
	var err error

	for i := 0; i <= w.retry; i++ {
		ctx, cancel := context.WithTimeout(w.ctx, w.timeout)
		err = handler(ctx, item.Params)
		cancel()

		if err == nil {
			return nil
		}

		if i < w.retry {
			time.Sleep(w.retryDelay)
		}
	}

	return err
}

// Pause 暂停工作线程
// Pause the worker
func (w *DefaultWorker) Pause() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.status != worker.StatusRunning {
		return nil
	}

	w.status = worker.StatusPaused
	return nil
}

// Resume 恢复工作线程
// Resume the worker
func (w *DefaultWorker) Resume() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.status != worker.StatusPaused {
		return nil
	}

	w.status = worker.StatusRunning
	return nil
}

// Stop 停止工作线程
// Stop the worker
func (w *DefaultWorker) Stop() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.status == worker.StatusStopped {
		return nil
	}

	if w.cancelFunc != nil {
		w.cancelFunc()
	}

	return nil
}

// GetStatus 获取工作线程状态
// GetStatus get worker status
func (w *DefaultWorker) GetStatus() worker.WorkerStatus {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.status
}

// getAvgProcessTime 获取平均处理时间
// getAvgProcessTime get average processing time
func (w *DefaultWorker) getAvgProcessTime() time.Duration {
	processed := atomic.LoadInt64(&w.totalProcessed)
	if processed == 0 {
		return 0
	}

	totalTime := atomic.LoadInt64(&w.totalTime)
	return time.Duration(totalTime / processed)
}

// getSuccessRate 获取成功率
// getSuccessRate get success rate
func (w *DefaultWorker) getSuccessRate() float64 {
	processed := atomic.LoadInt64(&w.totalProcessed)
	if processed == 0 {
		return 0
	}

	success := atomic.LoadInt64(&w.totalSuccess)
	return float64(success) / float64(processed)
}

// GetTotal 获取工作线程统计数据
// GetTotal get worker statistics
func (w *DefaultWorker) GetTotal() map[string]any {
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
