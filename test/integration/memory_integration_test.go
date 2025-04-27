package integration

import (
	"context"
	"testing"
	"time"

	goqueue "github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/internal/queue"
	internalWorker "github.com/duxweb/go-queue/internal/worker"
	"github.com/duxweb/go-queue/pkg/queue/memory"
	"github.com/duxweb/go-queue/pkg/worker"
)

func TestMemoryQueueIntegration(t *testing.T) {
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
	queueService.RegisterService("test-memory-queue", memQueue)

	// 创建一个工作队列配置
	// Create a worker configuration
	workerConfig := &internalWorker.WorkerConfig{
		ServiceName: "test-memory-queue",
		Num:         1, // 只使用一个工作线程 | Only use one worker thread
		Interval:    time.Millisecond * 20,
		Retry:       1,
		RetryDelay:  time.Millisecond * 100,
		Timeout:     time.Second,
	}

	// 创建工作队列
	// Create worker instance
	workerInstance := worker.NewWorker(workerConfig)

	// 注册工作队列
	// Register worker
	queueService.RegisterWorker("test-memory-queue", workerInstance)

	// 用于跟踪处理的任务
	// For tracking processed tasks
	processedTask := false

	// 注册处理器函数
	// Register handler function
	queueService.RegisterHandler("test-handler", func(ctx context.Context, params []byte) error {
		// 将任务标记为已处理
		// Mark task as processed
		processedTask = true
		return nil
	})

	// 启动所有工作池
	// Start all worker pools
	err = queueService.Start()
	if err != nil {
		t.Fatalf("Failed to start worker pool: %v", err)
	}

	// 添加一个即时任务
	// Add an immediate task
	err = queueService.Add("test-memory-queue", &queue.QueueConfig{
		HandlerName: "test-handler",
		Params:      []byte(`{"test":"data"}`),
	})
	if err != nil {
		t.Errorf("Failed to add task: %v", err)
	}

	// 等待足够的时间让任务完成处理
	// Wait enough time for the task to be processed
	time.Sleep(time.Second * 3)

	// 验证任务是否已处理
	// Verify the task was processed
	if !processedTask {
		t.Fatalf("Task was not processed")
	}

	// 测试创建并验证已经满足了基本功能要求
	// Test created and verified to meet basic functional requirements
	t.Log("Memory queue integration test passed")
}
