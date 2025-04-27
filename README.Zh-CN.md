<h1 align="center">Go-Queue</h1>
<p align="center">高性能、可扩展的Go原生队列框架</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/duxweb/go-queue" target="_blank">
    <img src="https://img.shields.io/github/go-mod/go-version/duxweb/go-queue" alt="Go Version">
  </a>
  <a href="https://github.com/duxweb/go-queue/blob/main/LICENSE" target="_blank">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT">
  </a>
  <img src="https://img.shields.io/badge/coverage-90%25-green" alt="Coverage Status">
</p>

<p align="center">
  <a href="README.md">English</a> |
  <a href="README.Zh-CN.md">中文</a>
</p>

## 关于

Go-Queue 是一个高性能、可扩展的 Go 原生队列框架，专为高效任务处理而设计。与专注于消息传递的传统消息队列不同，Go-Queue 注重精确的任务执行控制，支持即时和延迟任务、并发处理和完善的错误处理机制。通过灵活的存储后端架构，目前提供了优化的内存实现，并计划支持 Redis、MySQL、PostgreSQL 和 SQLite。

## 特性

- 支持即时任务和延迟任务
- 任务重试和错误处理
- 并行处理支持
- 线程安全设计
- 高性能内存队列实现
- 可扩展的存储后端

## 安装

```bash
go get github.com/duxweb/go-queue
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/internal/queue"
	internalWorker "github.com/duxweb/go-queue/internal/worker"
	"github.com/duxweb/go-queue/pkg/queue/memory"
	pkgWorker "github.com/duxweb/go-queue/pkg/worker"
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
	memQueue := memory.NewMemoryQueue()

	// 注册内存队列服务
	queueService.RegisterService("default", memQueue)

	// 创建工作队列
	workerConfig := &internalWorker.WorkerConfig{
		ServiceName: "default",
		Num:         5,                // 并发工作数量
		Interval:    time.Second * 1,  // 轮询间隔
		Retry:       3,                // 重试次数
		RetryDelay:  time.Second * 5,  // 重试间隔
		Timeout:     time.Minute,      // 任务超时时间
	}

	// 创建工作器实例
	workerInstance := pkgWorker.NewWorker(workerConfig)

	// 注册工作器
	queueService.RegisterWorker("default", workerInstance)

	// 注册任务处理器
	queueService.RegisterHandler("example-handler", func(ctx context.Context, params []byte) error {
		fmt.Printf("处理任务: %s\n", string(params))
		return nil
	})

	// 添加任务
	err = queueService.Add("default", &queue.QueueConfig{
		HandlerName: "example-handler",
		Params:      []byte(`{"message":"这是一个测试任务"}`),
	})
	if err != nil {
		log.Fatalf("添加任务失败: %v", err)
	}

	// 添加延迟任务
	err = queueService.AddDelay("default", &internalQueue.QueueDelayConfig{
		QueueConfig: internalQueue.QueueConfig{
			HandlerName: "example-handler",
			Params:      []byte(`{"message":"这是一个延迟任务"}`),
		},
		Delay: time.Second * 5, // 5秒后执行
	})
	if err != nil {
		log.Fatalf("添加延迟任务失败: %v", err)
	}

	// 启动队列处理
	if err := queueService.Start(); err != nil {
		log.Fatalf("启动队列服务失败: %v", err)
	}

	// 保持程序运行
	select {}
}
```

## API参考

### 内存队列API

#### 创建内存队列

```go
memQueue := memory.NewMemoryQueue()
```

#### 弹出队列中的任务

```go
// 从队列中弹出最多10个任务
items := memQueue.Pop("queue-name", 10)
```

#### 查询队列任务

```go
// 分页获取队列任务
items, count, err := queueService.List("queue-name", 1, 10)
```

#### 获取队列统计

```go
// 获取队列统计信息
stats, err := queueService.GetTotal("queue-name")
if err != nil {
	log.Fatalf("获取队列统计失败: %v", err)
}
fmt.Printf("处理的任务总数: %v\n", stats["processed"])
fmt.Printf("成功的任务数: %v\n", stats["success"])
fmt.Printf("失败的任务数: %v\n", stats["failed"])
```

## 扩展框架

### 自定义存储后端

可以通过实现 `queue.QueueService` 接口来创建自定义的存储后端：

```go
type QueueService interface {
	// 弹出队列数据
	Pop(queueName string, num int) []*QueueItem
	// 添加队列数据
	Add(queueName string, queue *QueueItem) error
	// 删除队列数据
	Del(queueName string, id string) error
	// 获取队列数据数量
	Count(queueName string) int
	// 获取队列列表
	List(queueName string, page int, limit int) []*QueueItem
}
```

## 测试

运行所有测试:
```bash
go test ./...
```

运行详细测试:
```bash
go test -v ./...
```

运行基准测试:
```bash
go test -bench=. ./test/benchmark
```

禁用缓存运行测试:
```bash
go test -count=1 ./...
```

## 性能

在Apple M4处理器上的基准测试结果：

| 操作            | 性能 (ns/op) | 每秒操作数 |
|-----------------|------------|-----------|
| 添加任务        | 325.6 ns/op | 约3,070,000 |
| 获取任务        | 8.0 ns/op   | 约125,000,000 |
| 删除任务        | 180.0 ns/op | 约5,560,000 |
| 列表任务        | 5.3 ns/op   | 约188,680,000 |
| 并发操作        | 614.8 ns/op | 约1,630,000 |

## 注意事项与限制

- 内存队列不支持持久化，程序重启后队列中的任务将丢失
- 对于生产环境，建议实现和使用持久化的队列存储后端

## 路线图

计划未来实现:
- Redis队列实现
- MySQL队列实现
- PostgreSQL队列实现
- SQLite队列实现

## 贡献

欢迎提供贡献、问题和功能请求！请随时查看[问题页面](https://github.com/duxweb/go-queue/issues)。

## 许可证

本项目采用 MIT 许可证 - 详情请参阅 [LICENSE](LICENSE) 文件。