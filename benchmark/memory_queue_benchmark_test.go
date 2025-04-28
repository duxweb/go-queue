package benchmark

import (
	"fmt"
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/drivers/memory"
)

// prepareQueueItemMemory 准备测试项
func prepareQueueItemMemory(queueName string, i int) *queue.QueueItem {
	return &queue.QueueItem{
		ID:          fmt.Sprintf("item-%d", i),
		WorkerName:  queueName,
		HandlerName: "benchmark-handler",
		Params:      []byte(`{"message":"benchmark task"}`),
		CreatedAt:   time.Now(),
		Retried:     0,
	}
}

// 使用内存队列的基准测试 - 添加操作
func BenchmarkMemoryQueueAdd(b *testing.B) {
	memQueue := memory.New()
	queueName := "benchmark-add"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := prepareQueueItemMemory(queueName, i)
		_ = memQueue.Add(queueName, item)
	}
}

// 使用内存队列的基准测试 - 弹出操作
func BenchmarkMemoryQueuePop(b *testing.B) {
	memQueue := memory.New()
	queueName := "benchmark-pop"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemMemory(queueName, i)
		_ = memQueue.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%1000 == 0 && i > 0 {
			// 每1000次操作重新添加项目
			b.StopTimer()
			for j := 0; j < 1000; j++ {
				item := prepareQueueItemMemory(queueName, j)
				_ = memQueue.Add(queueName, item)
			}
			b.StartTimer()
		}
		_ = memQueue.Pop(queueName, 1)
	}
}

// 使用内存队列的基准测试 - 删除操作
func BenchmarkMemoryQueueDel(b *testing.B) {
	memQueue := memory.New()
	queueName := "benchmark-del"

	// 预先添加一些项目并存储ID
	ids := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		item := prepareQueueItemMemory(queueName, i)
		ids[i] = item.ID
		_ = memQueue.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = memQueue.Del(queueName, ids[i])
	}
}

// 使用内存队列的基准测试 - 列表操作
func BenchmarkMemoryQueueList(b *testing.B) {
	memQueue := memory.New()
	queueName := "benchmark-list"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemMemory(queueName, i)
		_ = memQueue.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = memQueue.List(queueName, 1, 10)
	}
}

// 使用内存队列的基准测试 - 计数操作
func BenchmarkMemoryQueueCount(b *testing.B) {
	memQueue := memory.New()
	queueName := "benchmark-count"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemMemory(queueName, i)
		_ = memQueue.Add(queueName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = memQueue.Count(queueName)
	}
}
