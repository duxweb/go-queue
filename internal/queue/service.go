package queue

// QueueService 队列服务接口
// QueueService interface
type QueueService interface {
	// 弹出队列数据
	// Pop queue data
	Pop(queueName string, num int) []*QueueItem
	// 添加队列数据
	// Add queue data
	Add(queueName string, queue *QueueItem) error
	// 删除队列数据
	// Delete queue data
	Del(queueName string, id string) error
	// 获取队列数据数量
	// Get queue data count
	Count(queueName string) int
	// 获取队列总量
	// Get queue list
	List(queueName string, page int, limit int) []*QueueItem
}
