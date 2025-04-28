package main

import (
	"context"
	"fmt"
	"log"
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
		Num:         5,               // 并发工作数量
		Interval:    time.Second * 1, // 轮询间隔
		Retry:       3,               // 重试次数
		RetryDelay:  time.Second * 5, // 重试间隔
		Timeout:     time.Minute,     // 任务超时时间
	}

	// 注册工作器
	err = queueService.RegisterWorker("default", workerConfig)
	if err != nil {
		log.Fatalf("注册工作器失败: %v", err)
	}

	// 注册任务处理器
	queueService.RegisterHandler("example-handler", func(ctx context.Context, params []byte) error {
		fmt.Printf("处理任务: %s\n", string(params))
		return nil
	})

	// 添加任务
	id, err := queueService.Add("default", &goqueue.QueueConfig{
		HandlerName: "example-handler",
		Params:      []byte(`{"message":"这是一个测试任务"}`),
	})
	if err != nil {
		log.Fatalf("添加任务失败: %v", err)
	}

	fmt.Printf("添加任务成功: %s\n", id)

	// 添加延迟任务
	id, err = queueService.AddDelay("default", &goqueue.QueueDelayConfig{
		QueueConfig: goqueue.QueueConfig{
			HandlerName: "example-handler",
			Params:      []byte(`{"message":"这是一个延迟任务"}`),
		},
		Delay: time.Second * 5, // 5秒后执行
	})
	if err != nil {
		log.Fatalf("添加延迟任务失败: %v", err)
	}

	fmt.Printf("添加延迟任务成功: %s\n", id)

	// 启动队列处理
	if err := queueService.Start(); err != nil {
		log.Fatalf("启动队列服务失败: %v", err)
	}

	// 等待所有任务完成
	time.Sleep(time.Second * 10)

	// 获取队列统计信息
	stats, err := queueService.GetTotal("default")
	if err != nil {
		log.Fatalf("获取队列统计信息失败: %v", err)
	}

	fmt.Printf("队列统计信息: %+v\n", stats)

	// 关闭队列服务
	queueService.Stop()
}
