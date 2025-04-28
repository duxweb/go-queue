package queue

// QueueDriver 队列服务接口
// QueueDriver interface
type QueueDriver interface {
	// 弹出队列数据
	// Pop queue data
	Pop(workerName string, num int) []*QueueItem
	// 添加队列数据
	// Add queue data
	Add(workerName string, queue *QueueItem) error
	// 删除队列数据
	// Delete queue data
	Del(workerName string, id string) error
	// 获取队列数据数量
	// Get queue data count
	Count(workerName string) int
	// 获取队列总量
	// Get queue list
	List(workerName string, page int, limit int) []*QueueItem

	// 关闭队列
	// Close queue
	Close() error
}
