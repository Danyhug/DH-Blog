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

// Target reports the article this task writes back to.
func (a *AiGenTagTask) Target() int { return a.ArticleID }

// NewAiGenTask 创建AI生成标签任务
func NewAiGenTask(articleID int, content string) *AiGenTagTask {
	return &AiGenTagTask{
		ArticleID: articleID,
		Content:   content,
	}
}

// AiGenSummaryTask AI生成摘要任务
type AiGenSummaryTask struct {
	ArticleID int
	Content   string
}

func (a *AiGenSummaryTask) Type() string {
	return "AI_Gen_Summary"
}

func (a *AiGenSummaryTask) Payload() interface{} {
	return a
}

// Target reports the article this task writes back to.
func (a *AiGenSummaryTask) Target() int { return a.ArticleID }

// NewAiGenSummaryTask 创建AI生成摘要任务
func NewAiGenSummaryTask(articleID int, content string) *AiGenSummaryTask {
	return &AiGenSummaryTask{
		ArticleID: articleID,
		Content:   content,
	}
}
