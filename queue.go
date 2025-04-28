package queue

import (
	"time"
)

type QueueItem struct {
	ID          string    `json:"id"`
	WorkerName  string    `json:"worker_name"`
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
