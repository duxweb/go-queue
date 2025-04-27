package benchmark

import (
	"fmt"
	"testing"
	"time"

	"github.com/duxweb/go-queue/internal/queue"
	"github.com/duxweb/go-queue/pkg/queue/memory"
)

func BenchmarkMemoryQueue_Add(b *testing.B) {
	q := memory.NewMemoryQueue()
	queueName := "benchmark-queue"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("task-%d", i)
		item := &queue.QueueItem{
			ID:          id,
			QueueName:   queueName,
			HandlerName: "benchmark-handler",
			Params:      []byte(`{"test":"data"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		_ = q.Add(queueName, item)
	}
}

func BenchmarkMemoryQueue_Pop(b *testing.B) {
	q := memory.NewMemoryQueue()
	queueName := "benchmark-queue"

	// 预先添加1000个任务
	// Pre-add 1000 tasks
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("task-%d", i)
		item := &queue.QueueItem{
			ID:          id,
			QueueName:   queueName,
			HandlerName: "benchmark-handler",
			Params:      []byte(`{"test":"data"}`),
			CreatedAt:   time.Now().Add(-time.Second), // 已经可以执行的任务 | Tasks ready for execution
			Retried:     0,
		}
		_ = q.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Pop(queueName, 10)
	}
}

func BenchmarkMemoryQueue_Del(b *testing.B) {
	q := memory.NewMemoryQueue()
	queueName := "benchmark-queue"

	// 创建一个预先分配好的任务ID
	// Create an array of pre-allocated task IDs
	taskIDs := make([]string, b.N)

	// 预先添加N个任务
	// Pre-add N tasks
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("task-%d", i)
		taskIDs[i] = id
		item := &queue.QueueItem{
			ID:          id,
			QueueName:   queueName,
			HandlerName: "benchmark-handler",
			Params:      []byte(`{"test":"data"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		_ = q.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Del(queueName, taskIDs[i])
	}
}

func BenchmarkMemoryQueue_List(b *testing.B) {
	q := memory.NewMemoryQueue()
	queueName := "benchmark-queue"

	// 预先添加1000个任务
	// Pre-add 1000 tasks
	for i := 0; i < 1000; i++ {
		id := fmt.Sprintf("task-%d", i)
		item := &queue.QueueItem{
			ID:          id,
			QueueName:   queueName,
			HandlerName: "benchmark-handler",
			Params:      []byte(`{"test":"data"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		_ = q.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 分页获取10个任务
		// Get page with 10 items
		q.List(queueName, (i%100)+1, 10)
	}
}

func BenchmarkMemoryQueue_Concurrent(b *testing.B) {
	q := memory.NewMemoryQueue()
	queueName := "benchmark-concurrent"

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			id := fmt.Sprintf("task-%d", i)
			item := &queue.QueueItem{
				ID:          id,
				QueueName:   queueName,
				HandlerName: "benchmark-handler",
				Params:      []byte(`{"test":"data"}`),
				CreatedAt:   time.Now(),
				Retried:     0,
			}
			_ = q.Add(queueName, item)
			_ = q.Pop(queueName, 1)
			_ = q.Del(queueName, id)
			i++
		}
	})
}
