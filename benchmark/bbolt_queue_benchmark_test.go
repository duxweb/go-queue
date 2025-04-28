package benchmark

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/drivers/bbolt"
)

// prepareQueueItemBBolt 准备测试项
func prepareQueueItemBBolt(workerName string, i int) *queue.QueueItem {
	return &queue.QueueItem{
		ID:          fmt.Sprintf("item-%d", i),
		WorkerName:  workerName,
		HandlerName: "benchmark-handler",
		Params:      []byte(`{"message":"benchmark task"}`),
		CreatedAt:   time.Now(),
		Retried:     0,
	}
}

// 使用BBolt队列的基准测试 - 添加操作
func BenchmarkBBoltQueueAdd(b *testing.B) {
	dbPath := "./test-benchmark-bbolt.db"
	defer os.Remove(dbPath)

	boltQueue, err := bbolt.New(dbPath, nil)
	if err != nil {
		b.Fatalf("创建BBolt队列失败: %v", err)
	}
	defer boltQueue.Close()

	workerName := "benchmark-add"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := prepareQueueItemBBolt(workerName, i)
		_ = boltQueue.Add(workerName, item)
	}
}

// 使用BBolt队列的基准测试 - 弹出操作
func BenchmarkBBoltQueuePop(b *testing.B) {
	dbPath := "./test-benchmark-bbolt.db"
	defer os.Remove(dbPath)

	boltQueue, err := bbolt.New(dbPath, nil)
	if err != nil {
		b.Fatalf("创建BBolt队列失败: %v", err)
	}
	defer boltQueue.Close()

	workerName := "benchmark-pop"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemBBolt(workerName, i)
		_ = boltQueue.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%1000 == 0 && i > 0 {
			// 每1000次操作重新添加项目
			b.StopTimer()
			for j := 0; j < 1000; j++ {
				item := prepareQueueItemBBolt(workerName, j)
				_ = boltQueue.Add(workerName, item)
			}
			b.StartTimer()
		}
		_ = boltQueue.Pop(workerName, 1)
	}
}

// 使用BBolt队列的基准测试 - 删除操作
func BenchmarkBBoltQueueDel(b *testing.B) {
	dbPath := "./test-benchmark-bbolt.db"
	defer os.Remove(dbPath)

	boltQueue, err := bbolt.New(dbPath, nil)
	if err != nil {
		b.Fatalf("创建BBolt队列失败: %v", err)
	}
	defer boltQueue.Close()

	workerName := "benchmark-del"

	// 预先添加一些项目并存储ID
	ids := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		item := prepareQueueItemBBolt(workerName, i)
		ids[i] = item.ID
		_ = boltQueue.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = boltQueue.Del(workerName, ids[i])
	}
}

// 使用BBolt队列的基准测试 - 列表操作
func BenchmarkBBoltQueueList(b *testing.B) {
	dbPath := "./test-benchmark-bbolt.db"
	defer os.Remove(dbPath)

	boltQueue, err := bbolt.New(dbPath, nil)
	if err != nil {
		b.Fatalf("创建BBolt队列失败: %v", err)
	}
	defer boltQueue.Close()

	workerName := "benchmark-list"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemBBolt(workerName, i)
		_ = boltQueue.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = boltQueue.List(workerName, 1, 10)
	}
}

// 使用BBolt队列的基准测试 - 计数操作
func BenchmarkBBoltQueueCount(b *testing.B) {
	dbPath := "./test-benchmark-bbolt.db"
	defer os.Remove(dbPath)

	boltQueue, err := bbolt.New(dbPath, nil)
	if err != nil {
		b.Fatalf("创建BBolt队列失败: %v", err)
	}
	defer boltQueue.Close()

	workerName := "benchmark-count"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemBBolt(workerName, i)
		_ = boltQueue.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = boltQueue.Count(workerName)
	}
}
