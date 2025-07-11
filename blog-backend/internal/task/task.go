package task

import (
	"context"
)

// Task 表示一个后台任务
type Task interface {
	// Type 返回任务类型，方便管理
	Type() string
	// Payload 返回任务负载
	Payload() interface{}
}

// Handler 任务处理函数
type Handler func(ctx context.Context, payload interface{}) error

// AiGenTagTask AI生成标签任务
type AiGenTagTask struct {
	ArticleID int
	Content   string
}

func (a *AiGenTagTask) Type() string {
	return "AI_Gen_Tags"
}

func (a *AiGenTagTask) Payload() interface{} {
	return a
}

// NewAiGenTask 创建AI生成标签任务
func NewAiGenTask(articleID int, content string) *AiGenTagTask {
	return &AiGenTagTask{
		ArticleID: articleID,
		Content:   content,
	}
}
