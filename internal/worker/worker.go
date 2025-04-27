package worker

import (
	"context"
	"time"

	"github.com/duxweb/go-queue/internal/queue"
)

// HandlerFunc 处理器函数定义
// HandlerFunc handler function definition
type HandlerFunc func(ctx context.Context, params []byte) error

// WorkerStatus 工作线程状态
// WorkerStatus worker status
type WorkerStatus int

const (
	StatusStopped WorkerStatus = iota
	StatusRunning
	StatusPaused
)

// String 将工作状态转换为字符串表示
// String convert worker status to string representation
func (s WorkerStatus) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusRunning:
		return "running"
	case StatusPaused:
		return "paused"
	default:
		return "unknown"
	}
}

// Worker 工作线程接口
// Worker interface
type Worker interface {
	// Start 启动工作线程
	// Start the worker
	Start(ctx context.Context, service queue.QueueService, handlers map[string]HandlerFunc) error

	// Pause 暂停工作线程
	// Pause the worker
	Pause() error

	// Resume 恢复工作线程
	// Resume the worker
	Resume() error

	// Stop 停止工作线程
	// Stop the worker
	Stop() error

	// GetStatus 获取工作线程状态
	// GetStatus get worker status
	GetStatus() WorkerStatus

	// GetTotal 获取工作线程统计数据
	// GetTotal get worker statistics
	GetTotal() map[string]any
}

// WorkerConfig 工作线程配置
// WorkerConfig worker configuration
type WorkerConfig struct {
	ServiceName string
	Num         int
	Interval    time.Duration
	Retry       int
	RetryDelay  time.Duration
	Timeout     time.Duration
	SuccessFunc func(item *queue.QueueItem)
	FailFunc    func(item *queue.QueueItem, err error)
}
