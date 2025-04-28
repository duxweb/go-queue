package drivers_test

import (
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/duxweb/go-queue/drivers"
	"github.com/stretchr/testify/assert"
)

func TestMemoryQueueBasicOperations(t *testing.T) {
	// 创建内存队列
	memQueue := drivers.NewMemoryQueue()

	// 测试空队列
	items := memQueue.Pop("test-queue", 10)
	assert.Empty(t, items, "从空队列获取项目应该返回空列表")

	// 添加项目
	testItem := &queue.QueueItem{
		ID:          "test-id",
		WorkerName:  "test-queue",
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   time.Now(),
	}

	err := memQueue.Add("test-queue", testItem)
	assert.NoError(t, err, "添加队列项应该成功")

	// 验证计数
	count := memQueue.Count("test-queue")
	assert.Equal(t, 1, count, "队列应该有1个项目")

	// 获取项目
	poppedItems := memQueue.Pop("test-queue", 10)
	assert.Len(t, poppedItems, 1, "应该获取1个项目")
	assert.Equal(t, testItem.ID, poppedItems[0].ID, "获取的项目ID应该匹配")

	// 验证队列现在为空
	count = memQueue.Count("test-queue")
	assert.Equal(t, 0, count, "弹出后队列应该为空")
}

func TestMemoryQueueDelayedItems(t *testing.T) {
	// 创建内存队列
	memQueue := drivers.NewMemoryQueue()

	// 添加一个延迟的项目（未来时间）
	futureTime := time.Now().Add(time.Hour)
	delayedItem := &queue.QueueItem{
		ID:          "delayed-id",
		WorkerName:  "test-queue",
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   futureTime,
	}

	err := memQueue.Add("test-queue", delayedItem)
	assert.NoError(t, err, "添加延迟队列项应该成功")

	// 尝试获取项目，但应该不会返回（因为还没有到执行时间）
	poppedItems := memQueue.Pop("test-queue", 10)
	assert.Empty(t, poppedItems, "不应该获取未到执行时间的项目")

	// 验证项目仍在队列中
	count := memQueue.Count("test-queue")
	assert.Equal(t, 1, count, "延迟的项目应该仍在队列中")
}
