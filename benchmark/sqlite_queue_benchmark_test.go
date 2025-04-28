package benchmark

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/drivers/sqlite"
)

// prepareQueueItemSQLite 准备测试项
func prepareQueueItemSQLite(workerName string, i int) *queue.QueueItem {
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

// setupSQLiteQueue 设置SQLite测试队列
func setupSQLiteQueue(b *testing.B) *sqlite.SQLiteQueue {
	// 使用临时数据库文件
	tmpDBPath := "benchmark_sqlite.db"

	// 确保测试开始前删除已有的测试数据库
	os.Remove(tmpDBPath)

	options := &sqlite.SQLiteOptions{
		DBPath: tmpDBPath,
	}

	q, err := sqlite.New(options)
	if err != nil {
		b.Fatalf("Failed to create SQLite queue: %v", err)
	}

	// 在测试完成后清理
	b.Cleanup(func() {
		q.Close()
		os.Remove(tmpDBPath)
	})

	return q
}

// 使用SQLite队列的基准测试 - 添加操作
func BenchmarkSQLiteQueueAdd(b *testing.B) {
	q := setupSQLiteQueue(b)
	workerName := "benchmark-add"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := prepareQueueItemSQLite(workerName, i)
		_ = q.Add(workerName, item)
	}
}

// 使用SQLite队列的基准测试 - 弹出操作
func BenchmarkSQLiteQueuePop(b *testing.B) {
	q := setupSQLiteQueue(b)
	workerName := "benchmark-pop"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemSQLite(workerName, i)
		_ = q.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%1000 == 0 && i > 0 {
			// 每1000次操作重新添加项目
			b.StopTimer()
			for j := 0; j < 1000; j++ {
				item := prepareQueueItemSQLite(workerName, j)
				_ = q.Add(workerName, item)
			}
			b.StartTimer()
		}
		_ = q.Pop(workerName, 1)
	}
}

// 使用SQLite队列的基准测试 - 删除操作
func BenchmarkSQLiteQueueDel(b *testing.B) {
	q := setupSQLiteQueue(b)
	workerName := "benchmark-del"

	// 预先添加一些项目并存储ID
	ids := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		item := prepareQueueItemSQLite(workerName, i)
		ids[i] = item.ID
		_ = q.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Del(workerName, ids[i])
	}
}

// 使用SQLite队列的基准测试 - 列表操作
func BenchmarkSQLiteQueueList(b *testing.B) {
	q := setupSQLiteQueue(b)
	workerName := "benchmark-list"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemSQLite(workerName, i)
		_ = q.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.List(workerName, 1, 10)
	}
}

// 使用SQLite队列的基准测试 - 计数操作
func BenchmarkSQLiteQueueCount(b *testing.B) {
	q := setupSQLiteQueue(b)
	workerName := "benchmark-count"

	// 预先添加一些项目
	for i := 0; i < 1000; i++ {
		item := prepareQueueItemSQLite(workerName, i)
		_ = q.Add(workerName, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Count(workerName)
	}
}
