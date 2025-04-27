package queue

import (
	"context"
	"time"
)

type Queue interface {
	// 初始化队列工作器
	// Initialize queue worker
	Worker(queueName string)

	// 启动队列处理
	// Start queue processing
	Start() error

	// 注册队列处理函数
	// Register queue handler function
	Register(queueName, name string, callback func(ctx context.Context, params []byte) error) error

	// 添加队列任务
	// Add queue task
	Add(queueName string, add QueueConfig) (id string, err error)

	// 添加延迟队列任务
	// Add delayed queue task
	AddDelay(queueName string, add QueueDelayConfig) (id string, err error)

	// 删除队列任务
	// Delete queue task
	Del(queueName string, id string) error

	// 获取队列任务列表
	// Get queue task list
	List(queueName string, page int, limit int) (data []QueueItem, count int64, err error)

	// 获取所有队列名称
	// Get all queue names
	Names() []string

	// 关闭队列
	// Close queue
	Close() error
}

type QueueItem struct {
	ID          string    `json:"id"`
	QueueName   string    `json:"queue_name"`
	HandlerName string    `json:"handler_name"`
	Params      []byte    `json:"params"`
	CreatedAt   time.Time `json:"created_at"`
	RunAt       time.Time `json:"run_at"`
	Retried     int       `json:"retried"`
}

type QueueConfig struct {
	HandlerName string
	Params      []byte
}

type QueueDelayConfig struct {
	QueueConfig
	Delay time.Duration
}
