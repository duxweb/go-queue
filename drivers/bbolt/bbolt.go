package bbolt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/duxweb/go-queue"
	"go.etcd.io/bbolt"
)

// 存储桶名称
const (
	// 队列项的存储桶
	queueBucket = "queue_items"
)

// BoltQueue 是BBolt队列驱动实现
type BoltQueue struct {
	db         *bbolt.DB
	dbPath     string
	bucketName []byte
	options    *Options
}

// Options BBolt队列选项
type Options struct {
	// 文件模式
	FileMode os.FileMode
	// 打开超时
	Timeout time.Duration
	// 选项
	Options *bbolt.Options
}

// DefaultOptions 返回默认选项
func DefaultOptions() *Options {
	return &Options{
		FileMode: 0600,
		Timeout:  time.Second * 1,
		Options:  nil,
	}
}

// New 创建新的BBolt队列实例
// path: 数据库文件路径
// opts: 可选配置，传nil则使用默认值
func New(path string, opts *Options) (*BoltQueue, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// 确保目录存在
	dbPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("无法获取绝对路径: %w", err)
	}

	// 打开数据库
	db, err := bbolt.Open(dbPath, opts.FileMode, &bbolt.Options{
		Timeout: opts.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("无法打开BBolt数据库: %w", err)
	}

	// 创建队列存储桶
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(queueBucket))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("无法创建队列存储桶: %w", err)
	}

	return &BoltQueue{
		db:         db,
		dbPath:     dbPath,
		bucketName: []byte(queueBucket),
		options:    opts,
	}, nil
}

// 获取队列键前缀
func getQueueKey(workerName string) []byte {
	return []byte(fmt.Sprintf("queue:%s:", workerName))
}

// 获取队列项键
func getItemKey(workerName string, itemID string) []byte {
	return []byte(fmt.Sprintf("queue:%s:item:%s", workerName, itemID))
}

// Pop 从队列中弹出指定数量的项
func (b *BoltQueue) Pop(workerName string, num int) []*queue.QueueItem {
	if num <= 0 {
		return []*queue.QueueItem{}
	}

	var items []*queue.QueueItem
	now := time.Now()

	// 开始读写事务
	err := b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		if bucket == nil {
			return nil
		}

		// 队列键前缀
		prefix := getQueueKey(workerName)

		// 游标遍历所有匹配项
		c := bucket.Cursor()
		count := 0

		for k, v := c.Seek(prefix); k != nil && count < num && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			var item queue.QueueItem
			if err := json.Unmarshal(v, &item); err != nil {
				continue // 跳过无效项
			}

			// 检查是否可以执行（当前时间大于或等于队列项的创建时间）
			if now.After(item.CreatedAt) || now.Equal(item.CreatedAt) {
				items = append(items, &item)
				// 从队列中删除此项
				if err := bucket.Delete(k); err != nil {
					return err
				}
				count++
			}
		}

		return nil
	})

	if err != nil {
		// 发生错误，返回空数组
		return []*queue.QueueItem{}
	}

	return items
}

// Add 添加队列项
func (b *BoltQueue) Add(workerName string, item *queue.QueueItem) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		if bucket == nil {
			return fmt.Errorf("队列存储桶不存在")
		}

		// 序列化队列项
		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("序列化队列项失败: %w", err)
		}

		// 保存队列项
		key := getItemKey(workerName, item.ID)
		if err := bucket.Put(key, data); err != nil {
			return fmt.Errorf("保存队列项失败: %w", err)
		}

		return nil
	})
}

// Del 从队列中删除指定的项
func (b *BoltQueue) Del(workerName string, id string) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		if bucket == nil {
			return nil
		}

		// 删除队列项
		key := getItemKey(workerName, id)
		return bucket.Delete(key)
	})
}

// Count 获取队列中的项目数量
func (b *BoltQueue) Count(workerName string) int {
	var count int

	b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		if bucket == nil {
			return nil
		}

		// 队列键前缀
		prefix := getQueueKey(workerName)

		// 统计所有匹配的键
		c := bucket.Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			count++
		}

		return nil
	})

	return count
}

// List 获取分页的队列项列表
func (b *BoltQueue) List(workerName string, page int, limit int) []*queue.QueueItem {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	var items []*queue.QueueItem
	start := (page - 1) * limit

	b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(b.bucketName)
		if bucket == nil {
			return nil
		}

		// 队列键前缀
		prefix := getQueueKey(workerName)

		// 获取所有匹配项
		c := bucket.Cursor()
		currentIdx := 0
		count := 0

		for k, v := c.Seek(prefix); k != nil && count < limit && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			// 分页逻辑
			if currentIdx < start {
				currentIdx++
				continue
			}

			var item queue.QueueItem
			if err := json.Unmarshal(v, &item); err != nil {
				continue // 跳过无效项
			}

			items = append(items, &item)
			count++
		}

		return nil
	})

	return items
}

// Close 关闭队列数据库
func (b *BoltQueue) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}
