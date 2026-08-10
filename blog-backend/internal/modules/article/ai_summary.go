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

	summary = normalizeSummary(summary)
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

// summaryMaxRunes mirrors the length cap stated in the summary prompt. Models
// overshoot it often enough that the homepage needs a hard guarantee.
const summaryMaxRunes = 120

// normalizeSummary trims the model output and clamps it to summaryMaxRunes.
// A clamped summary is cut back to its last sentence end when one survives in
// the second half, so the homepage shows a finished sentence instead of a word
// sliced in half.
func normalizeSummary(summary string) string {
	summary = strings.TrimSpace(summary)
	runes := []rune(summary)
	if len(runes) <= summaryMaxRunes {
		return summary
	}
	clipped := runes[:summaryMaxRunes]
	if end := lastSentenceEnd(clipped); end >= summaryMaxRunes/2 {
		clipped = clipped[:end+1]
	}
	return strings.TrimSpace(string(clipped))
}

func lastSentenceEnd(runes []rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		switch runes[i] {
		case '。', '！', '？', '.', '!', '?':
			return i
		}
	}
	return -1
}
