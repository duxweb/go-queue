package redis

import (
	"testing"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// 测试用的 Redis 配置
var testRedisOptions = &Options{
	Addr:     "localhost:6379",
	Password: "",
	DB:       15, // 使用临时的 DB 15 以避免干扰生产数据
	Timeout:  time.Second * 2,
}

// 设置测试 Redis 队列
func setupTestRedisQueue(t *testing.T) *RedisQueue {
	queue, err := New(testRedisOptions)
	if err != nil {
		t.Skipf("跳过 Redis 测试：%v", err)
		return nil
	}

	// 清空测试数据库
	err = queue.FlushAll()
	assert.NoError(t, err)

	return queue
}

// 测试 Redis 队列是否实现了队列驱动接口
func TestRedisQueueImplementsQueueDriver(t *testing.T) {
	var _ queue.QueueDriver = (*RedisQueue)(nil)
}

// 测试 Redis 队列基本操作
func TestRedisQueueBasicOperations(t *testing.T) {
	redisQueue := setupTestRedisQueue(t)
	if redisQueue == nil {
		return
	}
	defer redisQueue.Close()

	// 测试工作线程名称
	workerName := "test-worker"

	// 测试初始计数
	count := redisQueue.Count(workerName)
	assert.Equal(t, 0, count, "初始队列应为空")

	// 创建测试队列项
	item1 := &queue.QueueItem{
		ID:          uuid.New().String(),
		WorkerName:  workerName,
		HandlerName: "test-handler",
		Params:      []byte(`{"test":"data1"}`),
		CreatedAt:   time.Now(),
		RunAt:       time.Now(),
		Retried:     0,
	}
	item2 := &queue.QueueItem{
		ID:          uuid.New().String(),
		WorkerName:  workerName,
		HandlerName: "test-handler",
		Params:      []byte(`{"test":"data2"}`),
		CreatedAt:   time.Now(),
		RunAt:       time.Now(),
		Retried:     0,
	}

	// 测试添加
	err := redisQueue.Add(workerName, item1)
	assert.NoError(t, err)
	err = redisQueue.Add(workerName, item2)
	assert.NoError(t, err)

	// 测试计数
	count = redisQueue.Count(workerName)
	assert.Equal(t, 2, count, "队列中应有两个项目")

	// 测试列表
	items := redisQueue.List(workerName, 1, 10)
	assert.Equal(t, 2, len(items), "列表应返回两个项目")

	// 测试弹出
	poppedItems := redisQueue.Pop(workerName, 1)
	assert.Equal(t, 1, len(poppedItems), "应该弹出一个项目")

	// 再次测试计数
	count = redisQueue.Count(workerName)
	assert.Equal(t, 1, count, "弹出一个后，队列中应有一个项目")

	// 再次获取列表以获取剩余的项目
	remainingItems := redisQueue.List(workerName, 1, 10)
	assert.Equal(t, 1, len(remainingItems), "列表应返回一个剩余项目")

	// 测试删除剩余的项目
	err = redisQueue.Del(workerName, remainingItems[0].ID)
	assert.NoError(t, err)

	// 最终测试计数
	count = redisQueue.Count(workerName)
	assert.Equal(t, 0, count, "删除后，队列应为空")
}

// 测试 Redis 队列延迟项目
func TestRedisQueueDelayedItems(t *testing.T) {
	redisQueue := setupTestRedisQueue(t)
	if redisQueue == nil {
		return
	}
	defer redisQueue.Close()

	// 测试工作线程名称
	workerName := "test-worker-delayed"

	// 创建延迟的测试队列项
	futureTime := time.Now().Add(time.Second * 2)
	delayedItem := &queue.QueueItem{
		ID:          uuid.New().String(),
		WorkerName:  workerName,
		HandlerName: "test-handler",
		Params:      []byte(`{"test":"delayed"}`),
		CreatedAt:   time.Now(),
		RunAt:       futureTime,
		Retried:     0,
	}

	// 测试添加
	err := redisQueue.Add(workerName, delayedItem)
	assert.NoError(t, err)

	// 尝试立即弹出，不应该返回任何项目
	poppedItems := redisQueue.Pop(workerName, 1)
	assert.Equal(t, 0, len(poppedItems), "延迟项目不应该被弹出")

	// 等待延迟时间过去
	time.Sleep(time.Second * 3)

	// 再次尝试弹出，应该返回延迟项目
	poppedItems = redisQueue.Pop(workerName, 1)
	assert.Equal(t, 1, len(poppedItems), "延迟项目应该被弹出")
	assert.Equal(t, delayedItem.ID, poppedItems[0].ID)
}

// 测试 Redis 队列分页
func TestRedisQueuePagination(t *testing.T) {
	redisQueue := setupTestRedisQueue(t)
	if redisQueue == nil {
		return
	}
	defer redisQueue.Close()

	// 测试工作线程名称
	workerName := "test-worker-pagination"

	// 添加多个项目
	for i := 0; i < 15; i++ {
		item := &queue.QueueItem{
			ID:          uuid.New().String(),
			WorkerName:  workerName,
			HandlerName: "test-handler",
			Params:      []byte(`{"test":"pagination"}`),
			CreatedAt:   time.Now(),
			RunAt:       time.Now(),
			Retried:     0,
		}
		err := redisQueue.Add(workerName, item)
		assert.NoError(t, err)
	}

	// 测试计数
	count := redisQueue.Count(workerName)
	assert.Equal(t, 15, count, "队列中应有15个项目")

	// 测试分页 - 第一页
	items := redisQueue.List(workerName, 1, 5)
	assert.Equal(t, 5, len(items), "第一页应返回5个项目")

	// 测试分页 - 第二页
	items = redisQueue.List(workerName, 2, 5)
	assert.Equal(t, 5, len(items), "第二页应返回5个项目")

	// 测试分页 - 第三页
	items = redisQueue.List(workerName, 3, 5)
	assert.Equal(t, 5, len(items), "第三页应返回5个项目")

	// 测试分页 - 第四页（超出范围）
	items = redisQueue.List(workerName, 4, 5)
	assert.Equal(t, 0, len(items), "第四页应返回0个项目")
}

// 测试 Redis 队列使用自定义客户端
func TestRedisQueueWithCustomClient(t *testing.T) {
	// 创建自定义 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15,
	})

	// 使用自定义客户端创建队列
	opts := WithClient(client)
	redisQueue, err := New(opts)
	if err != nil {
		t.Skipf("跳过 Redis 测试：%v", err)
		return
	}
	defer redisQueue.Close()

	// 测试队列操作
	workerName := "test-worker-custom-client"

	// 创建测试队列项
	item := &queue.QueueItem{
		ID:          uuid.New().String(),
		WorkerName:  workerName,
		HandlerName: "test-handler",
		Params:      []byte(`{"test":"custom-client"}`),
		CreatedAt:   time.Now(),
		RunAt:       time.Now(),
		Retried:     0,
	}

	// 测试添加
	err = redisQueue.Add(workerName, item)
	assert.NoError(t, err)

	// 测试计数
	count := redisQueue.Count(workerName)
	assert.Equal(t, 1, count, "队列中应有一个项目")
}

// 测试 Redis 队列选项
func TestRedisQueueOptions(t *testing.T) {
	// 测试默认选项
	opts := DefaultOptions()
	assert.Equal(t, "localhost:6379", opts.Addr)
	assert.Equal(t, "", opts.Password)
	assert.Equal(t, 0, opts.DB)
	assert.Equal(t, 10, opts.PoolSize)
	assert.Equal(t, 5, opts.MinIdleConns)
	assert.Equal(t, time.Second*5, opts.Timeout)
}
