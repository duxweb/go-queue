package integration

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	goqueue "github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/internal/queue"
	internalWorker "github.com/duxweb/go-queue/internal/worker"
	"github.com/duxweb/go-queue/pkg/queue/memory"
	"github.com/duxweb/go-queue/pkg/worker"
)

func TestMemoryDelayQueue(t *testing.T) {
	// 创建一个上下文，用于控制测试
	// Create a context for test control
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建队列服务配置
	// Create queue service configuration
	config := &goqueue.Config{
		Context: ctx,
	}

	// 创建新的队列服务实例
	// Create a new queue service instance
	queueService, err := goqueue.New(config)
	if err != nil {
		t.Fatalf("Failed to create queue service: %v", err)
	}

	// 创建一个内存队列
	// Create a memory queue
	memQueue := memory.NewMemoryQueue()

	// 注册内存队列服务
	// Register memory queue service
	queueService.RegisterService("delay-queue", memQueue)

	// 创建一个工作队列配置
	// Create a worker configuration
	workerConfig := &internalWorker.WorkerConfig{
		ServiceName: "delay-queue",
		Num:         1, // 只使用一个工作线程 | Only use one worker thread
		Interval:    time.Millisecond * 50,
		Retry:       1,
		RetryDelay:  time.Millisecond * 100,
		Timeout:     time.Second,
	}

	// 创建工作队列
	// Create worker instance
	workerInstance := worker.NewWorker(workerConfig)

	// 注册工作队列
	// Register worker
	queueService.RegisterWorker("delay-queue", workerInstance)

	// 用于记录任务处理时间
	// For recording task processing time
	var processTime atomic.Value
	processTime.Store(time.Time{})

	// 注册处理器函数
	// Register handler function
	queueService.RegisterHandler("delay-handler", func(ctx context.Context, params []byte) error {
		// 记录处理时间
		// Record processing time
		processTime.Store(time.Now())
		return nil
	})

	// 启动所有工作池
	// Start all worker pools
	err = queueService.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}

	startTime := time.Now()

	// 添加一个延迟任务
	// Add a delayed task
	delayTime := time.Millisecond * 500
	err = queueService.AddDelay("delay-queue", &queue.QueueDelayConfig{
		QueueConfig: queue.QueueConfig{
			HandlerName: "delay-handler",
			Params:      []byte(`{"test":"data"}`),
		},
		Delay: delayTime,
	})
	if err != nil {
		t.Errorf("Failed to add delayed task: %v", err)
	}

	// 等待足够的时间让任务完成处理
	// Wait enough time for the task to be processed
	time.Sleep(time.Second * 2)

	// 验证任务是否已处理
	// Verify the task was processed
	procTime := processTime.Load().(time.Time)
	if procTime.IsZero() {
		t.Fatalf("Task was not processed")
	}

	// 验证任务是否遵守延迟时间
	// Verify the task respected the delay time
	actualDelay := procTime.Sub(startTime)
	minExpectedDelay := delayTime - time.Millisecond*100 // 允许有100ms的误差 | Allow 100ms margin of error

	if actualDelay < minExpectedDelay {
		t.Errorf("Task should have been delayed at least %v before execution, but was executed after %v",
			minExpectedDelay, actualDelay)
	} else {
		t.Logf("Task was correctly delayed by %v before execution", actualDelay)
	}

	// 测试通过
	// Test passed
	t.Log("Memory delay queue test passed")
}
