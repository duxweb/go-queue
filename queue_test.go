package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 使用 mock 驱动来测试 Service 类的方法
type mockDriver struct {
	items map[string][]*QueueItem
}

func newMockDriver() *mockDriver {
	return &mockDriver{
		items: make(map[string][]*QueueItem),
	}
}

func (m *mockDriver) Pop(queueName string, num int) []*QueueItem {
	if len(m.items[queueName]) == 0 {
		return []*QueueItem{}
	}

	result := m.items[queueName][:num]
	m.items[queueName] = m.items[queueName][num:]
	return result
}

func (m *mockDriver) Add(queueName string, item *QueueItem) error {
	if _, ok := m.items[queueName]; !ok {
		m.items[queueName] = []*QueueItem{}
	}
	m.items[queueName] = append(m.items[queueName], item)
	return nil
}

func (m *mockDriver) Del(queueName string, id string) error {
	if _, ok := m.items[queueName]; !ok {
		return nil
	}

	for i, item := range m.items[queueName] {
		if item.ID == id {
			m.items[queueName] = append(m.items[queueName][:i], m.items[queueName][i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockDriver) Count(queueName string) int {
	return len(m.items[queueName])
}

func (m *mockDriver) List(queueName string, page int, limit int) []*QueueItem {
	if len(m.items[queueName]) == 0 {
		return []*QueueItem{}
	}

	start := (page - 1) * limit
	if start >= len(m.items[queueName]) {
		return []*QueueItem{}
	}

	end := start + limit
	if end > len(m.items[queueName]) {
		end = len(m.items[queueName])
	}

	return m.items[queueName][start:end]
}

func (m *mockDriver) Close() error {
	return nil
}

// 测试创建服务实例
func TestServiceNew(t *testing.T) {
	ctx := context.Background()
	service, err := New(&Config{
		Context: ctx,
	})

	assert.NoError(t, err, "创建服务实例应该成功")
	assert.NotNil(t, service, "服务实例不应为空")
	assert.NotNil(t, service.drivers, "驱动map不应为空")
	assert.NotNil(t, service.workers, "工作器map不应为空")
	assert.NotNil(t, service.handlers, "处理器map不应为空")
	assert.Equal(t, ctx, service.ctx, "上下文应该匹配")
}

// 测试注册驱动
func TestServiceRegisterDriver(t *testing.T) {
	service, _ := New(&Config{Context: context.Background()})

	// 注册模拟驱动
	mockDriver := newMockDriver()
	service.RegisterDriver("mock-driver", mockDriver)

	// 验证驱动已注册
	assert.Equal(t, mockDriver, service.drivers["mock-driver"], "驱动应该已注册")
}

// 测试注册工作器
func TestServiceRegisterWorker(t *testing.T) {
	service, _ := New(&Config{Context: context.Background()})

	// 注册模拟驱动
	mockDriver := newMockDriver()
	service.RegisterDriver("mock-driver", mockDriver)

	// 注册工作器
	err := service.RegisterWorker("test-worker", &WorkerConfig{
		ServiceName: "mock-driver",
		Num:         1,
		Interval:    time.Millisecond * 100,
	})

	assert.NoError(t, err, "注册工作器应该成功")
	assert.NotNil(t, service.workers["test-worker"], "工作器应该已注册")
	assert.Equal(t, "test-worker", service.workers["test-worker"].Name, "工作器名称应该匹配")
	assert.Equal(t, mockDriver, service.workers["test-worker"].Driver, "工作器驱动应该匹配")

	// 测试注册不存在的驱动
	err = service.RegisterWorker("invalid-worker", &WorkerConfig{
		ServiceName: "non-existent-driver",
	})
	assert.Error(t, err, "使用不存在的驱动注册工作器应该失败")
}

// 测试注册处理器
func TestServiceRegisterHandler(t *testing.T) {
	service, _ := New(&Config{Context: context.Background()})

	// 定义处理器函数
	handler := func(ctx context.Context, params []byte) error {
		return nil
	}

	// 注册处理器
	service.RegisterHandler("test-handler", handler)

	// 验证处理器已注册
	assert.NotNil(t, service.handlers["test-handler"], "处理器应该已注册")
}

// 测试获取工作池名称
func TestServiceNames(t *testing.T) {
	service, _ := New(&Config{Context: context.Background()})

	// 注册模拟驱动
	mockDriver := newMockDriver()
	service.RegisterDriver("mock-driver", mockDriver)

	// 注册工作器
	service.RegisterWorker("worker1", &WorkerConfig{
		ServiceName: "mock-driver",
		Num:         1,
	})

	service.RegisterWorker("worker2", &WorkerConfig{
		ServiceName: "mock-driver",
		Num:         1,
	})

	// 获取工作池名称
	names := service.Names()

	assert.Len(t, names, 2, "应该有2个工作池")
	assert.Contains(t, names, "worker1", "应该包含worker1")
	assert.Contains(t, names, "worker2", "应该包含worker2")
}

// 测试获取工作器
func TestServiceGetWorker(t *testing.T) {
	service, _ := New(&Config{Context: context.Background()})

	// 注册模拟驱动
	mockDriver := newMockDriver()
	service.RegisterDriver("mock-driver", mockDriver)

	// 注册工作器
	service.RegisterWorker("test-worker", &WorkerConfig{
		ServiceName: "mock-driver",
		Num:         1,
	})

	// 获取工作器
	worker := service.GetWorker("test-worker")
	assert.NotNil(t, worker, "应该返回工作器")
	assert.Equal(t, "test-worker", worker.Name, "工作器名称应该匹配")

	// 获取不存在的工作器
	worker = service.GetWorker("non-existent-worker")
	assert.Nil(t, worker, "不存在的工作器应该返回nil")
}
