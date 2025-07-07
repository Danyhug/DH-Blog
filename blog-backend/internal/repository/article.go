package repository

import (
	"errors"
	"fmt"

	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"gorm.io/gorm"
)

type ArticleRepository struct {
	DB           *gorm.DB
	CategoryRepo *CategoryRepository
	TagRepo      *TagRepository
}

func NewArticleRepository(db *gorm.DB, categoryRepo *CategoryRepository, tagRepo *TagRepository) *ArticleRepository {
	return &ArticleRepository{DB: db, CategoryRepo: categoryRepo, TagRepo: tagRepo}
}

// GetArticleById 根据id获取文章信息
func (r *ArticleRepository) GetArticleById(id int) (data model.Article, err error) {
	tx := r.DB.Preload("Tags").Where(&model.Article{BaseModel: model.BaseModel{ID: int(id)}}).First(&data)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return model.Article{}, fmt.Errorf("查询文章失败: %w", errs.ErrArticleNotFound)
		}
		return model.Article{}, fmt.Errorf("数据库查询文章失败: %w", tx.Error)
	}

	// 1. 获取文章分类的默认标签id
	defaultTagIDs, err := r.CategoryRepo.GetCategoryDefaultTagIDs(int(data.CategoryID))
	if err != nil {
		return model.Article{}, fmt.Errorf("获取分类默认标签失败: %w", err)
	}

	// 2. 获取文章的标签id
	articleTagIDs := make([]int, 0, len(data.Tags))
	for _, tag := range data.Tags {
		articleTagIDs = append(articleTagIDs, tag.ID)
	}

	// 3. 将默认标签id和文章标签id去重后为一个ids集合
	uniqueTagIDsMap := make(map[int]struct{})
	for _, id := range defaultTagIDs {
		uniqueTagIDsMap[id] = struct{}{}
	}
	for _, id := range articleTagIDs {
		uniqueTagIDsMap[id] = struct{}{}
	}

	var allUniqueTagIDs []int
	for id := range uniqueTagIDsMap {
		allUniqueTagIDs = append(allUniqueTagIDs, id)
	}

	return data, nil
}

// GetArticleTitleByID 根据 ID 获取文章标题
func (r *ArticleRepository) GetArticleTitleByID(id int) (string, error) {
	var article model.Article
	err := r.DB.Select("title").First(&article, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("查询文章标题失败: %w", errs.ErrArticleNotFound)
		}
		return "", fmt.Errorf("数据库查询文章标题失败: %w", err)
	}
	return article.Title, nil
}

// GetLockedArticle 获取需要解密的文章
func (r *ArticleRepository) GetLockedArticle(id int, password string) (model.Article, error) {
	var article model.Article
	err := r.DB.Where("id = ? AND is_locked = ? AND lock_password = ?", id, true, password).First(&article).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Article{}, fmt.Errorf("查询加密文章失败: %w", errs.ErrArticleNotFound)
		}
		return model.Article{}, fmt.Errorf("数据库查询加密文章失败: %w", err)
	}
	return article, nil
}

// SaveArticle 保存文章，并处理标签关联
func (r *ArticleRepository) SaveArticle(article *model.Article) error {
	// 使用事务来确保数据一致性
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 保存文章主体
		if err := tx.Create(article).Error; err != nil {
			return fmt.Errorf("保存文章主体失败: %w", err)
		}

		// 2. 处理标签关联
		// 在事务内部创建 TagRepository 实例，确保所有操作都在同一个事务中
		tagRepoTx := NewTagRepository(tx)
		var tagsToAssociate []*model.Tag
		for _, tagSlug := range article.TagSlugs {
			tag, err := tagRepoTx.GetTagBySlug(tagSlug)
			if err != nil {
				if errors.Is(err, errs.ErrNotFound) {
					return fmt.Errorf("标签 slug '%s' 不存在: %w", tagSlug, errs.ErrNotFound)
				} else {
					return fmt.Errorf("查询标签失败: %w", err)
				}
			} else {
				tagsToAssociate = append(tagsToAssociate, &tag)
			}
		}

		// 3. 建立文章与标签的多对多关联
		if err := tx.Model(article).Association("Tags").Replace(tagsToAssociate); err != nil {
			return fmt.Errorf("建立文章标签关联失败: %w", err)
		}

		return nil
	})
}

// UpdateArticle 更新文章
func (r *ArticleRepository) UpdateArticle(article *model.Article) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 更新文章主体信息
		if err := tx.Model(&article).Updates(article).Error; err != nil {
			return fmt.Errorf("更新文章主体失败: %w", err)
		}

		// 2. 处理标签关联
		// 在事务内部创建 TagRepository 实例，确保所有操作都在同一个事务中
		tagRepoTx := NewTagRepository(tx)
		var tagsToAssociate []*model.Tag
		for _, tagSlug := range article.TagSlugs {
			tag, err := tagRepoTx.GetTagBySlug(tagSlug)
			if err != nil {
				if errors.Is(err, errs.ErrNotFound) {
					return fmt.Errorf("标签 slug '%s' 不存在: %w", tagSlug, errs.ErrNotFound)
				} else {
					return fmt.Errorf("查询标签失败: %w", err)
				}
			} else {
				tagsToAssociate = append(tagsToAssociate, &tag)
			}
		}

		// 3. 更新文章与标签的多对多关联
		if err := tx.Model(article).Association("Tags").Replace(tagsToAssociate); err != nil {
			return fmt.Errorf("更新文章标签关联失败: %w", err)
		}

		return nil
	})
}

// DeleteArticle 删除文章
func (r *ArticleRepository) DeleteArticle(id int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 删除文章与标签的关联
		if err := tx.Model(&model.Article{BaseModel: model.BaseModel{ID: int(id)}}).Association("Tags").Clear(); err != nil {
			return fmt.Errorf("删除文章标签关联失败: %w", err)
		}
		// 2. 删除文章本身
		if err := tx.Delete(&model.Article{}, id).Error; err != nil {
			return fmt.Errorf("删除文章失败: %w", err)
		}
		return nil
	})
}

// GetArticleList 获取文章列表（带分页）
func (r *ArticleRepository) GetArticleList(page, pageSize int) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	offset := (page - 1) * pageSize

	// 查询总数
	if err := r.DB.Model(&model.Article{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章总数失败: %w", err)
	}

	// 查询文章列表，并预加载标签
	if err := r.DB.Offset(offset).Limit(pageSize).Preload("Tags").Find(&articles).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章列表失败: %w", err)
	}

	return articles, total, nil
}

// AddViewCount 增加文章浏览次数
func (r *ArticleRepository) AddViewCount(id int) {
	r.DB.Model(&model.Article{}).Where("id = ?", id).Update("views", gorm.Expr("views + 1"))
}
