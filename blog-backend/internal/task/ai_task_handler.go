package task

import (
	"context"
	"fmt"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/service"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RegisterAITaskHandlers 注册AI相关的任务处理函数
func RegisterAITaskHandlers(
	dispatcher *Dispatcher,
	db *gorm.DB,
	aiService service.AIService,
	tagRepo *repository.TagRepository,
) {
	// 注册AI生成标签任务处理函数
	dispatcher.Register("AI_Gen_Tags", func(ctx context.Context, payload interface{}) error {
		return handleAIGenTagsTask(ctx, payload, db, aiService, tagRepo)
	})
}

// handleAIGenTagsTask 处理AI生成标签任务
func handleAIGenTagsTask(
	ctx context.Context,
	payload interface{},
	db *gorm.DB,
	aiService service.AIService,
	tagRepo *repository.TagRepository,
) error {
	aiTask, ok := payload.(*AiGenTagTask)
	if !ok {
		return fmt.Errorf("无效的任务负载类型")
	}

	// 调用AI服务生成标签
	tagNames, err := aiService.GenerateTags(aiTask.Content)
	if err != nil {
		return fmt.Errorf("生成标签失败: %w", err)
	}

	logrus.Infof("为文章 %d 生成标签: %v", aiTask.ArticleID, tagNames)

	// 开启事务
	return db.Transaction(func(tx *gorm.DB) error {
		// 查找文章
		var article model.Article
		if err := tx.First(&article, aiTask.ArticleID).Error; err != nil {
			return fmt.Errorf("查找文章失败: %w", err)
		}

		// 获取文章当前的标签
		var currentTags []*model.Tag
		if err := tx.Model(&article).Association("Tags").Find(&currentTags); err != nil {
			return fmt.Errorf("获取文章当前标签失败: %w", err)
		}

		// 创建当前标签名称的集合，用于检查重复
		currentTagNames := make(map[string]bool)
		for _, tag := range currentTags {
			currentTagNames[tag.Name] = true
		}

		// 过滤掉已存在的标签名称
		var newTagNames []string
		for _, name := range tagNames {
			// 跳过空白标签名
			if name == "" {
				continue
			}

			// 如果标签名不在当前标签中，添加到新标签列表
			if !currentTagNames[name] {
				newTagNames = append(newTagNames, name)
			}
		}

		// 如果没有新标签，直接返回成功
		if len(newTagNames) == 0 {
			logrus.Infof("文章 %d 没有新的标签需要添加", aiTask.ArticleID)
			return nil
		}

		// 查找或创建新标签
		newTags, err := tagRepo.FindOrCreateByNames(tx, newTagNames)
		if err != nil {
			return fmt.Errorf("查找或创建标签失败: %w", err)
		}

		logrus.Infof("将为文章 %d 添加 %d 个新标签", aiTask.ArticleID, len(newTags))

		// 将新标签添加到文章中
		if err := tx.Model(&article).Association("Tags").Append(newTags); err != nil {
			return fmt.Errorf("添加文章标签关联失败: %w", err)
		}

		logrus.Infof("成功为文章 %d 添加AI生成的标签", aiTask.ArticleID)
		return nil
	})
}
