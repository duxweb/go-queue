package memory

import (
	"fmt"
	"testing"
	"time"

	"github.com/duxweb/go-queue/internal/queue"
)

func TestMemoryQueue_Add(t *testing.T) {
	q := NewMemoryQueue()
	queueName := "test-queue"

	// 测试添加单个队列项
	// Test adding a single queue item
	item := &queue.QueueItem{
		ID:          "1",
		QueueName:   queueName,
		HandlerName: "test-handler",
		Params:      []byte(`{"test":"data"}`),
		CreatedAt:   time.Now(),
		Retried:     0,
	}

	err := q.Add(queueName, item)
	if err != nil {
		t.Errorf("Failed to add queue item: %v", err)
	}

	// 确认队列中有一个项目
	// Confirm there is one item in the queue
	if count := q.Count(queueName); count != 1 {
		t.Errorf("Expected queue count to be 1, got: %d", count)
	}

	// 测试添加多个队列项
	// Test adding multiple queue items
	for i := 2; i <= 5; i++ {
		id := fmt.Sprintf("%d", i)
		item := &queue.QueueItem{
			ID:          id,
			QueueName:   queueName,
			HandlerName: "test-handler",
			Params:      []byte(`{"test":"data"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		err := q.Add(queueName, item)
		if err != nil {
			t.Errorf("Failed to add queue item: %v", err)
		}
	}

	// 确认队列中有5个项目
	// Confirm there are 5 items in the queue
	if count := q.Count(queueName); count != 5 {
		t.Errorf("Expected queue count to be 5, got: %d", count)
	}
}

func TestMemoryQueue_Get(t *testing.T) {
	q := NewMemoryQueue()
	queueName := "test-queue"

	// 添加5个队列项，其中3个可以立即执行，2个延迟执行
	// Add 5 queue items, 3 for immediate execution, 2 delayed
	now := time.Now()

	// 可立即执行的项目
	// Items for immediate execution
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("%d", i)
		item := &queue.QueueItem{
			ID:          id,
			QueueName:   queueName,
			HandlerName: "test-handler",
			Params:      []byte(`{"test":"data"}`),
			CreatedAt:   now.Add(-time.Minute * time.Duration(i)), // 过去的时间 | Past time
			Retried:     0,
		}
		_ = q.Add(queueName, item)
	}

	// 延迟执行的项目
	// Delayed execution items
	for i := 4; i <= 5; i++ {
		id := fmt.Sprintf("%d", i)
		item := &queue.QueueItem{
			ID:          id,
			QueueName:   queueName,
			HandlerName: "test-handler",
			Params:      []byte(`{"test":"data"}`),
			CreatedAt:   now.Add(time.Minute * time.Duration(i)), // 未来的时间 | Future time
			Retried:     0,
		}
		_ = q.Add(queueName, item)
	}

	// 测试获取2个可执行的项目 - 这将从队列中移除它们
	// Test getting 2 executable items - this will remove them from the queue
	items := q.Pop(queueName, 2)
	if len(items) != 2 {
		t.Errorf("Expected to get 2 items, got: %d", len(items))
	}

	// 再次获取 - 应该只返回1个可执行项目，因为前2个已经被移除
	// Get again - should only return 1 executable item, as the first 2 were removed
	items = q.Pop(queueName, 10)
	if len(items) != 1 {
		t.Errorf("Expected to get 1 executable item, got: %d", len(items))
	}

	// 验证队列中还剩2个延迟执行的项目
	// Verify there are 2 delayed items left in the queue
	if count := q.Count(queueName); count != 2 {
		t.Errorf("Expected 2 items left in queue, got: %d", count)
	}

	// 测试非存在的队列
	// Test non-existent queue
	items = q.Pop("non-existent", 10)
	if len(items) != 0 {
		t.Errorf("Expected to get 0 items, got: %d", len(items))
	}
}

func TestMemoryQueue_Del(t *testing.T) {
	q := NewMemoryQueue()
	queueName := "test-queue"

	// 添加队列项
	// Add queue item
	item := &queue.QueueItem{
		ID:          "1",
		QueueName:   queueName,
		HandlerName: "test-handler",
		Params:      []byte(`{"test":"data"}`),
		CreatedAt:   time.Now(),
		Retried:     0,
	}
	_ = q.Add(queueName, item)

	// 删除队列项
	// Delete queue item
	err := q.Del(queueName, "1")
	if err != nil {
		t.Errorf("Failed to delete queue item: %v", err)
	}

	// 确认队列为空
	// Confirm queue is empty
	if count := q.Count(queueName); count != 0 {
		t.Errorf("Expected queue count to be 0, got: %d", count)
	}

	// 测试删除不存在的项目
	// Test deleting non-existent item
	err = q.Del(queueName, "999")
	if err != nil {
		t.Errorf("Deleting non-existent queue item should succeed, but got error: %v", err)
	}

	// 测试删除不存在的队列
	// Test deleting from non-existent queue
	err = q.Del("non-existent", "1")
	if err != nil {
		t.Errorf("Deleting from non-existent queue should succeed, but got error: %v", err)
	}
}

func TestMemoryQueue_List(t *testing.T) {
	q := NewMemoryQueue()
	queueName := "test-queue"

	// 添加10个队列项
	// Add 10 queue items
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("%d", i)
		item := &queue.QueueItem{
			ID:          id,
			QueueName:   queueName,
			HandlerName: "test-handler",
			Params:      []byte(`{"test":"data"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		_ = q.Add(queueName, item)
	}

	// 测试第一页，每页3个
	// Test first page, 3 items per page
	items := q.List(queueName, 1, 3)
	if len(items) != 3 {
		t.Errorf("Expected to get 3 items, got: %d", len(items))
	}

	// 测试第二页，每页3个
	// Test second page, 3 items per page
	items = q.List(queueName, 2, 3)
	if len(items) != 3 {
		t.Errorf("Expected to get 3 items, got: %d", len(items))
	}

	// 测试最后一页
	// Test last page
	items = q.List(queueName, 4, 3)
	if len(items) != 1 {
		t.Errorf("Expected to get 1 item, got: %d", len(items))
	}

	// 测试超出范围的页
	// Test page out of range
	items = q.List(queueName, 5, 3)
	if len(items) != 0 {
		t.Errorf("Expected to get 0 items, got: %d", len(items))
	}

	// 测试非存在的队列
	// Test non-existent queue
	items = q.List("non-existent", 1, 3)
	if len(items) != 0 {
		t.Errorf("Expected to get 0 items, got: %d", len(items))
	}
}

func TestMemoryQueue_Count(t *testing.T) {
	q := NewMemoryQueue()
	queueName := "test-queue"

	// 测试空队列
	// Test empty queue
	if count := q.Count(queueName); count != 0 {
		t.Errorf("Expected empty queue count to be 0, got: %d", count)
	}

	// 添加5个队列项
	// Add 5 queue items
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("%d", i)
		item := &queue.QueueItem{
			ID:          id,
			QueueName:   queueName,
			HandlerName: "test-handler",
			Params:      []byte(`{"test":"data"}`),
			CreatedAt:   time.Now(),
			Retried:     0,
		}
		_ = q.Add(queueName, item)
	}

	// 测试队列数量
	// Test queue count
	if count := q.Count(queueName); count != 5 {
		t.Errorf("Expected queue count to be 5, got: %d", count)
	}

	// 测试不存在的队列
	// Test non-existent queue
	if count := q.Count("non-existent"); count != 0 {
		t.Errorf("Expected non-existent queue count to be 0, got: %d", count)
	}
}
