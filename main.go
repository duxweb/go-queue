package queue

import (
	"context"
	"errors"
	"time"

	"github.com/duxweb/go-queue/internal/queue"
	"github.com/duxweb/go-queue/internal/worker"
)

// Service 队列服务实现
type Service struct {
	services map[string]queue.QueueService
	workers  map[string]worker.Worker
	handlers map[string]worker.HandlerFunc
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
		services: make(map[string]queue.QueueService),
		workers:  make(map[string]worker.Worker),
		handlers: make(map[string]worker.HandlerFunc),
		ctx:      config.Context,
	}, nil
}

// RegisterService 注册队列服务
// RegisterService register queue service
func (s *Service) RegisterService(serviceName string, driver queue.QueueService) {
	s.services[serviceName] = driver
}

// RegisterWorker 注册工作队列
// RegisterWorker register worker
func (q *Service) RegisterWorker(queueName string, worker worker.Worker) {
	q.workers[queueName] = worker
}

// RegisterHandler 注册处理器
// RegisterHandler register handler
func (q *Service) RegisterHandler(handlerName string, handler worker.HandlerFunc) {
	q.handlers[handlerName] = handler
}

// Add 添加任务
// Add task
func (q *Service) Add(queueName string, config *queue.QueueConfig) error {
	return q.AddDelay(queueName, &queue.QueueDelayConfig{
		QueueConfig: *config,
		Delay:       0,
	})
}

// AddDelay 添加延迟任务
// AddDelay add delayed task
func (q *Service) AddDelay(queueName string, config *queue.QueueDelayConfig) error {
	queueService, ok := q.services[queueName]
	if !ok {
		return errors.New("queue not found")
	}

	return queueService.Add(queueName, &queue.QueueItem{
		QueueName:   queueName,
		HandlerName: config.HandlerName,
		Params:      config.Params,
		CreatedAt:   time.Now().Add(config.Delay),
		Retried:     0,
	})
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
	for name, worker := range q.workers {
		serviceName := name // 使用本地变量避免闭包问题
		go worker.Start(q.ctx, q.services[serviceName], q.handlers)
	}
	return nil
}

// List 获取队列数据
// List get queue data
func (q *Service) List(queueName string, page int, limit int) ([]*queue.QueueItem, error) {
	queueService, ok := q.services[queueName]
	if !ok {
		return nil, errors.New("queue not found")
	}

	return queueService.List(queueName, page, limit), nil
}

// Count 获取队列数据数量
// Count get queue data count
func (q *Service) Count(queueName string) (int, error) {
	queueService, ok := q.services[queueName]
	if !ok {
		return 0, errors.New("queue not found")
	}

	return queueService.Count(queueName), nil
}

// Del 删除队列数据
// Del delete queue data
func (q *Service) Del(queueName string, id string) error {
	queueService, ok := q.services[queueName]
	if !ok {
		return errors.New("queue not found")
	}
	return queueService.Del(queueName, id)
}

// GetTotal 获取队列统计
// GetTotal get queue statistics
func (q *Service) GetTotal(queueName string) (map[string]any, error) {
	worker, ok := q.workers[queueName]
	if !ok {
		return nil, errors.New("queue not found")
	}
	return worker.GetTotal(), nil
}
