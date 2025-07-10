package task

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

func NewAiGenTask(articleID int, content string) *AiGenTagTask {
	return &AiGenTagTask{
		ArticleID: articleID,
		Content:   content,
	}
}
