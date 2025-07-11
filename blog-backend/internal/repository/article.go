package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dh-blog/internal/dhcache"
	"dh-blog/internal/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 缓存键前缀
const (
	PrefixArticle     = "article:"
	PrefixArticleList = "article:list:"
	ExpireShort       = time.Minute * 5 // 短期缓存，5分钟
	ExpireLong        = time.Hour * 24  // 长期缓存，24小时
)

var (
	ErrArticleNotFound = errors.New("文章不存在")
)

// ArticleRepository 封装文章相关的数据库操作
type ArticleRepository struct {
	*GormRepository[model.Article, int]
	db           *gorm.DB
	CategoryRepo *CategoryRepository
	TagRepo      *TagRepository
	cache        dhcache.Cache
}

// NewArticleRepository 创建文章仓库
func NewArticleRepository(db *gorm.DB, categoryRepo *CategoryRepository, tagRepo *TagRepository, cache dhcache.Cache) *ArticleRepository {
	return &ArticleRepository{
		GormRepository: NewGormRepository[model.Article, int](db),
		db:             db,
		CategoryRepo:   categoryRepo,
		TagRepo:        tagRepo,
		cache:          cache,
	}
}

// FindByIDWithPreload 根据ID获取文章信息，并预加载指定的关联
func (r *ArticleRepository) FindByIDWithPreload(ctx context.Context, id int, preloads ...string) (*model.Article, error) {
	return r.GormRepository.FindByIDWithPreload(ctx, id, preloads...)
}

// GetArticleById 根据id获取文章信息（保留原方法名以兼容现有代码）
func (r *ArticleRepository) GetArticleById(id int) (data model.Article, err error) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("%s%d", PrefixArticle, id)
	if cached, found := r.cache.Get(cacheKey); found {
		if article, ok := cached.(model.Article); ok {
			logrus.Debugf("从缓存获取文章: %d", id)
			return article, nil
		} else {
			// 缓存类型转换失败
			logrus.Warnf("文章缓存类型转换失败: %d, 将从数据库重新获取", id)
		}
	}

	// 使用 FindByID 获取文章，但忽略返回值，因为我们需要使用 Preload 加载关联
	_, err = r.FindByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Article{}, fmt.Errorf("查询文章失败: %w", ErrArticleNotFound)
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

	// 存入缓存
	r.cache.Set(cacheKey, data, ExpireShort)
	logrus.Debugf("文章已缓存: %d", id)

	return data, nil
}

// SaveArticle 保存文章
func (r *ArticleRepository) SaveArticle(article *model.Article) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 计算文章字数
		article.WordNum = countWords(article.Content)

		// 根据 tagNames 查找或创建标签
		tags, err := r.TagRepo.FindOrCreateByNames(tx, article.TagNames)
		if err != nil {
			return fmt.Errorf("查找或创建标签失败: %w", err)
		}

		// 如果文章有分类且没有指定标签，则获取分类的默认标签
		if article.CategoryID > 0 && len(article.TagNames) == 0 {
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

	// 清除文章列表缓存
	if err == nil {
		r.clearArticleListCache()
	} else {
		logrus.Errorf("保存文章失败: %v", err)
	}

	return err
}

// clearArticleListCache 清除文章列表相关的所有缓存
func (r *ArticleRepository) clearArticleListCache() {
	// 由于无法精确删除所有分页缓存，这里使用一个标记键来表示缓存已失效
	cacheKey := fmt.Sprintf("%scount", PrefixArticleList)
	if deleted := r.cache.Delete(cacheKey); !deleted {
		logrus.Warnf("清除文章列表缓存失败: 缓存中未找到")
	} else {
		logrus.Debug("已清除文章列表缓存")
	}
}

// UpdateArticle 更新文章
func (r *ArticleRepository) UpdateArticle(article *model.Article) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 计算文章字数
		article.WordNum = countWords(article.Content)

		// 根据 tagNames 查找或创建标签
		tags, err := r.TagRepo.FindOrCreateByNames(tx, article.TagNames)
		if err != nil {
			return fmt.Errorf("查找或创建标签失败: %w", err)
		}

		// 如果文章有分类且没有指定标签，则获取分类的默认标签
		if article.CategoryID > 0 && len(article.TagNames) == 0 {
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

	// 清除文章缓存
	if err == nil {
		// 清除文章详情缓存
		cacheKey := fmt.Sprintf("%s%d", PrefixArticle, article.ID)
		if deleted := r.cache.Delete(cacheKey); !deleted {
			logrus.Warnf("清除文章缓存失败: %d, 缓存中未找到", article.ID)
		}

		// 清除文章列表缓存
		r.clearArticleListCache()
		logrus.Debugf("已清除文章缓存: %d", article.ID)
	} else {
		logrus.Errorf("更新文章失败: %d, 错误: %v", article.ID, err)
	}

	return err
}

// GetArticlesByTagName 根据标签名获取文章列表
func (r *ArticleRepository) GetArticlesByTagName(tagName string) (data []model.Article, err error) {
	// 缓存键
	cacheKey := fmt.Sprintf("%stag:%s", PrefixArticleList, tagName)

	// 尝试从缓存获取
	if cached, found := r.cache.Get(cacheKey); found {
		if articles, ok := cached.([]model.Article); ok {
			logrus.Debugf("从缓存获取标签文章列表: %s", tagName)
			return articles, nil
		} else {
			logrus.Warnf("标签文章列表缓存类型转换失败: %s, 将从数据库重新获取", tagName)
		}
	}

	// 从数据库获取
	err = r.db.Joins("JOIN article_tags ON article_tags.article_id = articles.id").
		Joins("JOIN tags ON tags.id = article_tags.tag_id").
		Where("tags.name = ?", tagName).
		Find(&data).Error

	if err != nil {
		return nil, fmt.Errorf("获取标签文章列表失败: %s, 错误: %w", tagName, err)
	}

	// 存入缓存
	r.cache.Set(cacheKey, data, ExpireShort)
	logrus.Debugf("标签文章列表已缓存: %s", tagName)

	return data, nil
}

// UpdateArticleViewCount 更新文章浏览次数
func (r *ArticleRepository) UpdateArticleViewCount(id int) {
	// 更新数据库中的浏览次数
	if err := r.db.Model(&model.Article{}).Where("id = ?", id).Update("views", gorm.Expr("views + 1")).Error; err != nil {
		logrus.Errorf("更新文章浏览次数失败: %d, 错误: %v", id, err)
		return
	}

	// 缓存键
	cacheKey := fmt.Sprintf("%s%d", PrefixArticle, id)

	// 尝试从缓存获取
	if cached, found := r.cache.Get(cacheKey); found {
		if article, ok := cached.(model.Article); ok {
			// 更新缓存中的浏览次数
			article.Views++
			// 更新缓存
			r.cache.Set(cacheKey, article, ExpireShort)
			logrus.Debugf("更新缓存中的文章浏览次数: %d, 新浏览次数: %d", id, article.Views)
			return
		} else {
			// 缓存类型转换失败
			logrus.Warnf("文章缓存类型转换失败: %d, 无法更新缓存中的浏览次数", id)
		}
	} else {
		logrus.Debugf("缓存中未找到文章: %d, 跳过缓存更新", id)
	}

	// 如果缓存中没有，不做任何操作
	// 下次获取文章时会从数据库中获取最新的浏览次数
}

// FindPage 重写分页查询方法，添加缓存支持
func (r *ArticleRepository) FindPage(ctx context.Context, page, pageSize int) ([]model.Article, int64, error) {
	// 缓存键
	cacheKey := fmt.Sprintf("%spage:%d:%d", PrefixArticleList, page, pageSize)
	cacheCountKey := fmt.Sprintf("%scount", PrefixArticleList)

	// 尝试从缓存获取文章列表
	var articles []model.Article
	var total int64

	if cached, found := r.cache.Get(cacheKey); found {
		if cachedArticles, ok := cached.([]model.Article); ok {
			articles = cachedArticles

			// 尝试从缓存获取总数
			if cachedTotal, foundTotal := r.cache.Get(cacheCountKey); foundTotal {
				if cachedCount, ok := cachedTotal.(int64); ok {
					logrus.Debugf("从缓存获取文章分页: %d, %d", page, pageSize)
					return articles, cachedCount, nil
				} else {
					logrus.Warnf("文章总数缓存类型转换失败，将从数据库重新获取")
				}
			} else {
				logrus.Debugf("缓存中未找到文章总数，将从数据库获取")
			}
		} else {
			logrus.Warnf("文章列表缓存类型转换失败，将从数据库重新获取")
		}
	}

	// 缓存未命中或类型转换失败，从数据库获取
	var err error
	articles, total, err = r.GormRepository.FindPage(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("获取文章分页失败: %w", err)
	}

	// 存入缓存
	r.cache.Set(cacheKey, articles, ExpireShort)
	r.cache.Set(cacheCountKey, total, ExpireLong)
	logrus.Debugf("文章分页已缓存: %d, %d", page, pageSize)

	return articles, total, nil
}

// Delete 重写删除方法，添加缓存清理
func (r *ArticleRepository) Delete(ctx context.Context, id int) error {
	err := r.GormRepository.Delete(ctx, id)

	// 清除缓存
	if err == nil {
		// 清除文章详情缓存
		cacheKey := fmt.Sprintf("%s%d", PrefixArticle, id)
		if deleted := r.cache.Delete(cacheKey); !deleted {
			logrus.Warnf("清除文章缓存失败: %d, 错误: 缓存中未找到", id)
		}

		// 清除文章列表缓存
		r.clearArticleListCache()
		logrus.Debugf("已清除已删除文章的缓存: %d", id)
	} else {
		logrus.Errorf("删除文章失败: %d, 错误: %v", id, err)
	}

	return err
}

// countWords 计算字符串中的单词数量
func countWords(s string) int {
	// 一个简单的单词计数逻辑，可以根据需要进行优化
	return len(strings.Fields(s))
}
