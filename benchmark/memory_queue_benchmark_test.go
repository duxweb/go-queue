package benchmark

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/drivers"
)

// 测试添加任务的性能
func BenchmarkMemoryQueueAdd(b *testing.B) {
	memQueue := drivers.NewMemoryQueue()
	queueName := "benchmark-add"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &queue.QueueItem{
			ID:          fmt.Sprintf("task-%d", i),
			WorkerName:  queueName,
			HandlerName: "benchmark-handler",
			Params:      []byte(`{"message":"benchmark task"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		_ = memQueue.Add(queueName, item)
	}
}

// 测试弹出任务的性能
func BenchmarkMemoryQueuePop(b *testing.B) {
	memQueue := drivers.NewMemoryQueue()
	queueName := "benchmark-pop"

	// 预先添加一些任务
	for i := 0; i < 10000; i++ {
		item := &queue.QueueItem{
			ID:          fmt.Sprintf("task-%d", i),
			WorkerName:  queueName,
			HandlerName: "benchmark-handler",
			Params:      []byte(`{"message":"benchmark task"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		_ = memQueue.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memQueue.Pop(queueName, 1)
	}
}

// 测试删除任务的性能
func BenchmarkMemoryQueueDel(b *testing.B) {
	memQueue := drivers.NewMemoryQueue()
	queueName := "benchmark-del"

	// 预先添加任务
	for i := 0; i < b.N; i++ {
		item := &queue.QueueItem{
			ID:          fmt.Sprintf("task-%d", i),
			WorkerName:  queueName,
			HandlerName: "benchmark-handler",
			Params:      []byte(`{"message":"benchmark task"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		_ = memQueue.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = memQueue.Del(queueName, fmt.Sprintf("task-%d", i))
	}
}

// 测试列出任务的性能
func BenchmarkMemoryQueueList(b *testing.B) {
	memQueue := drivers.NewMemoryQueue()
	queueName := "benchmark-list"

	// 预先添加一些任务
	for i := 0; i < 10000; i++ {
		item := &queue.QueueItem{
			ID:          fmt.Sprintf("task-%d", i),
			WorkerName:  queueName,
			HandlerName: "benchmark-handler",
			Params:      []byte(`{"message":"benchmark task"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		_ = memQueue.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memQueue.List(queueName, 1, 10)
	}
}

// 测试并发操作的性能
func BenchmarkMemoryQueueConcurrent(b *testing.B) {
	memQueue := drivers.NewMemoryQueue()
	queueName := "benchmark-concurrent"

	// 设置并发数
	concurrency := 10
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var wg sync.WaitGroup
			wg.Add(concurrency)

			for i := 0; i < concurrency; i++ {
				go func(idx int) {
					defer wg.Done()

					// 执行添加操作
					item := &queue.QueueItem{
						ID:          fmt.Sprintf("task-%d", idx),
						WorkerName:  queueName,
						HandlerName: "benchmark-handler",
						Params:      []byte(`{"message":"concurrent benchmark task"}`),
						CreatedAt:   time.Now(),
						Retried:     0,
					}
					_ = memQueue.Add(queueName, item)

					// 执行弹出操作
					_ = memQueue.Pop(queueName, 1)

					// 执行删除操作
					_ = memQueue.Del(queueName, fmt.Sprintf("task-%d", idx))
				}(i)
			}

			wg.Wait()
		}
	})
}
