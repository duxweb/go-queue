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
	Client   *redis.Client
	Addr     string
	Password string
	DB       int
	Prefix   string
	PoolSize int
	Timeout  time.Duration
}

// DefaultOptions 返回默认选项
// DefaultOptions returns default options
func DefaultOptions() *Options {
	return &Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		PoolSize: 0,
		Timeout:  time.Second * 5,
	}
}

// New 创建新的 Redis 队列实例
// New creates a new Redis queue instance
func New(opts *Options) (*RedisQueue, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	var client *redis.Client
	ctx := context.Background()

	if opts.Client != nil {
		client = opts.Client
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     opts.Addr,
			Password: opts.Password,
			DB:       opts.DB,
			PoolSize: opts.PoolSize,
		})
	}

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
func (r *RedisQueue) getItemKey(itemID string) string {
	return r.options.Prefix + queueItemPrefix + itemID
}

// 获取队列时间有序集合键
func (r *RedisQueue) getQueueTimeKey(workerName string) string {
	return r.options.Prefix + queueTimeSortedSet + workerName
}

// 获取队列列表键
func (r *RedisQueue) getQueueListKey(workerName string) string {
	return r.options.Prefix + queueListPrefix + workerName
}

// Pop 从队列中弹出指定数量的项
// Pop pops specified number of items from the queue
func (r *RedisQueue) Pop(workerName string, num int) []*queue.QueueItem {
	if num <= 0 {
		return []*queue.QueueItem{}
	}

	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	now := float64(time.Now().UnixNano()) / 1e6

	queueTimeKey := r.getQueueTimeKey(workerName)
	queueListKey := r.getQueueListKey(workerName)

	script := `
		local ids = redis.call('ZRANGEBYSCORE', KEYS[1], '0', ARGV[1], 'LIMIT', 0, ARGV[2])
		if #ids > 0 then
			for i, id in ipairs(ids) do
				redis.call('ZREM', KEYS[1], id)
				redis.call('SREM', KEYS[2], id)
			end
		end
		return ids
	`

	ids, err := r.client.Eval(ctx, script, []string{queueTimeKey, queueListKey},
		strconv.FormatFloat(now, 'f', 0, 64), num).Result()

	if err != nil || ids == nil {
		return []*queue.QueueItem{}
	}

	idList, ok := ids.([]interface{})
	if !ok || len(idList) == 0 {
		return []*queue.QueueItem{}
	}

	pipeline := r.client.Pipeline()
	getCommands := make([]*redis.StringCmd, len(idList))
	delCommands := make([]*redis.IntCmd, len(idList))

	for i, idInterface := range idList {
		id, ok := idInterface.(string)
		if !ok {
			continue
		}
		itemKey := r.getItemKey(id)
		getCommands[i] = pipeline.Get(ctx, itemKey)
		delCommands[i] = pipeline.Del(ctx, itemKey)
	}

	_, err = pipeline.Exec(ctx)
	if err != nil {
		return []*queue.QueueItem{}
	}

	var items []*queue.QueueItem
	for _, cmd := range getCommands {
		if cmd == nil {
			continue
		}

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

// Add 添加队列项
// Add adds an item to the queue
func (r *RedisQueue) Add(workerName string, item *queue.QueueItem) error {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("序列化队列项失败: %w", err)
	}

	runTime := float64(item.RunAt.UnixNano()) / 1e6

	pipeline := r.client.Pipeline()

	pipeline.Set(ctx, r.getItemKey(item.ID), data, 0)

	pipeline.ZAdd(ctx, r.getQueueTimeKey(workerName), redis.Z{
		Score:  runTime,
		Member: item.ID,
	})

	pipeline.SAdd(ctx, r.getQueueListKey(workerName), item.ID)

	_, err = pipeline.Exec(ctx)
	return err
}

// Del 从队列中删除指定的项
// Del removes the specified item from the queue
func (r *RedisQueue) Del(workerName string, id string) error {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	pipeline := r.client.Pipeline()

	pipeline.ZRem(ctx, r.getQueueTimeKey(workerName), id)

	pipeline.SRem(ctx, r.getQueueListKey(workerName), id)

	pipeline.Del(ctx, r.getItemKey(id))

	_, err := pipeline.Exec(ctx)
	return err
}

// Count 获取队列中的项目数量
// Count gets the number of items in the queue
func (r *RedisQueue) Count(workerName string) int {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	count, err := r.client.SCard(ctx, r.getQueueListKey(workerName)).Result()
	if err != nil {
		return 0
	}

	return int(count)
}

// List 获取分页的队列项列表
// List gets a paginated list of queue items
func (r *RedisQueue) List(workerName string, page int, limit int) []*queue.QueueItem {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	ids, err := r.client.SMembers(ctx, r.getQueueListKey(workerName)).Result()
	if err != nil || len(ids) == 0 {
		return []*queue.QueueItem{}
	}

	start := (page - 1) * limit
	end := start + limit
	if start >= len(ids) {
		return []*queue.QueueItem{}
	}
	if end > len(ids) {
		end = len(ids)
	}

	pipeline := r.client.Pipeline()
	getCommands := make([]*redis.StringCmd, 0, end-start)
	for i := start; i < end; i++ {
		if i < len(ids) {
			getCommands = append(getCommands, pipeline.Get(ctx, r.getItemKey(ids[i])))
		}
	}

	_, err = pipeline.Exec(ctx)
	if err != nil {
		return []*queue.QueueItem{}
	}

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
// Close closes the queue
func (r *RedisQueue) Close() error {
	return r.client.Close()
}

// WithClient 使用现有的 Redis 客户端
// WithClient uses an existing Redis client
func WithClient(client *redis.Client) *Options {
	opts := DefaultOptions()
	opts.Client = client
	return opts
}

// CleanupQueue 清理队列（仅用于测试/管理）
// CleanupQueue cleans up the queue (for testing/management only)
func (r *RedisQueue) CleanupQueue(workerName string) error {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	ids, err := r.client.SMembers(ctx, r.getQueueListKey(workerName)).Result()
	if err != nil {
		return err
	}

	pipeline := r.client.Pipeline()

	pipeline.Del(ctx, r.getQueueListKey(workerName))

	pipeline.Del(ctx, r.getQueueTimeKey(workerName))

	for _, id := range ids {
		pipeline.Del(ctx, r.getItemKey(id))
	}

	_, err = pipeline.Exec(ctx)
	return err
}

// FlushAll 清空所有队列数据（仅用于测试/管理）
// FlushAll flushes all queue data (for testing/management only)
func (r *RedisQueue) FlushAll() error {
	ctx, cancel := context.WithTimeout(r.ctx, r.options.Timeout)
	defer cancel()

	keys, err := r.client.Keys(ctx, r.options.Prefix+"queue:*").Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}

	return nil
}
