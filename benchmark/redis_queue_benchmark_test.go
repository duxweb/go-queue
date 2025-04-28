package benchmark

import (
	"fmt"
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	redisdriver "github.com/duxweb/go-queue/drivers/redis"
)

// prepareQueueItemRedis 准备测试项
func prepareQueueItemRedis(workerName string, i int) *queue.QueueItem {
	return &queue.QueueItem{
		ID:          fmt.Sprintf("item-%d", i),
		WorkerName:  workerName,
		HandlerName: "benchmark-handler",
		Params:      []byte(`{"message":"benchmark task"}`),
		CreatedAt:   time.Now(),
		RunAt:       time.Now(),
		Retried:     0,
	}
}

// setupRedisQueue 设置Redis测试队列
func setupRedisQueue(b *testing.B) *redisdriver.RedisQueue {
	// 使用临时数据库
	options := &redisdriver.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15, // 使用DB 15作为测试数据库
		PoolSize: 10,
		Timeout:  time.Second * 3,
	}

	q, err := redisdriver.New(options)
	if err != nil {
		b.Skipf("Redis不可用，跳过测试: %v", err)
		return nil
	}

	// 在测试前清理数据库
	err = q.FlushAll()
	if err != nil {
		b.Fatalf("清理Redis测试数据库失败: %v", err)
	}

	// 在测试完成后清理
	b.Cleanup(func() {
		q.FlushAll()
		q.Close()
	})

	return q
}

// 使用Redis队列的基准测试 - 添加操作
func BenchmarkRedisQueueAdd(b *testing.B) {
	q := setupRedisQueue(b)
	if q == nil {
		return
	}

	workerName := "benchmark-add"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := prepareQueueItemRedis(workerName, i)
		_ = q.Add(workerName, item)
	}
}

// 使用Redis队列的基准测试 - 弹出操作
func BenchmarkRedisQueuePop(b *testing.B) {
	q := setupRedisQueue(b)
	if q == nil {
		return
	}

	workerName := "benchmark-pop"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemRedis(workerName, i)
		_ = q.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%1000 == 0 && i > 0 {
			// 每1000次操作重新添加项目
			b.StopTimer()
			for j := 0; j < 1000; j++ {
				item := prepareQueueItemRedis(workerName, j)
				_ = q.Add(workerName, item)
			}
			b.StartTimer()
		}
		_ = q.Pop(workerName, 1)
	}
}

// 使用Redis队列的基准测试 - 删除操作
func BenchmarkRedisQueueDel(b *testing.B) {
	q := setupRedisQueue(b)
	if q == nil {
		return
	}

	workerName := "benchmark-del"

	// 预先添加一些项目并存储ID
	ids := make([]string, b.N)
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		item := prepareQueueItemRedis(workerName, i)
		ids[i] = item.ID
		_ = q.Add(workerName, item)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Del(workerName, ids[i])
	}
}

// 使用Redis队列的基准测试 - 列表操作
func BenchmarkRedisQueueList(b *testing.B) {
	q := setupRedisQueue(b)
	if q == nil {
		return
	}

	workerName := "benchmark-list"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemRedis(workerName, i)
		_ = q.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.List(workerName, 1, 10)
	}
}

// 使用Redis队列的基准测试 - 计数操作
func BenchmarkRedisQueueCount(b *testing.B) {
	q := setupRedisQueue(b)
	if q == nil {
		return
	}

	workerName := "benchmark-count"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemRedis(workerName, i)
		_ = q.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Count(workerName)
	}
}

// 使用Redis队列的基准测试 - 批量添加操作
func BenchmarkRedisQueueBatchAdd(b *testing.B) {
	q := setupRedisQueue(b)
	if q == nil {
		return
	}

	workerName := "benchmark-batch-add"
	batchSize := 100

	b.ResetTimer()
	for i := 0; i < b.N; i += batchSize {
		// 确保不超过 b.N
		currentBatchSize := batchSize
		if i+batchSize > b.N {
			currentBatchSize = b.N - i
		}

		items := make([]*queue.QueueItem, currentBatchSize)
		for j := 0; j < currentBatchSize; j++ {
			items[j] = prepareQueueItemRedis(workerName, i+j)
		}

		// 创建一个批量添加的测试
		for _, item := range items {
			_ = q.Add(workerName, item)
		}
	}
}
