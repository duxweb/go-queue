# Go-Queue

[![Go Report Card](https://goreportcard.com/badge/github.com/duxweb/go-queue)](https://goreportcard.com/report/github.com/duxweb/go-queue)
[![GoDoc](https://godoc.org/github.com/duxweb/go-queue?status.svg)](https://godoc.org/github.com/duxweb/go-queue)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

一个高性能、可扩展的Go语言队列库，支持多种队列驱动、任务重试机制和延迟执行。

A high-performance, extensible queue library for Go, supporting multiple queue drivers, task retry mechanisms, and delayed execution.

[English](#english) | [中文](#中文)

<a id="english"></a>
## Features

- **Multiple Queue Drivers**: Built-in memory queue driver, extensible to add other drivers
- **Concurrency Control**: Configure the number of workers for each queue
- **Automatic Retry**: Configurable retry count and retry delay
- **Delayed Execution**: Schedule tasks to run at a specific time
- **Timeout Control**: Set maximum execution time for tasks
- **Statistics**: Track processed tasks, success rates, and processing times
- **Callbacks**: Custom callbacks for task success and failure events
- **Graceful Shutdown**: Close queues safely without losing tasks

## Installation

```bash
go get -u github.com/duxweb/go-queue
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    goqueue "github.com/duxweb/go-queue"
    "github.com/duxweb/go-queue/drivers"
)

func main() {
    // Create context
    ctx := context.Background()

    // Create queue service config
    config := &goqueue.Config{
        Context: ctx,
    }

    // Create new queue service
    queueService, _ := goqueue.New(config)

    // Create memory queue instance
    memQueue := drivers.NewMemoryQueue()

    // Register queue driver
    queueService.RegisterDriver("default", memQueue)

    // Configure worker
    workerConfig := &goqueue.WorkerConfig{
        ServiceName: "default",
        Num:         5,                // Concurrent workers
        Interval:    time.Second * 1,  // Polling interval
        Retry:       3,                // Retry attempts
        RetryDelay:  time.Second * 5,  // Delay between retries
        Timeout:     time.Minute,      // Task timeout
    }

    // Register worker
    queueService.RegisterWorker("default", workerConfig)

    // Register task handler
    queueService.RegisterHandler("example-handler", func(ctx context.Context, params []byte) error {
        fmt.Printf("Processing task: %s\n", string(params))
        return nil
    })

    // Add a task
    id, _ := queueService.Add("default", &goqueue.QueueConfig{
        HandlerName: "example-handler",
        Params:      []byte(`{"message":"This is a test task"}`),
    })

    fmt.Printf("Task added successfully: %s\n", id)

    // Start queue processing
    queueService.Start()

    // Wait for tasks to complete
    time.Sleep(time.Second * 10)

    // Stop queue service
    queueService.Stop()
}
```

## Advanced Usage

### Delayed Tasks

```go
// Add a delayed task (runs after 5 seconds)
id, _ := queueService.AddDelay("default", &goqueue.QueueDelayConfig{
    QueueConfig: goqueue.QueueConfig{
        HandlerName: "example-handler",
        Params:      []byte(`{"message":"This is a delayed task"}`),
    },
    Delay: time.Second * 5,
})
```

### Task Retry

Configure retry behavior:

```go
workerConfig := &goqueue.WorkerConfig{
    ServiceName: "default",
    Retry:       3,                // Maximum retry attempts
    RetryDelay:  time.Second * 2,  // Delay between retries
}
```

### Success and Failure Callbacks

```go
workerConfig := &goqueue.WorkerConfig{
    // ... other configs
    SuccessFunc: func(item *goqueue.QueueItem) {
        fmt.Printf("Task executed successfully: %s\n", item.ID)
    },
    FailFunc: func(item *goqueue.QueueItem, err error) {
        fmt.Printf("Task failed: %s, error: %v\n", item.ID, err)
    },
}
```

## Example Projects

Check the `example` directory for complete, runnable examples:

- `basic/` - Basic queue operations
- `retry/` - Task retry mechanism
- `multi_queues/` - Multiple parallel queues

## License

MIT License

<a id="中文"></a>
## 特性

- **多种队列驱动**: 内置内存队列驱动，可扩展添加其他驱动
- **并发控制**: 为每个队列配置工作线程数量
- **自动重试**: 可配置重试次数和重试延迟
- **延迟执行**: 可以安排任务在指定时间执行
- **超时控制**: 为任务设置最大执行时间
- **统计数据**: 跟踪已处理任务、成功率和处理时间
- **回调函数**: 任务成功和失败事件的自定义回调
- **优雅关闭**: 安全关闭队列，不丢失任务

## 安装

```bash
go get -u github.com/duxweb/go-queue
```

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "time"

    goqueue "github.com/duxweb/go-queue"
    "github.com/duxweb/go-queue/drivers"
)

func main() {
    // 创建上下文
    ctx := context.Background()

    // 创建队列服务配置
    config := &goqueue.Config{
        Context: ctx,
    }

    // 创建新的队列服务
    queueService, _ := goqueue.New(config)

    // 创建内存队列实例
    memQueue := drivers.NewMemoryQueue()

    // 注册队列驱动
    queueService.RegisterDriver("default", memQueue)

    // 配置工作器
    workerConfig := &goqueue.WorkerConfig{
        ServiceName: "default",
        Num:         5,               // 并发工作数量
        Interval:    time.Second * 1, // 轮询间隔
        Retry:       3,               // 重试次数
        RetryDelay:  time.Second * 5, // 重试间隔
        Timeout:     time.Minute,     // 任务超时时间
    }

    // 注册工作器
    queueService.RegisterWorker("default", workerConfig)

    // 注册任务处理器
    queueService.RegisterHandler("example-handler", func(ctx context.Context, params []byte) error {
        fmt.Printf("处理任务: %s\n", string(params))
        return nil
    })

    // 添加任务
    id, _ := queueService.Add("default", &goqueue.QueueConfig{
        HandlerName: "example-handler",
        Params:      []byte(`{"message":"这是一个测试任务"}`),
    })

    fmt.Printf("添加任务成功: %s\n", id)

    // 启动队列处理
    queueService.Start()

    // 等待任务完成
    time.Sleep(time.Second * 10)

    // 停止队列服务
    queueService.Stop()
}
```

## 高级用法

### 延迟任务

```go
// 添加延迟任务（5秒后执行）
id, _ := queueService.AddDelay("default", &goqueue.QueueDelayConfig{
    QueueConfig: goqueue.QueueConfig{
        HandlerName: "example-handler",
        Params:      []byte(`{"message":"这是一个延迟任务"}`),
    },
    Delay: time.Second * 5,
})
```

### 任务重试

配置重试行为：

```go
workerConfig := &goqueue.WorkerConfig{
    ServiceName: "default",
    Retry:       3,               // 最大重试次数
    RetryDelay:  time.Second * 2, // 重试间隔
}
```

### 成功和失败回调

```go
workerConfig := &goqueue.WorkerConfig{
    // ... 其他配置
    SuccessFunc: func(item *goqueue.QueueItem) {
        fmt.Printf("任务执行成功: %s\n", item.ID)
    },
    FailFunc: func(item *goqueue.QueueItem, err error) {
        fmt.Printf("任务执行失败: %s, 错误: %v\n", item.ID, err)
    },
}
```

## 示例项目

查看 `example` 目录以获取完整的、可运行的示例：

- `basic/` - 基本队列操作
- `retry/` - 任务重试机制
- `multi_queues/` - 多个并行队列

## 许可证

MIT 许可证
