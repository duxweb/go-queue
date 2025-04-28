package queue_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/drivers"
	"github.com/stretchr/testify/assert"
)

func TestWorkerBasicFunctionality(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建队列服务
	config := &queue.Config{
		Context: ctx,
	}

	queueService, err := queue.New(config)
	assert.NoError(t, err, "创建队列服务应该成功")

	// 创建内存队列
	memQueue := drivers.NewMemoryQueue()
	queueService.RegisterDriver("test-worker", memQueue)

	// 创建工作队列配置
	workerConfig := &queue.WorkerConfig{
		ServiceName: "test-worker",
		Num:         2,
		Interval:    time.Millisecond * 100,
		Retry:       2,
		RetryDelay:  time.Millisecond * 100,
		Timeout:     time.Second * 5,
	}

	// 注册工作器
	err = queueService.RegisterWorker("test-worker", workerConfig)
	assert.NoError(t, err, "注册工作器应该成功")

	// 跟踪成功处理的任务
	var processedCount int32 = 0

	// 注册任务处理器
	queueService.RegisterHandler("test-handler", func(ctx context.Context, params []byte) error {
		atomic.AddInt32(&processedCount, 1)
		return nil
	})

	// 添加五个任务
	for i := 0; i < 5; i++ {
		id, err := queueService.Add("test-worker", &queue.QueueConfig{
			HandlerName: "test-handler",
			Params:      []byte(`{"action":"test"}`),
		})
		assert.NoError(t, err, "添加任务应该成功")
		assert.NotEmpty(t, id, "任务ID不应该为空")
	}

	// 启动队列处理
	err = queueService.Start()
	assert.NoError(t, err, "启动队列服务应该成功")

	// 等待任务完成
	time.Sleep(time.Second * 3)

	// 验证所有任务已处理
	assert.Equal(t, int32(5), atomic.LoadInt32(&processedCount), "应该处理所有5个任务")

	// 关闭队列服务
	err = queueService.Stop()
	assert.NoError(t, err, "停止队列服务应该成功")
}

func TestWorkerRetry(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建队列服务
	config := &queue.Config{
		Context: ctx,
	}

	queueService, err := queue.New(config)
	assert.NoError(t, err, "创建队列服务应该成功")

	// 创建内存队列
	memQueue := drivers.NewMemoryQueue()
	queueService.RegisterDriver("test-retry", memQueue)

	// 创建工作队列配置
	workerConfig := &queue.WorkerConfig{
		ServiceName: "test-retry",
		Num:         1,
		Interval:    time.Millisecond * 100,
		Retry:       2,
		RetryDelay:  time.Millisecond * 200,
		Timeout:     time.Second * 1,
	}

	// 注册工作器
	err = queueService.RegisterWorker("test-retry", workerConfig)
	assert.NoError(t, err, "注册工作器应该成功")

	// 跟踪任务处理的尝试次数
	var attemptCount int32 = 0
	var successCount int32 = 0
	var failCount int32 = 0

	// 记录成功和失败
	worker := queueService.GetWorker("test-retry")
	worker.SetSuccessFunc(func(item *queue.QueueItem) {
		atomic.AddInt32(&successCount, 1)
	})
	worker.SetFailFunc(func(item *queue.QueueItem, err error) {
		atomic.AddInt32(&failCount, 1)
	})

	// 注册失败后重试的任务处理器
	queueService.RegisterHandler("retry-handler", func(ctx context.Context, params []byte) error {
		count := atomic.AddInt32(&attemptCount, 1)

		// 前两次尝试失败，第三次成功
		if count < 3 {
			return assert.AnError
		}

		return nil
	})

	// 添加一个会失败然后重试的任务
	id, err := queueService.Add("test-retry", &queue.QueueConfig{
		HandlerName: "retry-handler",
		Params:      []byte(`{"action":"retry"}`),
	})
	assert.NoError(t, err, "添加任务应该成功")
	assert.NotEmpty(t, id, "任务ID不应该为空")

	// 启动队列处理
	err = queueService.Start()
	assert.NoError(t, err, "启动队列服务应该成功")

	// 等待任务重试和完成
	time.Sleep(time.Second * 5)

	// 验证尝试次数
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount), "应该尝试3次")
	assert.Equal(t, int32(1), atomic.LoadInt32(&successCount), "应该有1次成功")
	assert.Equal(t, int32(2), atomic.LoadInt32(&failCount), "应该有2次失败")

	// 关闭队列服务
	err = queueService.Stop()
	assert.NoError(t, err, "停止队列服务应该成功")
}

func TestWorkerTimeout(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建队列服务
	config := &queue.Config{
		Context: ctx,
	}

	queueService, err := queue.New(config)
	assert.NoError(t, err, "创建队列服务应该成功")

	// 创建内存队列
	memQueue := drivers.NewMemoryQueue()
	queueService.RegisterDriver("test-timeout", memQueue)

	// 记录失败回调
	var timeoutCount int32 = 0

	// 创建工作队列配置
	workerConfig := &queue.WorkerConfig{
		ServiceName: "test-timeout",
		Num:         1,
		Interval:    time.Millisecond * 100,
		Retry:       1,
		RetryDelay:  time.Millisecond * 200,
		Timeout:     time.Millisecond * 100, // 设置非常短的超时时间
	}

	// 注册工作器
	err = queueService.RegisterWorker("test-timeout", workerConfig)
	assert.NoError(t, err, "注册工作器应该成功")

	// 设置失败回调
	worker := queueService.GetWorker("test-timeout")
	worker.SetFailFunc(func(item *queue.QueueItem, err error) {
		atomic.AddInt32(&timeoutCount, 1)
	})

	// 注册一个会超时的任务处理器
	queueService.RegisterHandler("timeout-handler", func(ctx context.Context, params []byte) error {
		// 模拟长时间运行的任务
		select {
		case <-ctx.Done():
			// 上下文被取消（超时）
			return ctx.Err()
		case <-time.After(time.Second * 1):
			// 这个不应该执行到，因为任务会提前超时
			return nil
		}
	})

	// 添加一个会超时的任务
	id, err := queueService.Add("test-timeout", &queue.QueueConfig{
		HandlerName: "timeout-handler",
		Params:      []byte(`{"action":"timeout"}`),
	})
	assert.NoError(t, err, "添加任务应该成功")
	assert.NotEmpty(t, id, "任务ID不应该为空")
	// 启动队列处理
	err = queueService.Start()
	assert.NoError(t, err, "启动队列服务应该成功")

	// 等待任务执行和超时
	time.Sleep(time.Second * 3)

	// 验证失败次数
	// 应该是原始执行 + 重试 = 2次
	assert.Equal(t, int32(2), atomic.LoadInt32(&timeoutCount), "应该有两次超时失败")

	// 关闭队列服务
	err = queueService.Stop()
	assert.NoError(t, err, "停止队列服务应该成功")
}
