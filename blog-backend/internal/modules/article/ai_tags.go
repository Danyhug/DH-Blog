package article

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// ProcessTagGeneration is the business handler registered with the generic
// task scheduler. Queueing and retries stay in internal/task.
func (h *Handler) ProcessTagGeneration(ctx context.Context, articleID int, content string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("任务上下文已取消: %w", err)
	}
	start := time.Now()
	logrus.Infof("开始处理文章 %d 的AI标签生成任务", articleID)

	existingTagNames, err := h.tagRepository.GetAllTagNamesWithCache(ctx)
	if err != nil {
		logrus.Warnf("获取现有标签失败: %v，将使用空标签列表", err)
		existingTagNames = []string{}
	}
	logrus.Infof("获取到 %d 个现有标签供AI参考", len(existingTagNames))

	if h.ai == nil {
		return fmt.Errorf("AI标签服务未配置")
	}
	result := make(chan struct {
		tags []string
		err  error
	}, 1)
	go func() {
		tags, callErr := h.ai.GenerateTags(content, existingTagNames)
		result <- struct {
			tags []string
			err  error
		}{tags: tags, err: callErr}
	}()

	var tagNames []string
	select {
	case generated := <-result:
		if generated.err != nil {
			return fmt.Errorf("生成标签失败: %w", generated.err)
		}
		tagNames = generated.tags
	case <-ctx.Done():
		return fmt.Errorf("AI标签生成超时: %w", ctx.Err())
	}

	logrus.Infof("为文章 %d 生成标签: %v (耗时: %v)", articleID, tagNames, time.Since(start))
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("任务上下文已取消: %w", err)
	}
	if err := h.articleRepository.AppendGeneratedTags(ctx, articleID, tagNames); err != nil {
		return err
	}
	logrus.Infof("成功为文章 %d 添加AI生成的标签 (总耗时: %v)", articleID, time.Since(start))
	return nil
}
