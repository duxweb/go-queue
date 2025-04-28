package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	goqueue "github.com/duxweb/go-queue"
	boltdriver "github.com/duxweb/go-queue/drivers/bbolt"
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

	// 获取临时文件路径
	dbPath := "./bbolt-queue.db"
	// 确保退出时清理数据库文件
	defer os.Remove(dbPath)

	// 创建BBolt队列实例
	boltQueue, err := boltdriver.New(dbPath, nil)
	if err != nil {
		log.Fatalf("创建BBolt队列失败: %v", err)
	}
	defer boltQueue.Close()

	// 注册队列服务
	queueService.RegisterDriver("bbolt-queue", boltQueue)

	// 创建工作队列配置
	workerConfig := &goqueue.WorkerConfig{
		DeviceName: "bbolt-queue",
		Num:        3,                // 并发工作数量
		Interval:   time.Second * 1,  // 轮询间隔
		Retry:      2,                // 重试次数
		RetryDelay: time.Second * 2,  // 重试间隔
		Timeout:    time.Second * 30, // 任务超时时间
		SuccessFunc: func(item *goqueue.QueueItem) {
			fmt.Printf("任务执行成功: %s, 已重试: %d次\n", item.ID, item.Retried)
		},
		FailFunc: func(item *goqueue.QueueItem, err error) {
			fmt.Printf("任务执行失败: %s, 错误: %v\n", item.ID, err)
		},
	}

	// 注册工作器
	err = queueService.RegisterWorker("bbolt-worker", workerConfig)
	if err != nil {
		log.Fatalf("注册工作器失败: %v", err)
	}

	// 注册任务处理器
	queueService.RegisterHandler("bbolt-handler", func(ctx context.Context, params []byte) error {
		fmt.Printf("处理任务: %s\n", string(params))
		time.Sleep(time.Millisecond * 500) // 模拟处理时间
		return nil
	})

	// 注册一个会失败的任务处理器（用于测试重试）
	var attempt int
	queueService.RegisterHandler("retry-handler", func(ctx context.Context, params []byte) error {
		fmt.Printf("处理重试任务: %s, 尝试次数: %d\n", string(params), attempt)
		if attempt == 0 {
			attempt++
			return fmt.Errorf("模拟任务失败")
		}
		return nil
	})

	fmt.Println("=== 添加常规任务 ===")
	// 添加一些任务
	for i := 1; i <= 5; i++ {
		id, err := queueService.Add("bbolt-worker", &goqueue.QueueConfig{
			HandlerName: "bbolt-handler",
			Params:      []byte(fmt.Sprintf(`{"message":"BBolt任务 #%d"}`, i)),
		})
		if err != nil {
			log.Fatalf("添加任务失败: %v", err)
		}
		fmt.Printf("添加任务成功: %s\n", id)
	}

	fmt.Println("\n=== 添加延迟任务 ===")
	// 添加延迟任务
	id, err := queueService.AddDelay("bbolt-worker", &goqueue.QueueDelayConfig{
		QueueConfig: goqueue.QueueConfig{
			HandlerName: "bbolt-handler",
			Params:      []byte(`{"message":"BBolt延迟任务"}`),
		},
		Delay: time.Second * 3, // 3秒后执行
	})
	if err != nil {
		log.Fatalf("添加延迟任务失败: %v", err)
	}
	fmt.Printf("添加延迟任务成功: %s\n", id)

	fmt.Println("\n=== 添加重试任务 ===")
	// 添加一个会失败然后重试的任务
	id, err = queueService.Add("bbolt-worker", &goqueue.QueueConfig{
		HandlerName: "retry-handler",
		Params:      []byte(`{"message":"BBolt重试任务"}`),
	})
	if err != nil {
		log.Fatalf("添加重试任务失败: %v", err)
	}
	fmt.Printf("添加重试任务成功: %s\n", id)

	// 启动队列处理
	fmt.Println("\n=== 启动队列处理 ===")
	if err := queueService.Start(); err != nil {
		log.Fatalf("启动队列服务失败: %v", err)
	}

	// 等待任务执行完成
	time.Sleep(time.Second * 6)

	// 获取队列统计信息
	stats, err := queueService.GetTotal("bbolt-worker")
	if err != nil {
		log.Fatalf("获取队列统计信息失败: %v", err)
	}

	fmt.Println("\n=== 队列统计信息 ===")
	fmt.Printf("已处理任务: %d\n", stats["processed"])
	fmt.Printf("成功任务: %d\n", stats["success"])
	fmt.Printf("失败任务: %d\n", stats["failed"])
	fmt.Printf("重试次数: %d\n", stats["retry"])
	fmt.Printf("平均处理时间: %s\n", stats["avg_time"])
	fmt.Printf("成功率: %.2f%%\n", stats["success_rate"].(float64)*100)

	// 关闭队列服务
	fmt.Println("\n=== 关闭队列服务 ===")
	if err := queueService.Stop(); err != nil {
		log.Fatalf("停止队列服务失败: %v", err)
	}
}
