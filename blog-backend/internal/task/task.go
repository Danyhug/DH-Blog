package task

import (
	"context"
)

type Task interface {
	// Type 返回任务类型，方便管理
	Type() string
	// Payload 返回任务负载
	Payload() interface{}
}

// Handler 任务处理函数
type Handler func(ctx context.Context, payload interface{}) error
