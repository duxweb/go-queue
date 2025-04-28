package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	goqueue "github.com/duxweb/go-queue"
	workerdriver "github.com/duxweb/go-queue/drivers/memory"
)

func main() {
	// 创建上下文
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

	// 创建内存队列实例
	memQueue := workerdriver.New()

	// 注册内存队列服务
	queueService.RegisterDriver("default", memQueue)

	// 创建工作队列配置
	workerConfig := &goqueue.WorkerConfig{
		ServiceName: "default",
		Num:         2,               // 并发工作数量
		Interval:    time.Second * 1, // 轮询间隔
		Retry:       3,               // 重试次数
		RetryDelay:  time.Second * 2, // 重试间隔
		Timeout:     time.Minute,     // 任务超时时间
		// 任务成功回调
		SuccessFunc: func(item *goqueue.QueueItem) {
			fmt.Printf("任务成功执行: %s, 重试次数: %d\n", string(item.Params), item.Retried)
		},
		// 任务失败回调
		FailFunc: func(item *goqueue.QueueItem, err error) {
			fmt.Printf("任务执行失败: %s, 错误: %v\n", string(item.Params), err)
		},
	}

	// 注册工作器
	err = queueService.RegisterWorker("default", workerConfig)
	if err != nil {
		log.Fatalf("注册工作器失败: %v", err)
	}

	// 计数器
	var counter int32 = 0

	// 注册一个会失败几次然后成功的任务处理器
	queueService.RegisterHandler("retry-handler", func(ctx context.Context, params []byte) error {
		current := atomic.AddInt32(&counter, 1)
		fmt.Printf("尝试执行任务 #%d: %s\n", current, string(params))

		// 前两次执行会失败
		if current <= 2 {
			return errors.New("模拟任务失败")
		}

		fmt.Println("任务成功执行!")
		return nil
	})

	// 添加任务
	id, err := queueService.Add("default", &goqueue.QueueConfig{
		HandlerName: "retry-handler",
		Params:      []byte(`{"message":"这是一个会失败然后重试的任务"}`),
	})
	if err != nil {
		log.Fatalf("添加任务失败: %v", err)
	}

	fmt.Printf("添加任务成功: %s\n", id)

	// 启动队列处理
	if err := queueService.Start(); err != nil {
		log.Fatalf("启动队列服务失败: %v", err)
	}

	// 等待足够长的时间以观察重试行为
	time.Sleep(time.Second * 15)

	// 获取队列统计信息
	stats, err := queueService.GetTotal("default")
	if err != nil {
		log.Fatalf("获取队列统计信息失败: %v", err)
	}

	fmt.Printf("队列统计信息: %+v\n", stats)

	// 关闭队列服务
	queueService.Stop()
}
