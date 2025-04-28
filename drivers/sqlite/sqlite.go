package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/duxweb/go-queue"
	_ "modernc.org/sqlite"
)

// SQLiteQueue 是基于SQLite的队列实现
type SQLiteQueue struct {
	db     *sql.DB
	dbPath string
}

// SQLiteOptions SQLite配置选项
type SQLiteOptions struct {
	DBPath string // 数据库文件路径
}

// DefaultOptions 默认配置选项
func DefaultOptions() *SQLiteOptions {
	return &SQLiteOptions{
		DBPath: "queue.db",
	}
}

// New 创建一个新的SQLite队列实例
func New(options *SQLiteOptions) (*SQLiteQueue, error) {
	if options == nil {
		options = DefaultOptions()
	}

	db, err := sql.Open("sqlite", options.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// 创建队列表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS queue_items (
			id TEXT PRIMARY KEY,
			worker_name TEXT NOT NULL,
			handler_name TEXT NOT NULL,
			params BLOB,
			created_at TIMESTAMP NOT NULL,
			run_at TIMESTAMP NOT NULL,
			retried INTEGER DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_worker_run_at ON queue_items(worker_name, run_at);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create queue table: %w", err)
	}

	return &SQLiteQueue{
		db:     db,
		dbPath: options.DBPath,
	}, nil
}

// Pop 从队列中获取并移除指定数量的队列项
func (s *SQLiteQueue) Pop(workerName string, num int) []*queue.QueueItem {
	var result []*queue.QueueItem

	// 开始事务
	tx, err := s.db.Begin()
	if err != nil {
		return result
	}
	defer tx.Rollback()

	// 获取可以执行的任务（当前时间大于或等于队列项的执行时间）
	now := time.Now().UTC()
	rows, err := tx.Query(`
		SELECT id, worker_name, handler_name, params, created_at, run_at, retried
		FROM queue_items
		WHERE worker_name = ? AND run_at <= ?
		ORDER BY run_at
		LIMIT ?
	`, workerName, now.Format(time.RFC3339), num)

	if err != nil {
		return result
	}
	defer rows.Close()

	// 收集要删除的ID
	var ids []interface{}
	var placeholders []string

	for rows.Next() {
		item := &queue.QueueItem{}
		var createdAtStr, runAtStr string

		err := rows.Scan(
			&item.ID,
			&item.WorkerName,
			&item.HandlerName,
			&item.Params,
			&createdAtStr,
			&runAtStr,
			&item.Retried,
		)
		if err != nil {
			continue
		}

		// 解析时间
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		item.RunAt, _ = time.Parse(time.RFC3339, runAtStr)

		result = append(result, item)
		ids = append(ids, item.ID)
		placeholders = append(placeholders, "?")
	}

	// 如果没有项目，则直接返回
	if len(ids) == 0 {
		tx.Commit()
		return result
	}

	// 删除已获取的项目
	query := fmt.Sprintf("DELETE FROM queue_items WHERE id IN (?")
	if len(ids) > 1 {
		for i := 1; i < len(ids); i++ {
			query += ",?"
		}
	}
	query += ")"

	_, err = tx.Exec(query, ids...)
	if err != nil {
		return result[:0] // 出错时返回空结果
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return result[:0] // 出错时返回空结果
	}

	return result
}

// Add 向队列添加新的队列项
func (s *SQLiteQueue) Add(workerName string, item *queue.QueueItem) error {
	_, err := s.db.Exec(`
		INSERT INTO queue_items
		(id, worker_name, handler_name, params, created_at, run_at, retried)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		workerName,
		item.HandlerName,
		item.Params,
		item.CreatedAt.UTC().Format(time.RFC3339),
		item.RunAt.UTC().Format(time.RFC3339),
		item.Retried,
	)

	if err != nil {
		return fmt.Errorf("failed to add item to queue: %w", err)
	}

	return nil
}

// Del 从队列中删除指定的队列项
func (s *SQLiteQueue) Del(workerName string, id string) error {
	_, err := s.db.Exec("DELETE FROM queue_items WHERE worker_name = ? AND id = ?", workerName, id)
	if err != nil {
		return fmt.Errorf("failed to delete item from queue: %w", err)
	}
	return nil
}

// Count 获取队列中的项目数量
func (s *SQLiteQueue) Count(workerName string) int {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM queue_items WHERE worker_name = ?", workerName).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// List 获取分页的队列项列表
func (s *SQLiteQueue) List(workerName string, page int, limit int) []*queue.QueueItem {
	var result []*queue.QueueItem

	offset := (page - 1) * limit
	rows, err := s.db.Query(`
		SELECT id, worker_name, handler_name, params, created_at, run_at, retried
		FROM queue_items
		WHERE worker_name = ?
		ORDER BY run_at
		LIMIT ? OFFSET ?
	`, workerName, limit, offset)

	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		item := &queue.QueueItem{}
		var createdAtStr, runAtStr string

		err := rows.Scan(
			&item.ID,
			&item.WorkerName,
			&item.HandlerName,
			&item.Params,
			&createdAtStr,
			&runAtStr,
			&item.Retried,
		)
		if err != nil {
			continue
		}

		// 解析时间
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		item.RunAt, _ = time.Parse(time.RFC3339, runAtStr)

		result = append(result, item)
	}

	return result
}

// Close 关闭队列
func (s *SQLiteQueue) Close() error {
	return s.db.Close()
}
