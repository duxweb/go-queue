# Go-Queue

[![Go Report Card](https://goreportcard.com/badge/github.com/duxweb/go-queue)](https://goreportcard.com/report/github.com/duxweb/go-queue)
[![GoDoc](https://godoc.org/github.com/duxweb/go-queue?status.svg)](https://godoc.org/github.com/duxweb/go-queue)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

A high-performance, extensible queue library for Go, supporting multiple queue drivers, task retry mechanisms, and delayed execution.
一个高性能、可扩展的Go语言队列库，支持多种队列驱动、任务重试机制和延迟执行。

## Features / 特性

- **Multiple Queue Drivers**: Built-in memory queue driver, extensible to add other drivers
- **多种队列驱动**: 内置内存队列驱动，可扩展添加其他驱动

- **Concurrency Control**: Configure the number of workers for each queue
- **并发控制**: 为每个队列配置工作线程数量

- **Automatic Retry**: Configurable retry count and retry delay
- **自动重试**: 可配置重试次数和重试延迟

- **Delayed Execution**: Schedule tasks to run at a specific time
- **延迟执行**: 可以安排任务在指定时间执行

- **Timeout Control**: Set maximum execution time for tasks
- **超时控制**: 为任务设置最大执行时间

- **Statistics**: Track processed tasks, success rates, and processing times
- **统计数据**: 跟踪已处理任务、成功率和处理时间

- **Callbacks**: Custom callbacks for task success and failure events
- **回调函数**: 任务成功和失败事件的自定义回调

- **Graceful Shutdown**: Close queues safely without losing tasks
- **优雅关闭**: 安全关闭队列，不丢失任务

## Installation / 安装

```bash
go get -u github.com/duxweb/go-queue
```

## Quick Start / 快速开始

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
    // 创建上下文
    ctx := context.Background()

    // Create queue service config
    // 创建队列服务配置
    config := &goqueue.Config{
        Context: ctx,
    }

    // Create new queue service
    // 创建新的队列服务
    queueService, _ := goqueue.New(config)

    // Create memory queue instance
    // 创建内存队列实例
    memQueue := drivers.NewMemoryQueue()

    // Register queue driver
    // 注册队列驱动
    queueService.RegisterDriver("default", memQueue)

    // Configure worker
    // 配置工作器
    workerConfig := &goqueue.WorkerConfig{
        DeviceName: "default",
        Num:         5,                // Concurrent workers / 并发工作数量
        Interval:    time.Second * 1,  // Polling interval / 轮询间隔
        Retry:       3,                // Retry attempts / 重试次数
        RetryDelay:  time.Second * 5,  // Delay between retries / 重试间隔
        Timeout:     time.Minute,      // Task timeout / 任务超时时间
    }

    // Register worker
    // 注册工作器
    queueService.RegisterWorker("default", workerConfig)

    // Register task handler
    // 注册任务处理器
    queueService.RegisterHandler("example-handler", func(ctx context.Context, params []byte) error {
        fmt.Printf("Processing task: %s\n", string(params))
        return nil
    })

    // Add a task
    // 添加任务
    id, _ := queueService.Add("default", &goqueue.QueueConfig{
        HandlerName: "example-handler",
        Params:      []byte(`{"message":"This is a test task"}`),
    })

    fmt.Printf("Task added successfully: %s\n", id)
    // fmt.Printf("添加任务成功: %s\n", id)

    // Start queue processing
    // 启动队列处理
    queueService.Start()

    // Wait for tasks to complete
    // 等待任务完成
    time.Sleep(time.Second * 10)

    // Stop queue service
    // 停止队列服务
    queueService.Stop()
}
```

## Advanced Usage / 高级用法

### Delayed Tasks / 延迟任务

```go
// Add a delayed task (runs after 5 seconds)
// 添加延迟任务（5秒后执行）
id, _ := queueService.AddDelay("default", &goqueue.QueueDelayConfig{
    QueueConfig: goqueue.QueueConfig{
        HandlerName: "example-handler",
        Params:      []byte(`{"message":"This is a delayed task"}`),
    },
    Delay: time.Second * 5,
})
```

### Task Retry / 任务重试

Configure retry behavior:
配置重试行为：

```go
workerConfig := &goqueue.WorkerConfig{
    DeviceName: "default",
    Retry:       3,                // Maximum retry attempts / 最大重试次数
    RetryDelay:  time.Second * 2,  // Delay between retries / 重试间隔
}
```

### Success and Failure Callbacks / 成功和失败回调

```go
workerConfig := &goqueue.WorkerConfig{
    // ... other configs / 其他配置
    SuccessFunc: func(item *goqueue.QueueItem) {
        fmt.Printf("Task executed successfully: %s\n", item.ID)
        // fmt.Printf("任务执行成功: %s\n", item.ID)
    },
    FailFunc: func(item *goqueue.QueueItem, err error) {
        fmt.Printf("Task failed: %s, error: %v\n", item.ID, err)
        // fmt.Printf("任务执行失败: %s, 错误: %v\n", item.ID, err)
    },
}
```

## Benchmark Results / 性能基准测试结果

Performance benchmark results for memory queue and BBolt queue drivers (tested on Apple M4):
内存队列和 BBolt 队列驱动的性能基准测试结果（在 Apple M4 上测试）：

### Memory Queue Performance / 内存队列性能

| Operation / 操作 | Iterations / 迭代次数 | Time per Operation / 每次操作时间 | Memory per Operation / 每次操作内存使用 | Allocations per Operation / 每次操作内存分配次数 |
|-----------|------------|-------------------|---------------------|--------------------------|
| Add / 添加 | 2,958,324  | 367.4 ns/op       | 316 B/op            | 4 allocs/op             |
| Pop / 弹出 | 48,910     | 24,391 ns/op      | 65,828 B/op         | 33 allocs/op            |
| Delete / 删除 | 6,601,080  | 214.6 ns/op       | 0 B/op              | 0 allocs/op             |
| List / 列表 | 224,417,496| 5.308 ns/op       | 0 B/op              | 0 allocs/op             |
| Count / 计数 | 262,194,860| 4.578 ns/op      | 0 B/op              | 0 allocs/op             |

### BBolt Queue Performance / BBolt 队列性能

| Operation / 操作 | Iterations / 迭代次数 | Time per Operation / 每次操作时间 | Memory per Operation / 每次操作内存使用 | Allocations per Operation / 每次操作内存分配次数 |
|-----------|------------|-------------------|---------------------|--------------------------|
| Add / 添加 | 148        | 8,043,025 ns/op   | 28,349 B/op         | 61 allocs/op            |
| Pop / 弹出 | 147        | 8,057,802 ns/op   | 23,654 B/op         | 67 allocs/op            |
| Delete / 删除 | 148        | 7,979,694 ns/op   | 21,414 B/op         | 54 allocs/op            |
| List / 列表 | 82,171     | 13,347 ns/op      | 5,010 B/op          | 105 allocs/op           |
| Count / 计数 | -          | -                 | -                   | -                       |

### Performance Comparison / 性能对比

| Operation / 操作 | Memory Queue / 内存队列 | BBolt Queue / BBolt 队列 | Memory vs BBolt (X times faster) / 内存队列比 BBolt 队列快（倍数） |
|-----------|-------------|------------|----------------------------------|
| Add / 添加 | 367.4 ns/op  | 8,043,025 ns/op | ~21,892x / ~21,892倍                  |
| Pop / 弹出 | 24,391 ns/op | 8,057,802 ns/op | ~330x / ~330倍                     |
| Delete / 删除 | 214.6 ns/op  | 7,979,694 ns/op | ~37,184x / ~37,184倍                  |
| List / 列表 | 5.308 ns/op  | 13,347 ns/op    | ~2,514x / ~2,514倍                   |

These benchmarks show that:
这些基准测试结果表明：

- Memory queue is significantly faster than BBolt queue for all operations
- 内存队列在所有操作上都明显快于 BBolt 队列

- Memory queue operations are mostly sub-microsecond, while BBolt operations are in the millisecond range
- 内存队列操作大多是亚微秒级的，而 BBolt 操作则是毫秒级的

- The memory queue has much lower memory allocation overhead
- 内存队列有更低的内存分配开销

- BBolt queue provides persistence at the cost of performance
- BBolt 队列以性能为代价提供了数据持久化

- Choose between them based on your need for speed vs persistence
- 根据你对速度与持久性的需求来选择合适的队列驱动

Run your own benchmarks with:
运行自己的基准测试：

```bash
cd benchmark
go test -bench=. -benchmem
```

## Example Projects / 示例项目

Check the `example` directory for complete, runnable examples:
查看 `example` 目录以获取完整的、可运行的示例：

- `basic/` - Basic queue operations / 基本队列操作
- `retry/` - Task retry mechanism / 任务重试机制
- `multi_queues/` - Multiple parallel queues / 多个并行队列

## License / 许可证

MIT License
MIT 许可证
