package article

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ProcessSummaryGeneration is the business handler registered with the generic
// task scheduler. Queueing and retries stay in internal/task.
func (h *Handler) ProcessSummaryGeneration(ctx context.Context, articleID int, content string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("任务上下文已取消: %w", err)
	}
	if strings.TrimSpace(content) == "" {
		logrus.Infof("文章 %d 正文为空，跳过摘要生成", articleID)
		return nil
	}
	start := time.Now()
	logrus.Infof("开始处理文章 %d 的AI摘要生成任务", articleID)

	if h.ai == nil {
		return fmt.Errorf("AI摘要服务未配置")
	}
	result := make(chan struct {
		summary string
		err     error
	}, 1)
	go func() {
		summary, callErr := h.ai.GenerateSummary(content)
		result <- struct {
			summary string
			err     error
		}{summary: summary, err: callErr}
	}()

	var summary string
	select {
	case generated := <-result:
		if generated.err != nil {
			return fmt.Errorf("生成摘要失败: %w", generated.err)
		}
		summary = generated.summary
	case <-ctx.Done():
		return fmt.Errorf("AI摘要生成超时: %w", ctx.Err())
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return fmt.Errorf("AI 返回的摘要为空")
	}
	logrus.Infof("为文章 %d 生成摘要 (耗时: %v)", articleID, time.Since(start))
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("任务上下文已取消: %w", err)
	}
	if err := h.articleRepository.SaveGeneratedSummary(ctx, articleID, summary); err != nil {
		return err
	}
	logrus.Infof("成功保存文章 %d 的AI摘要 (总耗时: %v)", articleID, time.Since(start))
	return nil
}
