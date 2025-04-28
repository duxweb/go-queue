package main

import (
	"context"
	"fmt"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/drivers/memory"
	"github.com/duxweb/go-queue/drivers/sqlite"
)

// 示例函数：使用内存驱动
func memoryExample() {
	// 创建内存队列实例
	memQueue := memory.New()

	// 使用队列
	useQueueDriver(memQueue, "Memory")
}

// 示例函数：使用SQLite驱动
func sqliteExample() {
	// 创建SQLite队列实例
	options := &sqlite.SQLiteOptions{
		DBPath: "sqlite-queue.db",
	}

	sqliteQueue, err := sqlite.New(options)
	if err != nil {
		panic(err)
	}

	// 使用队列
	useQueueDriver(sqliteQueue, "SQLite")
}

// 通用的队列使用函数
func useQueueDriver(driver queue.QueueDriver, driverName string) {
	// 创建队列配置
	config := &queue.Config{
		Context: context.Background(),
	}

	// 创建队列服务
	queueService, err := queue.New(config)
	if err != nil {
		panic(err)
	}

	// 注册驱动
	queueService.RegisterDriver("default", driver)

	// 注册工作器
	err = queueService.RegisterWorker("test-worker", &queue.WorkerConfig{
		DeviceName: "default",
		Num:        1,
		Interval:   time.Second * 1,
	})
	if err != nil {
		panic(err)
	}

	// 注册处理器
	queueService.RegisterHandler("test-handler", func(ctx context.Context, params []byte) error {
		fmt.Printf("处理任务: %s\n", string(params))
		return nil
	})

	// 添加队列项
	id, err := queueService.Add("test-worker", &queue.QueueConfig{
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s Queue: Added item with ID: %s\n", driverName, id)

	// 获取队列数量
	count, err := queueService.Count("test-worker")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s Queue Item Count: %d\n", driverName, count)

	// 关闭队列
	queueService.Stop()
}

func main() {
	// 运行内存队列示例
	memoryExample()

	// 运行SQLite队列示例
	sqliteExample()
}
