package memory

import (
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/stretchr/testify/assert"
)

// 测试驱动接口实现
func TestMemoryQueueImplementsQueueDriver(t *testing.T) {
	memQueue := New()
	var _ queue.QueueDriver = memQueue
}

// 测试基本队列操作
func TestMemoryQueueBasicOperations(t *testing.T) {
	memQueue := New()
	workerName := "test-queue"

	// 测试空队列
	assert.Equal(t, 0, memQueue.Count(workerName), "空队列数量应该为0")
	assert.Empty(t, memQueue.Pop(workerName, 10), "从空队列获取项目应该返回空列表")
	assert.Empty(t, memQueue.List(workerName, 1, 10), "列出空队列应该返回空列表")

	// 测试添加项目
	testItem := &queue.QueueItem{
		ID:          "test-id",
		WorkerName:  workerName,
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   time.Now(),
	}

	err := memQueue.Add(workerName, testItem)
	assert.NoError(t, err, "添加队列项应该成功")

	// 测试计数和列表
	assert.Equal(t, 1, memQueue.Count(workerName), "队列应该有1个项目")
	items := memQueue.List(workerName, 1, 10)
	assert.Len(t, items, 1, "应该列出1个项目")
	assert.Equal(t, testItem.ID, items[0].ID, "列出的项目ID应该匹配")

	// 测试弹出
	poppedItems := memQueue.Pop(workerName, 10)
	assert.Len(t, poppedItems, 1, "应该获取1个项目")
	assert.Equal(t, testItem.ID, poppedItems[0].ID, "获取的项目ID应该匹配")

	// 验证弹出后队列为空
	assert.Equal(t, 0, memQueue.Count(workerName), "弹出后队列应该为空")
}

// 测试延迟项目
func TestMemoryQueueDelayedItems(t *testing.T) {
	memQueue := New()
	workerName := "test-queue"

	// 添加一个延迟项目（未来时间）
	futureTime := time.Now().Add(time.Second * 1)
	delayedItem := &queue.QueueItem{
		ID:          "delayed-id",
		WorkerName:  workerName,
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   futureTime,
	}

	err := memQueue.Add(workerName, delayedItem)
	assert.NoError(t, err, "添加延迟队列项应该成功")

	// 尝试获取项目，但应该不会返回（因为还没有到执行时间）
	assert.Empty(t, memQueue.Pop(workerName, 10), "不应该获取未到执行时间的项目")

	// 验证项目仍在队列中
	assert.Equal(t, 1, memQueue.Count(workerName), "延迟的项目应该仍在队列中")

	// 等待时间到达，再次尝试弹出
	time.Sleep(time.Second * 1)
	poppedItems := memQueue.Pop(workerName, 10)
	assert.Len(t, poppedItems, 1, "延迟时间到达后应该能弹出项目")
	assert.Equal(t, delayedItem.ID, poppedItems[0].ID, "弹出的项目ID应该匹配")
}

// 测试删除操作
func TestMemoryQueueDelete(t *testing.T) {
	memQueue := New()
	workerName := "test-queue"

	// 添加项目
	testItem := &queue.QueueItem{
		ID:          "test-id",
		WorkerName:  workerName,
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   time.Now(),
	}

	err := memQueue.Add(workerName, testItem)
	assert.NoError(t, err, "添加队列项应该成功")
	assert.Equal(t, 1, memQueue.Count(workerName), "队列应该有1个项目")

	// 测试删除
	err = memQueue.Del(workerName, testItem.ID)
	assert.NoError(t, err, "删除队列项应该成功")
	assert.Equal(t, 0, memQueue.Count(workerName), "删除后队列应该为空")

	// 测试删除不存在的项目
	err = memQueue.Del(workerName, "non-existing-id")
	assert.NoError(t, err, "删除不存在的项目应该不返回错误")
}

// 测试分页列表
func TestMemoryQueuePagination(t *testing.T) {
	memQueue := New()
	workerName := "test-queue"

	// 添加多个项目
	for i := 0; i < 20; i++ {
		item := &queue.QueueItem{
			ID:          "item-" + time.Now().Add(time.Duration(i)*time.Millisecond).Format("150405.000"),
			WorkerName:  workerName,
			HandlerName: "test-handler",
			Params:      []byte(`{"key":"value"}`),
			CreatedAt:   time.Now(),
		}
		memQueue.Add(workerName, item)
	}

	assert.Equal(t, 20, memQueue.Count(workerName), "队列应该有20个项目")

	// 测试第一页
	page1 := memQueue.List(workerName, 1, 5)
	assert.Len(t, page1, 5, "第一页应该有5个项目")

	// 测试第二页
	page2 := memQueue.List(workerName, 2, 5)
	assert.Len(t, page2, 5, "第二页应该有5个项目")

	// 确保两页项目不重复
	for _, item1 := range page1 {
		for _, item2 := range page2 {
			assert.NotEqual(t, item1.ID, item2.ID, "不同页的项目不应重复")
		}
	}

	// 注意：根据内存队列的实现，页码为0或负数可能会返回空列表，或者可能会抛出异常
	// 这里我们只测试超出范围的页码
	assert.Empty(t, memQueue.List(workerName, 5, 5), "超出范围的页应该返回空列表")
}

// 测试关闭操作
func TestMemoryQueueClose(t *testing.T) {
	memQueue := New()
	err := memQueue.Close()
	assert.NoError(t, err, "关闭内存队列应该成功")
}
