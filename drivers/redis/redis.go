package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/duxweb/go-queue"
	"github.com/redis/go-redis/v9"
)

// 键格式常量
const (
	// 队列项的哈希表键前缀
	queueItemPrefix = "queue:item:"
	// 队列中的有序集合键（按时间排序）
	queueTimeSortedSet = "queue:time:"
	// 队列列表键前缀
	queueListPrefix = "queue:list:"
)

// RedisQueue 是 Redis 队列驱动实现
type RedisQueue struct {
	client  *redis.Client
	options *Options
	ctx     context.Context
}

// Options Redis 队列选项
type Options struct {
	// Redis 配置
	Addr     string
	Password string
	DB       int
	// 可选: 使用现有的 Redis 客户端
	Client *redis.Client
	// 连接池配置
	PoolSize     int
	MinIdleConns int
	// 命令超时
	Timeout time.Duration
}

// DefaultOptions 返回默认选项
func DefaultOptions() *Options {
	return &Options{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		Timeout:      time.Second * 5,
	}
}

// New 创建新的 Redis 队列实例
// opts: Redis 选项，传 nil 则使用默认值
func New(opts *Options) (*RedisQueue, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	var client *redis.Client
	ctx := context.Background()

	// 使用提供的客户端或创建新的
	if opts.Client != nil {
		client = opts.Client
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         opts.Addr,
			Password:     opts.Password,
			DB:           opts.DB,
			PoolSize:     opts.PoolSize,
			MinIdleConns: opts.MinIdleConns,
		})
	}

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("无法连接到 Redis: %w", err)
	}

	return &RedisQueue{
		client:  client,
		options: opts,
		ctx:     ctx,
	}, nil
}

// 获取项目主键
func getItemKey(itemID string) string {
	return queueItemPrefix + itemID
}

// 获取队列时间有序集合键
func getQueueTimeKey(workerName string) string {
	return queueTimeSortedSet + workerName
}

// 获取队列列表键
func getQueueListKey(workerName string) string {
	return queueListPrefix + workerName
}

// Pop 从队列中弹出指定数量的项
func (r *RedisQueue) Pop(workerName string, num int) []*queue.QueueItem {
	if num <= 0 {
		return []*queue.QueueItem{}
	}

	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	// 当前时间戳（毫秒）
	now := float64(time.Now().UnixNano()) / 1e6

	// 获取准备运行的任务（按分数/时间排序）
	queueTimeKey := getQueueTimeKey(workerName)

	// 使用 ZRANGEBYSCORE 获取当前可执行的任务（分数/时间 <= 当前时间）
	ids, err := r.client.ZRangeByScore(ctx, queueTimeKey, &redis.ZRangeBy{
		Min:    "0",
		Max:    strconv.FormatFloat(now, 'f', 0, 64),
		Offset: 0,
		Count:  int64(num),
	}).Result()

	if err != nil || len(ids) == 0 {
		return []*queue.QueueItem{}
	}

	// 从 Redis 获取这些项目的详细信息并从队列中删除它们
	pipeline := r.client.Pipeline()

	// 准备获取详细信息的命令
	getCommands := make([]*redis.StringCmd, len(ids))
	for i, id := range ids {
		getCommands[i] = pipeline.Get(ctx, getItemKey(id))
		// 从时间有序集合中移除
		pipeline.ZRem(ctx, queueTimeKey, id)
		// 从列表键中移除
		pipeline.SRem(ctx, getQueueListKey(workerName), id)
	}

	// 执行管道命令
	_, err = pipeline.Exec(ctx)
	if err != nil {
		return []*queue.QueueItem{}
	}

	// 处理结果
	var items []*queue.QueueItem
	for _, cmd := range getCommands {
		data, err := cmd.Result()
		if err != nil {
			continue
		}

		var item queue.QueueItem
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}

		items = append(items, &item)

		// 删除项目
		r.client.Del(ctx, getItemKey(item.ID))
	}

	return items
}

// Add 添加队列项
func (r *RedisQueue) Add(workerName string, item *queue.QueueItem) error {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	// 序列化队列项
	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("序列化队列项失败: %w", err)
	}

	// 运行时间的时间戳（毫秒）
	runTime := float64(item.RunAt.UnixNano()) / 1e6

	// 使用管道进行原子操作
	pipeline := r.client.Pipeline()

	// 存储队列项
	pipeline.Set(ctx, getItemKey(item.ID), data, 0)

	// 添加到时间有序集合
	pipeline.ZAdd(ctx, getQueueTimeKey(workerName), redis.Z{
		Score:  runTime,
		Member: item.ID,
	})

	// 添加到队列列表（用于计数和列表操作）
	pipeline.SAdd(ctx, getQueueListKey(workerName), item.ID)

	_, err = pipeline.Exec(ctx)
	return err
}

// Del 从队列中删除指定的项
func (r *RedisQueue) Del(workerName string, id string) error {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	// 使用管道进行原子操作
	pipeline := r.client.Pipeline()

	// 从时间有序集合中删除
	pipeline.ZRem(ctx, getQueueTimeKey(workerName), id)

	// 从队列列表中删除
	pipeline.SRem(ctx, getQueueListKey(workerName), id)

	// 删除队列项
	pipeline.Del(ctx, getItemKey(id))

	_, err := pipeline.Exec(ctx)
	return err
}

// Count 获取队列中的项目数量
func (r *RedisQueue) Count(workerName string) int {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	count, err := r.client.SCard(ctx, getQueueListKey(workerName)).Result()
	if err != nil {
		return 0
	}

	return int(count)
}

// List 获取分页的队列项列表
func (r *RedisQueue) List(workerName string, page int, limit int) []*queue.QueueItem {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	// 获取所有 ID
	ids, err := r.client.SMembers(ctx, getQueueListKey(workerName)).Result()
	if err != nil || len(ids) == 0 {
		return []*queue.QueueItem{}
	}

	// 排序和分页
	start := (page - 1) * limit
	end := start + limit
	if start >= len(ids) {
		return []*queue.QueueItem{}
	}
	if end > len(ids) {
		end = len(ids)
	}

	// 根据 ID 获取队列项
	pipeline := r.client.Pipeline()
	getCommands := make([]*redis.StringCmd, 0, end-start)
	for i := start; i < end; i++ {
		if i < len(ids) {
			getCommands = append(getCommands, pipeline.Get(ctx, getItemKey(ids[i])))
		}
	}

	_, err = pipeline.Exec(ctx)
	if err != nil {
		return []*queue.QueueItem{}
	}

	// 处理结果
	var items []*queue.QueueItem
	for _, cmd := range getCommands {
		data, err := cmd.Result()
		if err != nil {
			continue
		}

		var item queue.QueueItem
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}

		items = append(items, &item)
	}

	return items
}

// Close 关闭队列
func (r *RedisQueue) Close() error {
	return r.client.Close()
}

// WithClient 使用现有的 Redis 客户端
func WithClient(client *redis.Client) *Options {
	opts := DefaultOptions()
	opts.Client = client
	return opts
}

// CleanupQueue 清理队列（仅用于测试/管理）
func (r *RedisQueue) CleanupQueue(workerName string) error {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	// 获取所有项目 ID
	ids, err := r.client.SMembers(ctx, getQueueListKey(workerName)).Result()
	if err != nil {
		return err
	}

	// 使用管道批量删除
	pipeline := r.client.Pipeline()

	// 删除队列列表
	pipeline.Del(ctx, getQueueListKey(workerName))

	// 删除时间有序集合
	pipeline.Del(ctx, getQueueTimeKey(workerName))

	// 删除所有队列项
	for _, id := range ids {
		pipeline.Del(ctx, getItemKey(id))
	}

	_, err = pipeline.Exec(ctx)
	return err
}

// FlushAll 清空所有队列数据（仅用于测试/管理）
func (r *RedisQueue) FlushAll() error {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	// 获取所有以 "queue:" 开头的键
	keys, err := r.client.Keys(ctx, "queue:*").Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}

	return nil
}
