package bbolt

import (
	"os"
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/stretchr/testify/assert"
)

// 创建测试用的BBolt队列
func setupTestBoltQueue(t *testing.T) (*BoltQueue, func()) {
	dbPath := "./test-bbolt-unit.db"

	// 创建BBolt队列
	boltQueue, err := New(dbPath, nil)
	if err != nil {
		t.Fatalf("创建BBolt队列失败: %v", err)
	}

	// 返回清理函数
	cleanup := func() {
		boltQueue.Close()
		os.Remove(dbPath)
	}

	return boltQueue, cleanup
}

// 测试驱动接口实现
func TestBoltQueueImplementsQueueDriver(t *testing.T) {
	boltQueue, cleanup := setupTestBoltQueue(t)
	defer cleanup()

	var _ queue.QueueDriver = boltQueue
}

// 测试基本队列操作
func TestBoltQueueBasicOperations(t *testing.T) {
	boltQueue, cleanup := setupTestBoltQueue(t)
	defer cleanup()

	queueName := "test-queue"

	// 测试空队列
	assert.Equal(t, 0, boltQueue.Count(queueName), "空队列数量应该为0")
	assert.Empty(t, boltQueue.Pop(queueName, 10), "从空队列获取项目应该返回空列表")
	assert.Empty(t, boltQueue.List(queueName, 1, 10), "列出空队列应该返回空列表")

	// 测试添加项目
	testItem := &queue.QueueItem{
		ID:          "test-id",
		WorkerName:  queueName,
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   time.Now(),
	}

	err := boltQueue.Add(queueName, testItem)
	assert.NoError(t, err, "添加队列项应该成功")

	// 测试计数和列表
	assert.Equal(t, 1, boltQueue.Count(queueName), "队列应该有1个项目")
	items := boltQueue.List(queueName, 1, 10)
	assert.Len(t, items, 1, "应该列出1个项目")
	assert.Equal(t, testItem.ID, items[0].ID, "列出的项目ID应该匹配")

	// 测试弹出
	poppedItems := boltQueue.Pop(queueName, 10)
	assert.Len(t, poppedItems, 1, "应该获取1个项目")
	assert.Equal(t, testItem.ID, poppedItems[0].ID, "获取的项目ID应该匹配")

	// 验证弹出后队列为空
	assert.Equal(t, 0, boltQueue.Count(queueName), "弹出后队列应该为空")
}

// 测试延迟项目
func TestBoltQueueDelayedItems(t *testing.T) {
	boltQueue, cleanup := setupTestBoltQueue(t)
	defer cleanup()

	queueName := "test-queue"

	// 添加一个延迟项目（未来时间）
	futureTime := time.Now().Add(time.Second * 1)
	delayedItem := &queue.QueueItem{
		ID:          "delayed-id",
		WorkerName:  queueName,
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   futureTime,
	}

	err := boltQueue.Add(queueName, delayedItem)
	assert.NoError(t, err, "添加延迟队列项应该成功")

	// 尝试获取项目，但应该不会返回（因为还没有到执行时间）
	assert.Empty(t, boltQueue.Pop(queueName, 10), "不应该获取未到执行时间的项目")

	// 验证项目仍在队列中
	assert.Equal(t, 1, boltQueue.Count(queueName), "延迟的项目应该仍在队列中")

	// 等待时间到达，再次尝试弹出
	time.Sleep(time.Second * 1)
	poppedItems := boltQueue.Pop(queueName, 10)
	assert.Len(t, poppedItems, 1, "延迟时间到达后应该能弹出项目")
	assert.Equal(t, delayedItem.ID, poppedItems[0].ID, "弹出的项目ID应该匹配")
}

// 测试删除操作
func TestBoltQueueDelete(t *testing.T) {
	boltQueue, cleanup := setupTestBoltQueue(t)
	defer cleanup()

	queueName := "test-queue"

	// 添加项目
	testItem := &queue.QueueItem{
		ID:          "test-id",
		WorkerName:  queueName,
		HandlerName: "test-handler",
		Params:      []byte(`{"key":"value"}`),
		CreatedAt:   time.Now(),
	}

	err := boltQueue.Add(queueName, testItem)
	assert.NoError(t, err, "添加队列项应该成功")
	assert.Equal(t, 1, boltQueue.Count(queueName), "队列应该有1个项目")

	// 测试删除
	err = boltQueue.Del(queueName, testItem.ID)
	assert.NoError(t, err, "删除队列项应该成功")
	assert.Equal(t, 0, boltQueue.Count(queueName), "删除后队列应该为空")

	// 测试删除不存在的项目
	err = boltQueue.Del(queueName, "non-existing-id")
	assert.NoError(t, err, "删除不存在的项目应该不返回错误")
}

// 测试分页列表
func TestBoltQueuePagination(t *testing.T) {
	boltQueue, cleanup := setupTestBoltQueue(t)
	defer cleanup()

	queueName := "test-queue"

	// 添加多个项目
	for i := 0; i < 20; i++ {
		item := &queue.QueueItem{
			ID:          "item-" + time.Now().Add(time.Duration(i)*time.Millisecond).Format("150405.000"),
			WorkerName:  queueName,
			HandlerName: "test-handler",
			Params:      []byte(`{"key":"value"}`),
			CreatedAt:   time.Now(),
		}
		boltQueue.Add(queueName, item)
	}

	assert.Equal(t, 20, boltQueue.Count(queueName), "队列应该有20个项目")

	// 测试第一页
	page1 := boltQueue.List(queueName, 1, 5)
	assert.Len(t, page1, 5, "第一页应该有5个项目")

	// 测试第二页
	page2 := boltQueue.List(queueName, 2, 5)
	assert.Len(t, page2, 5, "第二页应该有5个项目")

	// 确保两页项目不重复
	for _, item1 := range page1 {
		for _, item2 := range page2 {
			assert.NotEqual(t, item1.ID, item2.ID, "不同页的项目不应重复")
		}
	}

	// 测试超出范围的页
	assert.Empty(t, boltQueue.List(queueName, 5, 5), "超出范围的页应该返回空列表")
}

// 测试BBolt选项
func TestBoltQueueOptions(t *testing.T) {
	// 测试默认选项
	options := DefaultOptions()
	assert.Equal(t, 0600, int(options.FileMode), "默认FileMode应该是0600")
	assert.Equal(t, time.Second*1, options.Timeout, "默认Timeout应该是1秒")

	// 测试自定义选项
	dbPath := "./test-bbolt-options.db"
	customOptions := &Options{
		FileMode: 0644,
		Timeout:  time.Second * 2,
	}

	boltQueue, err := New(dbPath, customOptions)
	assert.NoError(t, err, "使用自定义选项创建BBolt队列应该成功")

	// 清理
	defer func() {
		boltQueue.Close()
		os.Remove(dbPath)
	}()

	// 验证选项
	assert.Equal(t, customOptions, boltQueue.options, "选项应该被正确设置")
}
