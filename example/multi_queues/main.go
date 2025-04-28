package main

import (
	"context"
	"fmt"
	"log"
	"time"

	goqueue "github.com/duxweb/go-queue"
	workerdriver "github.com/duxweb/go-queue/drivers"
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

	// 创建两个内存队列实例
	highPriorityQueue := workerdriver.NewMemoryQueue()
	lowPriorityQueue := workerdriver.NewMemoryQueue()

	// 注册队列服务
	queueService.RegisterDriver("high-priority", highPriorityQueue)
	queueService.RegisterDriver("low-priority", lowPriorityQueue)

	// 创建高优先级工作队列配置
	highPriorityConfig := &goqueue.WorkerConfig{
		ServiceName: "high-priority",
		Num:         5,                // 并发工作数量 - 高优先级队列分配更多资源
		Interval:    time.Second * 1,  // 轮询间隔 - 高优先级队列更频繁地检查
		Retry:       3,                // 重试次数
		RetryDelay:  time.Second * 2,  // 重试间隔
		Timeout:     time.Second * 30, // 任务超时时间
		SuccessFunc: func(item *goqueue.QueueItem) {
			fmt.Printf("[高优先级] 任务成功执行: %s\n", string(item.Params))
		},
	}

	// 创建低优先级工作队列配置
	lowPriorityConfig := &goqueue.WorkerConfig{
		ServiceName: "low-priority",
		Num:         2,                // 并发工作数量 - 低优先级队列分配较少资源
		Interval:    time.Second * 3,  // 轮询间隔 - 低优先级队列检查频率较低
		Retry:       2,                // 重试次数
		RetryDelay:  time.Second * 5,  // 重试间隔
		Timeout:     time.Second * 60, // 任务超时时间
		SuccessFunc: func(item *goqueue.QueueItem) {
			fmt.Printf("[低优先级] 任务成功执行: %s\n", string(item.Params))
		},
	}

	// 注册工作器
	err = queueService.RegisterWorker("high-priority", highPriorityConfig)
	if err != nil {
		log.Fatalf("注册高优先级工作器失败: %v", err)
	}

	err = queueService.RegisterWorker("low-priority", lowPriorityConfig)
	if err != nil {
		log.Fatalf("注册低优先级工作器失败: %v", err)
	}

	// 注册任务处理器
	queueService.RegisterHandler("fast-task", func(ctx context.Context, params []byte) error {
		fmt.Printf("处理快速任务: %s\n", string(params))
		// 模拟快速处理
		time.Sleep(time.Millisecond * 100)
		return nil
	})

	queueService.RegisterHandler("slow-task", func(ctx context.Context, params []byte) error {
		fmt.Printf("处理慢速任务: %s\n", string(params))
		// 模拟较长时间处理
		time.Sleep(time.Second * 1)
		return nil
	})

	// 添加高优先级任务
	for i := 1; i <= 10; i++ {
		id, err := queueService.Add("high-priority", &goqueue.QueueConfig{
			HandlerName: "fast-task",
			Params:      []byte(fmt.Sprintf(`{"message":"高优先级任务 #%d"}`, i)),
		})
		if err != nil {
			log.Fatalf("添加高优先级任务失败: %v", err)
		}

		fmt.Printf("添加高优先级任务成功: %s\n", id)
	}

	// 添加低优先级任务
	for i := 1; i <= 5; i++ {
		id, err := queueService.Add("low-priority", &goqueue.QueueConfig{
			HandlerName: "slow-task",
			Params:      []byte(fmt.Sprintf(`{"message":"低优先级任务 #%d"}`, i)),
		})
		if err != nil {
			log.Fatalf("添加低优先级任务失败: %v", err)
		}

		fmt.Printf("添加低优先级任务成功: %s\n", id)
	}

	// 启动队列处理
	if err := queueService.Start(); err != nil {
		log.Fatalf("启动队列服务失败: %v", err)
	}

	// 等待足够长的时间以观察多队列处理
	time.Sleep(time.Second * 10)

	// 获取队列统计信息
	highStats, err := queueService.GetTotal("high-priority")
	if err != nil {
		log.Fatalf("获取高优先级队列统计信息失败: %v", err)
	}

	lowStats, err := queueService.GetTotal("low-priority")
	if err != nil {
		log.Fatalf("获取低优先级队列统计信息失败: %v", err)
	}

	fmt.Printf("高优先级队列统计: %+v\n", highStats)
	fmt.Printf("低优先级队列统计: %+v\n", lowStats)

	// 关闭队列服务
	queueService.Stop()
}
