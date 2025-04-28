package memory

import (
	"sync"
	"time"

	"github.com/duxweb/go-queue"
)

// MemoryQueue 是一个简单的内存队列实现
// MemoryQueue is a simple in-memory queue implementation
type MemoryQueue struct {
	mu      sync.RWMutex
	queues  map[string][]*queue.QueueItem
	indices map[string]map[string]int
}

// New 创建一个新的内存队列实例
// New creates a new memory queue instance
func New() *MemoryQueue {
	return &MemoryQueue{
		queues:  make(map[string][]*queue.QueueItem),
		indices: make(map[string]map[string]int),
	}
}

// Pop 从队列中获取并移除指定数量的队列项
// Pop retrieves and removes the specified number of queue items
func (m *MemoryQueue) Pop(workerName string, num int) []*queue.QueueItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, ok := m.queues[workerName]
	if !ok || len(q) == 0 {
		return []*queue.QueueItem{}
	}

	// 如果请求的数量大于队列中的项目数，则设置为队列长度
	if num > len(q) {
		num = len(q)
	}

	// 获取可执行的任务（当前时间大于或等于队列项的创建时间）
	var result []*queue.QueueItem
	now := time.Now()

	// 使用新的队列保存剩余项目
	var newQueue []*queue.QueueItem

	// 使用新的索引映射
	idxMap := make(map[string]int)

	// 遍历队列中的所有项目
	for _, item := range q {
		// 如果可以执行且结果数量未达到要求，则添加到结果中
		if (now.After(item.CreatedAt) || now.Equal(item.CreatedAt)) && len(result) < num {
			result = append(result, item)
			// 从索引映射中删除
			if idx, exists := m.indices[workerName]; exists {
				delete(idx, item.ID)
			}
		} else {
			// 否则保留在新队列中
			idxMap[item.ID] = len(newQueue)
			newQueue = append(newQueue, item)
		}
	}

	// 更新队列和索引
	m.queues[workerName] = newQueue
	m.indices[workerName] = idxMap

	return result
}

// Add 向队列添加新的队列项
// Add adds a new queue item to the queue
func (m *MemoryQueue) Add(workerName string, item *queue.QueueItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保队列和索引映射存在
	if _, ok := m.queues[workerName]; !ok {
		m.queues[workerName] = []*queue.QueueItem{}
		m.indices[workerName] = make(map[string]int)
	}

	// 添加到队列
	m.queues[workerName] = append(m.queues[workerName], item)

	// 更新索引映射
	m.indices[workerName][item.ID] = len(m.queues[workerName]) - 1

	return nil
}

// Del 从队列中删除指定的队列项
// Del removes the specified queue item from the queue
func (m *MemoryQueue) Del(workerName string, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, ok := m.queues[workerName]
	if !ok {
		return nil
	}

	// 使用索引映射快速查找位置
	idxMap, ok := m.indices[workerName]
	if !ok {
		return nil
	}

	idx, exists := idxMap[id]
	if !exists {
		return nil
	}

	// 删除找到的项
	if idx < len(q)-1 {
		// 如果不是最后一项，需要移动最后一项到当前位置
		lastItem := q[len(q)-1]
		q[idx] = lastItem
		// 更新最后一项的索引
		idxMap[lastItem.ID] = idx
	}

	// 从队列中移除最后一项
	m.queues[workerName] = q[:len(q)-1]
	// 从索引映射中删除
	delete(idxMap, id)

	return nil
}

// Count 获取队列中的项目数量
// Count gets the number of items in the queue
func (m *MemoryQueue) Count(workerName string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	q, ok := m.queues[workerName]
	if !ok {
		return 0
	}

	return len(q)
}

// List 获取分页的队列项列表
// List gets a paginated list of queue items
func (m *MemoryQueue) List(workerName string, page int, limit int) []*queue.QueueItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	q, ok := m.queues[workerName]
	if !ok || len(q) == 0 {
		return []*queue.QueueItem{}
	}

	start := (page - 1) * limit
	if start >= len(q) {
		return []*queue.QueueItem{}
	}

	end := start + limit
	if end > len(q) {
		end = len(q)
	}

	return q[start:end]
}

// Close 关闭队列
// Close the queue
func (m *MemoryQueue) Close() error {
	return nil
}
