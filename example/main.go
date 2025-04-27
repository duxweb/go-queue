package main

import (
	"context"
	"fmt"
	"log"
	"time"

	goqueue "github.com/duxweb/go-queue"
	internalQueue "github.com/duxweb/go-queue/internal/queue"
	internalWorker "github.com/duxweb/go-queue/internal/worker"
	"github.com/duxweb/go-queue/pkg/queue/memory"
	pkgWorker "github.com/duxweb/go-queue/pkg/worker"
)

func main() {
	// 创建一个上下文
	ctx := context.Background()

	// 创建队列服务配置
	config := &goqueue.Config{
		Context: ctx,
	}

	// 创建新的队列服务实例
	queueService, err := goqueue.New(config)
	if err != nil {
		log.Fatalf("创建队列服务失败: %v", err)
	}

	// 创建一个内存队列
	memQueue := memory.NewMemoryQueue()

	// 注册内存队列服务
	queueService.RegisterService("memory-queue", memQueue)

	// 创建一个工作队列配置
	workerConfig := &internalWorker.WorkerConfig{
		ServiceName: "memory-queue",
		Num:         5,
		Interval:    time.Second * 1,
		Retry:       3,
		RetryDelay:  time.Second * 5,
		Timeout:     time.Minute,
	}

	// 创建工作队列
	workerInstance := pkgWorker.NewWorker(workerConfig)

	// 注册工作队列
	queueService.RegisterWorker("memory-queue", workerInstance)

	// 注册处理器函数
	queueService.RegisterHandler("log-handler", func(ctx context.Context, params []byte) error {
		fmt.Printf("处理任务，参数: %s\n", string(params))
		return nil
	})

	// 添加一个即时任务
	err = queueService.Add("memory-queue", &internalQueue.QueueConfig{
		HandlerName: "log-handler",
		Params:      []byte(`{"message":"这是一个即时任务"}`),
	})
	if err != nil {
		log.Fatalf("添加即时任务失败: %v", err)
	}

	// 添加一个延迟任务
	err = queueService.AddDelay("memory-queue", &internalQueue.QueueDelayConfig{
		QueueConfig: internalQueue.QueueConfig{
			HandlerName: "log-handler",
			Params:      []byte(`{"message":"这是一个延迟任务"}`),
		},
		Delay: time.Second * 10, // 10秒后执行
	})
	if err != nil {
		log.Fatalf("添加延迟任务失败: %v", err)
	}

	// 启动所有工作池
	err = queueService.Start()
	if err != nil {
		log.Fatalf("启动工作池失败: %v", err)
	}

	// 等待任务处理完成
	time.Sleep(time.Second * 15)
	fmt.Println("示例程序运行完成")
}
