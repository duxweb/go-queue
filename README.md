# Go-Queue - High-performance, Extensible Native Go Queue Framework

[English](README.md) | [中文](README.Zh-CN.md)

Go-Queue is a high-performance, extensible native Go queue framework that supports multiple storage backends, suitable for task processing, delayed execution, and other scenarios.

## Design Philosophy

Go-Queue's design philosophy is to provide a native Go, high-performance queue framework with the following features:

- **Extensible Storage Backends**: Through a unified interface design, it supports extending any type of database as a queue storage backend, such as memory, Redis, SQLite, MySQL, PostgreSQL, etc.
- **Worker Pattern**: Uses a worker pattern to process queue tasks, supporting concurrent processing
- **Delayed Task Support**: Built-in support for delayed task execution
- **Elegant Error Handling**: Supports task retry, timeout control, and other mechanisms
- **Simple and Easy-to-use API**: Provides intuitive APIs, easy to integrate and use

The main difference between Go-Queue and message queues (like Kafka) is that Go-Queue focuses more on task processing rather than message passing models, suitable for business scenarios that require precise control over task execution.

## Features

- Support for immediate and delayed tasks
- Support for task retry and error handling
- Support for parallel processing
- Thread-safe design
- High-performance memory queue implementation
- Extensible storage backends

## Installation

```bash
go get github.com/duxweb/go-queue
```

## Usage

### Quick Start

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
	// Create context
	ctx := context.Background()

	// Create queue service configuration
	config := &goqueue.Config{
		Context: ctx,
	}

	// Create a new queue service instance
	queueService, err := goqueue.New(config)
	if err != nil {
		log.Fatalf("Failed to create queue service: %v", err)
	}

	// Create memory queue instance
	memQueue := memory.NewMemoryQueue()

	// Register memory queue service
	queueService.RegisterService("default", memQueue)

	// Create worker configuration
	workerConfig := &internalWorker.WorkerConfig{
		ServiceName: "default",
		Num:         5,                // Number of concurrent workers
		Interval:    time.Second * 1,  // Polling interval
		Retry:       3,                // Number of retries
		RetryDelay:  time.Second * 5,  // Retry delay
		Timeout:     time.Minute,      // Task timeout
	}

	// Create worker instance
	workerInstance := pkgWorker.NewWorker(workerConfig)

	// Register worker
	queueService.RegisterWorker("default", workerInstance)

	// Register task handler
	queueService.RegisterHandler("example-handler", func(ctx context.Context, params []byte) error {
		fmt.Printf("Processing task: %s\n", string(params))
		return nil
	})

	// Add task
	err = queueService.Add("default", &queue.QueueConfig{
		HandlerName: "example-handler",
		Params:      []byte(`{"message":"This is a test task"}`),
	})
	if err != nil {
		log.Fatalf("Failed to add task: %v", err)
	}

	// Add delayed task
	err = queueService.AddDelay("default", &internalQueue.QueueDelayConfig{
		QueueConfig: internalQueue.QueueConfig{
			HandlerName: "example-handler",
			Params:      []byte(`{"message":"This is a delayed task"}`),
		},
		Delay: time.Second * 5, // Execute after 5 seconds
	})
	if err != nil {
		log.Fatalf("Failed to add delayed task: %v", err)
	}

	// Start queue processing
	if err := queueService.Start(); err != nil {
		log.Fatalf("Failed to start queue service: %v", err)
	}

	// Keep the program running
	select {}
}
```

### Memory Queue API

#### Create Memory Queue

```go
memQueue := memory.NewMemoryQueue()
```

#### Pop Tasks from Queue

```go
// Pop up to 10 tasks from the queue
items := memQueue.Pop("queue-name", 10)
```

#### Query Queue Tasks

```go
// Get paginated queue tasks
items, count, err := queueService.List("queue-name", 1, 10)
```

#### Get Queue Statistics

```go
// Get queue statistics
stats, err := queueService.GetTotal("queue-name")
if err != nil {
	log.Fatalf("Failed to get queue statistics: %v", err)
}
fmt.Printf("Total processed tasks: %v\n", stats["processed"])
fmt.Printf("Successful tasks: %v\n", stats["success"])
fmt.Printf("Failed tasks: %v\n", stats["failed"])
```

## Custom Storage Backend

You can create a custom storage backend by implementing the `queue.QueueService` interface:

```go
type QueueService interface {
	// Get queue data
	Get(queueName string, num int) []*QueueItem
	// Add queue data
	Add(queueName string, queue *QueueItem) error
	// Delete queue data
	Del(queueName string, id string) error
	// Get queue data count
	Count(queueName string) int
	// Get queue list
	List(queueName string, page int, limit int) []*QueueItem
}
```

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Verbose Tests

```bash
go test -v ./...
```

### Run Benchmark Tests

```bash
go test -bench=. ./test/benchmark
```

### Run Tests with Cache Disabled

```bash
go test -count=1 ./...
```

## Performance Data

Benchmark results on Apple M4 processor:

| Operation       | Performance (ns/op) | Operations/second |
|-----------------|------------|-----------|
| Add Task (Add)    | 325.6 ns/op | ~3,070,000 |
| Get Task (Pop)    | 8.0 ns/op   | ~125,000,000 |
| Delete Task (Del) | 180.0 ns/op | ~5,560,000 |
| List Tasks (List) | 5.3 ns/op   | ~188,680,000 |
| Concurrent Operations | 614.8 ns/op | ~1,630,000 |

> Note: The Del operation of the memory queue has been optimized, with a performance improvement of approximately 340 times (from ~62814 ns/op to ~180 ns/op).

## Notes

- Memory queue does not support persistence, tasks in the queue will be lost after program restart
- For production environments, it is recommended to implement and use a persistent queue storage backend

## Extension Plans

- Redis queue implementation
- MySQL queue implementation
- PostgreSQL queue implementation
- SQLite queue implementation

## License

This project is licensed under the MIT License - see the LICENSE file for details.