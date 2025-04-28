package sqlite

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/stretchr/testify/assert"
)

func setupTestSQLiteQueue(t *testing.T) *SQLiteQueue {
	// 使用临时数据库文件
	tmpDBPath := "test_queue.db"

	// 确保测试开始前删除已有的测试数据库
	os.Remove(tmpDBPath)

	options := &SQLiteOptions{
		DBPath: tmpDBPath,
	}

	q, err := New(options)
	assert.NoError(t, err)
	assert.NotNil(t, q)

	t.Cleanup(func() {
		q.Close()
		os.Remove(tmpDBPath)
	})

	return q
}

func TestSQLiteQueueImplementsQueueDriver(t *testing.T) {
	var _ queue.QueueDriver = (*SQLiteQueue)(nil)
}

func TestSQLiteQueueBasicOperations(t *testing.T) {
	q := setupTestSQLiteQueue(t)

	// 测试空队列的Count
	assert.Equal(t, 0, q.Count("test-worker"))

	// 添加一个项目
	item := &queue.QueueItem{
		ID:          "test-id-1",
		WorkerName:  "test-worker",
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   time.Now(),
		RunAt:       time.Now(),
		Retried:     0,
	}

	err := q.Add("test-worker", item)
	assert.NoError(t, err)
	assert.Equal(t, 1, q.Count("test-worker"))

	// 弹出项目
	items := q.Pop("test-worker", 10)
	assert.Len(t, items, 1)
	assert.Equal(t, "test-id-1", items[0].ID)
	assert.Equal(t, 0, q.Count("test-worker"))

	// 测试删除项目
	// 先添加
	err = q.Add("test-worker", item)
	assert.NoError(t, err)
	assert.Equal(t, 1, q.Count("test-worker"))

	// 再删除
	err = q.Del("test-worker", "test-id-1")
	assert.NoError(t, err)
	assert.Equal(t, 0, q.Count("test-worker"))
}

func TestSQLiteQueueDelayedItems(t *testing.T) {
	q := setupTestSQLiteQueue(t)

	// 添加一个延迟项目（10秒后执行）
	futureTime := time.Now().Add(10 * time.Second)
	delayedItem := &queue.QueueItem{
		ID:          "delayed-id-1",
		WorkerName:  "test-worker",
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   time.Now(),
		RunAt:       futureTime,
		Retried:     0,
	}

	err := q.Add("test-worker", delayedItem)
	assert.NoError(t, err)
	assert.Equal(t, 1, q.Count("test-worker"))

	// 现在尝试Pop，应该返回空，因为项目被延迟了
	items := q.Pop("test-worker", 10)
	assert.Len(t, items, 0)
	assert.Equal(t, 1, q.Count("test-worker"))

	// 添加一个明确的立即执行的项目 - 使用明确的过去时间
	pastTime := time.Now().Add(-1 * time.Second)
	immediateItem := &queue.QueueItem{
		ID:          "immediate-id-1",
		WorkerName:  "test-worker",
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   pastTime,
		RunAt:       pastTime,
		Retried:     0,
	}

	err = q.Add("test-worker", immediateItem)
	assert.NoError(t, err)
	assert.Equal(t, 2, q.Count("test-worker"))

	// 现在Pop应该只返回立即执行的项目
	items = q.Pop("test-worker", 10)
	assert.Len(t, items, 1)
	assert.Equal(t, "immediate-id-1", items[0].ID)
	assert.Equal(t, 1, q.Count("test-worker"))
}

func TestSQLiteQueueDelete(t *testing.T) {
	q := setupTestSQLiteQueue(t)

	// 添加多个项目
	for i := 1; i <= 3; i++ {
		item := &queue.QueueItem{
			ID:          fmt.Sprintf("test-id-%d", i),
			WorkerName:  "test-worker",
			HandlerName: "test-handler",
			Params:      []byte(`{"key":"value"}`),
			CreatedAt:   time.Now(),
			RunAt:       time.Now(),
			Retried:     0,
		}

		err := q.Add("test-worker", item)
		assert.NoError(t, err)
	}

	assert.Equal(t, 3, q.Count("test-worker"))

	// 删除中间的项目
	err := q.Del("test-worker", "test-id-2")
	assert.NoError(t, err)
	assert.Equal(t, 2, q.Count("test-worker"))

	// 获取项目，确保删除的是正确的项目
	items := q.Pop("test-worker", 10)
	assert.Len(t, items, 2)

	// 检查是否包含ID 1和3，但不包含ID 2
	itemIDs := []string{items[0].ID, items[1].ID}
	assert.Contains(t, itemIDs, "test-id-1")
	assert.Contains(t, itemIDs, "test-id-3")
	assert.NotContains(t, itemIDs, "test-id-2")
}

func TestSQLiteQueuePagination(t *testing.T) {
	q := setupTestSQLiteQueue(t)

	// 添加10个项目
	for i := 1; i <= 10; i++ {
		item := &queue.QueueItem{
			ID:          fmt.Sprintf("test-id-%d", i),
			WorkerName:  "test-worker",
			HandlerName: "test-handler",
			Params:      []byte(`{"key":"value"}`),
			CreatedAt:   time.Now(),
			RunAt:       time.Now().Add(time.Duration(i) * time.Minute), // 按顺序延迟，以便测试排序
			Retried:     0,
		}

		err := q.Add("test-worker", item)
		assert.NoError(t, err)
	}

	assert.Equal(t, 10, q.Count("test-worker"))

	// 测试分页
	page1 := q.List("test-worker", 1, 3)
	assert.Len(t, page1, 3)
	assert.Equal(t, "test-id-1", page1[0].ID) // 按RunAt排序，所以最早执行的在前面

	page2 := q.List("test-worker", 2, 3)
	assert.Len(t, page2, 3)
	assert.Equal(t, "test-id-4", page2[0].ID)

	page4 := q.List("test-worker", 4, 3)
	assert.Len(t, page4, 1) // 最后一页只有1个
	assert.Equal(t, "test-id-10", page4[0].ID)

	// 超出范围的页码
	page5 := q.List("test-worker", 5, 3)
	assert.Len(t, page5, 0)
}

func TestSQLiteQueueOptions(t *testing.T) {
	// 测试默认选项
	defaultOptions := DefaultOptions()
	assert.Equal(t, "queue.db", defaultOptions.DBPath)

	// 测试自定义选项
	customOptions := &SQLiteOptions{
		DBPath: "custom.db",
	}

	// 清理
	os.Remove(customOptions.DBPath)

	q, err := New(customOptions)
	assert.NoError(t, err)
	assert.NotNil(t, q)
	assert.Equal(t, "custom.db", q.dbPath)

	q.Close()
	os.Remove(customOptions.DBPath)
}
