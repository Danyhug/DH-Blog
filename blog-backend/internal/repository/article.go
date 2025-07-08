package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"gorm.io/gorm"
)

// ArticleRepository 封装文章相关的数据库操作
type ArticleRepository struct {
	*GormRepository[model.Article, int]
	db           *gorm.DB
	CategoryRepo *CategoryRepository
	TagRepo      *TagRepository
}

// NewArticleRepository 创建文章仓库
func NewArticleRepository(db *gorm.DB, categoryRepo *CategoryRepository, tagRepo *TagRepository) *ArticleRepository {
	return &ArticleRepository{
		GormRepository: NewGormRepository[model.Article, int](db),
		db:             db,
		CategoryRepo:   categoryRepo,
		TagRepo:        tagRepo,
	}
}

// FindByIDWithPreload 根据ID获取文章信息，并预加载指定的关联
func (r *ArticleRepository) FindByIDWithPreload(ctx context.Context, id int, preloads ...string) (*model.Article, error) {
	return r.GormRepository.FindByIDWithPreload(ctx, id, preloads...)
}

// GetArticleById 根据id获取文章信息（保留原方法名以兼容现有代码）
func (r *ArticleRepository) GetArticleById(id int) (data model.Article, err error) {
	// 使用 FindByID 获取文章，但忽略返回值，因为我们需要使用 Preload 加载关联
	_, err = r.FindByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Article{}, fmt.Errorf("查询文章失败: %w", errs.ErrArticleNotFound)
		}
		return model.Article{}, fmt.Errorf("数据库查询文章失败: %w", err)
	}

	// 加载文章的标签
	if err := r.db.Preload("Tags").First(&data, id).Error; err != nil {
		return model.Article{}, fmt.Errorf("加载文章标签失败: %w", err)
	}

	// 确保Tags字段不为nil
	if data.Tags == nil {
		data.Tags = []*model.Tag{}
	}

	return data, nil
}

// SaveArticle 保存文章
func (r *ArticleRepository) SaveArticle(article *model.Article) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 计算文章字数
		article.WordNum = countWords(article.Content)
		
		// 根据 tagSlugs 查找或创建标签
		tags, err := r.TagRepo.FindOrCreateBySlugs(tx, article.TagSlugs)
		if err != nil {
			return fmt.Errorf("查找或创建标签失败: %w", err)
		}
		
		// 如果文章有分类且没有指定标签，则获取分类的默认标签
		if article.CategoryID > 0 && len(article.TagSlugs) == 0 {
			defaultTags, err := r.CategoryRepo.GetCategoryDefaultTags(article.CategoryID)
			if err != nil {
				return fmt.Errorf("获取分类默认标签失败: %w", err)
			}
			
			// 将默认标签添加到文章标签中
			if len(defaultTags) > 0 {
				// 将 []model.Tag 转换为 []*model.Tag
				for i := range defaultTags {
					tags = append(tags, &defaultTags[i])
				}
			}
		}
		
		article.Tags = tags

		// 创建文章
		if err := tx.Create(article).Error; err != nil {
			return fmt.Errorf("创建文章失败: %w", err)
		}
		return nil
	})
}

// UpdateArticle 更新文章
func (r *ArticleRepository) UpdateArticle(article *model.Article) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 计算文章字数
		article.WordNum = countWords(article.Content)
		
		// 根据 tagSlugs 查找或创建标签
		tags, err := r.TagRepo.FindOrCreateBySlugs(tx, article.TagSlugs)
		if err != nil {
			return fmt.Errorf("查找或创建标签失败: %w", err)
		}
		
		// 如果文章有分类且没有指定标签，则获取分类的默认标签
		if article.CategoryID > 0 && len(article.TagSlugs) == 0 {
			defaultTags, err := r.CategoryRepo.GetCategoryDefaultTags(article.CategoryID)
			if err != nil {
				return fmt.Errorf("获取分类默认标签失败: %w", err)
			}
			
			// 将默认标签添加到文章标签中
			if len(defaultTags) > 0 {
				// 将 []model.Tag 转换为 []*model.Tag
				for i := range defaultTags {
					tags = append(tags, &defaultTags[i])
				}
			}
		}
		
		article.Tags = tags

		// 更新文章与标签的关联
		if err := tx.Model(article).Association("Tags").Replace(tags); err != nil {
			return fmt.Errorf("更新文章标签关联失败: %w", err)
		}

		// 更新文章
		if err := tx.Save(article).Error; err != nil {
			return fmt.Errorf("更新文章失败: %w", err)
		}
		return nil
	})
}

// GetArticlesByTagSlug 根据标签slug获取文章列表
func (r *ArticleRepository) GetArticlesByTagSlug(tagSlug string) (data []model.Article, err error) {
	err = r.db.Joins("JOIN article_tags ON article_tags.article_id = articles.id").
		Joins("JOIN tags ON tags.id = article_tags.tag_id").
		Where("tags.slug = ?", tagSlug).
		Find(&data).Error
	return
}

// UpdateArticleViewCount 更新文章浏览次数
func (r *ArticleRepository) UpdateArticleViewCount(id int) {
	r.db.Model(&model.Article{}).Where("id = ?", id).Update("views", gorm.Expr("views + 1"))
}

// countWords 计算字符串中的单词数量
func countWords(s string) int {
	// 一个简单的单词计数逻辑，可以根据需要进行优化
	return len(strings.Fields(s))
}
