<h1 align="center">Go-Queue</h1>
<p align="center">High-performance, Extensible Native Go Queue Framework</p>

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

## About

Go-Queue is a high-performance, extensible native Go queue framework designed for efficient task processing. Unlike traditional message queues focused on message passing, Go-Queue emphasizes precise task execution control with support for immediate and delayed tasks, concurrent processing, and sophisticated error handling. With a flexible storage backend architecture, it currently offers an optimized memory implementation while planning support for Redis, MySQL, PostgreSQL, and SQLite.

## Features

- Support for immediate and delayed tasks
- Task retry and error handling
- Parallel processing
- Thread-safe design
- High-performance memory queue implementation
- Extensible storage backends

## Installation

```bash
go get github.com/duxweb/go-queue
```

## Quick Start

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
	err = queueService.AddDelay("default", &queue.QueueDelayConfig{
		QueueConfig: queue.QueueConfig{
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

## Core API Reference

### Create Queue Service

```go
// Create context
ctx := context.Background()

// Create queue service configuration
config := &goqueue.Config{
    Context: ctx,
}

// Create a new queue service instance
queueService, err := goqueue.New(config)
```

### Register Services and Workers

```go
// Register a queue service
queueService.RegisterService("queue-name", queueServiceImplementation)

// Register a worker
queueService.RegisterWorker("queue-name", workerInstance)

// Register a task handler
queueService.RegisterHandler("handler-name", func(ctx context.Context, params []byte) error {
    // Process the task
    return nil
})
```

### Adding Tasks

```go
// Add immediate task
err = queueService.Add("queue-name", &queue.QueueConfig{
    HandlerName: "handler-name",
    Params:      []byte(`{"key":"value"}`),
})

// Add delayed task
err = queueService.AddDelay("queue-name", &queue.QueueDelayConfig{
    QueueConfig: queue.QueueConfig{
        HandlerName: "handler-name",
        Params:      []byte(`{"key":"value"}`),
    },
    Delay: time.Minute * 5, // Execute after 5 minutes
})
```

### Task Management

```go
// List tasks (paginated)
items, err := queueService.List("queue-name", 1, 10)

// Count tasks
count, err := queueService.Count("queue-name")

// Delete task
err = queueService.Del("queue-name", "task-id")

// Get queue statistics
stats, err := queueService.GetTotal("queue-name")
```

### Starting Queue Processing

```go
// Start all registered workers
err := queueService.Start()
```

## Testing

Run all tests:
```bash
go test ./...
```

Run benchmark tests:
```bash
go test -bench=. ./test/benchmark
```

## Performance

Benchmark results on Apple M4 processor:

| Operation       | Performance (ns/op) | Operations/second |
|-----------------|------------|-----------|
| Add Task    | 325.6 ns/op | ~3,070,000 |
| Pop Task    | 8.0 ns/op   | ~125,000,000 |
| Delete Task | 180.0 ns/op | ~5,560,000 |
| List Tasks  | 5.3 ns/op   | ~188,680,000 |
| Concurrent Operations | 614.8 ns/op | ~1,630,000 |

## Roadmap

Future implementations planned:
- Redis queue implementation
- MySQL queue implementation
- PostgreSQL queue implementation
- SQLite queue implementation

## Contributing

Contributions, issues, and feature requests are welcome! Feel free to check the [issues page](https://github.com/duxweb/go-queue/issues).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.