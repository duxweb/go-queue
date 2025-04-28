package queue

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Service 队列服务实现
type Service struct {
	drivers  map[string]QueueDriver
	workers  map[string]*Worker
	handlers map[string]HandlerFunc
	ctx      context.Context
}

// Config 队列服务配置
type Config struct {
	Context context.Context
}

// New 创建新的服务实例
// New create a new service instance
func New(config *Config) (*Service, error) {
	return &Service{
		drivers:  make(map[string]QueueDriver),
		workers:  make(map[string]*Worker),
		handlers: make(map[string]HandlerFunc),
		ctx:      config.Context,
	}, nil
}

// RegisterDriver 注册队列驱动
// RegisterDriver register queue driver
func (s *Service) RegisterDriver(deviceName string, driver QueueDriver) {
	s.drivers[deviceName] = driver
}

// RegisterWorker 注册工作队列
// RegisterWorker register worker
func (q *Service) RegisterWorker(workerName string, config *WorkerConfig) error {
	worker := NewWorker(config)

	if _, ok := q.drivers[config.DeviceName]; !ok {
		return errors.New("queue driver not found")
	}
	worker.Name = workerName
	worker.Driver = q.drivers[config.DeviceName]
	q.workers[workerName] = worker
	return nil
}

// RegisterHandler 注册处理器
// RegisterHandler register handler
func (q *Service) RegisterHandler(handlerName string, handler HandlerFunc) {
	q.handlers[handlerName] = handler
}

// Add 添加任务
// Add task
func (q *Service) Add(workerName string, config *QueueConfig) (string, error) {
	return q.AddDelay(workerName, &QueueDelayConfig{
		QueueConfig: *config,
		Delay:       0,
	})
}

// AddDelay 添加延迟任务
// AddDelay add delayed task
func (q *Service) AddDelay(workerName string, config *QueueDelayConfig) (string, error) {
	worker, ok := q.workers[workerName]
	if !ok {
		return "", errors.New("worker not found")
	}

	id := uuid.New().String()
	err := worker.Driver.Add(workerName, &QueueItem{
		ID:          id,
		WorkerName:  workerName,
		HandlerName: config.HandlerName,
		Params:      config.Params,
		CreatedAt:   time.Now(),
		RunAt:       time.Now().Add(config.Delay),
		Retried:     0,
	})

	if err != nil {
		return "", err
	}

	return id, nil
}

// Names 获取所有工作池名称
// Names get all worker pool names
func (q *Service) Names() []string {
	names := make([]string, 0, len(q.workers))
	for name := range q.workers {
		names = append(names, name)
	}
	return names
}

// Start 启动所有工作池
// Start all worker pools
func (q *Service) Start() error {
	for _, worker := range q.workers {
		go worker.Start(q.ctx, q.handlers)
	}
	return nil
}

// Pause 暂停所有工作池
// Pause all worker pools
func (q *Service) Pause() error {
	for _, worker := range q.workers {
		worker.Pause()
	}

	return nil
}

// Resume 恢复所有工作池
// Resume all worker pools
func (q *Service) Resume() error {
	for _, worker := range q.workers {
		worker.Resume()
	}
	return nil
}

// Stop 停止所有工作池
// Stop all worker pools
func (q *Service) Stop() error {
	for _, worker := range q.workers {
		worker.Stop()
	}

	for _, driver := range q.drivers {
		driver.Close()
	}
	return nil
}

// List 获取队列数据
// List get queue data
func (q *Service) List(workerName string, page int, limit int) ([]*QueueItem, error) {
	worker, ok := q.workers[workerName]
	if !ok {
		return nil, errors.New("worker not found")
	}

	return worker.Driver.List(workerName, page, limit), nil
}

// Count 获取队列数据数量
// Count get queue data count
func (q *Service) Count(workerName string) (int, error) {
	worker, ok := q.workers[workerName]
	if !ok {
		return 0, errors.New("worker not found")
	}

	return worker.Driver.Count(workerName), nil
}

// Del 删除队列数据
// Del delete queue data
func (q *Service) Del(workerName string, id string) error {
	worker, ok := q.workers[workerName]
	if !ok {
		return errors.New("worker not found")
	}

	return worker.Driver.Del(workerName, id)
}

// GetTotal 获取队列统计
// GetTotal get queue statistics
func (q *Service) GetTotal(workerName string) (map[string]any, error) {
	worker, ok := q.workers[workerName]
	if !ok {
		return nil, errors.New("queue not found")
	}
	return worker.GetTotal(), nil
}

// GetWorker 获取工作器
// GetWorker get worker
func (q *Service) GetWorker(workerName string) *Worker {
	return q.workers[workerName]
}
