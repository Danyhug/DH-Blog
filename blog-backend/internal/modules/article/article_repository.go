package article

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	PrefixArticle     = "article:"
	PrefixArticleList = "article:list:"
	ExpireShort       = 5 * time.Minute
	ExpireLong        = 24 * time.Hour
)

var ErrArticleNotFound = errors.New("文章不存在")

type ArticleRepository struct {
	*gormRepository[Article, int]
	db                 *gorm.DB
	categoryRepository *CategoryRepository
	tagRepository      *TagRepository
	cache              Cache
}

func NewArticleRepository(db *gorm.DB, categoryRepository *CategoryRepository, tagRepository *TagRepository, cache Cache) *ArticleRepository {
	return &ArticleRepository{
		gormRepository:     newGormRepository[Article, int](db),
		db:                 db,
		categoryRepository: categoryRepository,
		tagRepository:      tagRepository,
		cache:              cache,
	}
}

func (r *ArticleRepository) GetArticleById(id int) (Article, error) {
	cacheKey := fmt.Sprintf("%s%d", PrefixArticle, id)
	if cached, found := r.cache.Get(cacheKey); found {
		if article, ok := cached.(Article); ok {
			logrus.Debugf("从缓存获取文章: %d", id)
			return article, nil
		}
		logrus.Warnf("文章缓存类型转换失败: %d, 将从数据库重新获取", id)
	}

	if _, err := r.FindByID(context.Background(), id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Article{}, fmt.Errorf("查询文章失败: %w", ErrArticleNotFound)
		}
		return Article{}, fmt.Errorf("数据库查询文章失败: %w", err)
	}

	var article Article
	if err := r.db.Preload("Tags").First(&article, id).Error; err != nil {
		return Article{}, fmt.Errorf("加载文章标签失败: %w", err)
	}
	if article.Tags == nil {
		article.Tags = []*Tag{}
	}
	_ = r.cache.Set(cacheKey, article, ExpireShort)
	logrus.Debugf("文章已缓存: %d", id)
	return article, nil
}

func (r *ArticleRepository) SaveArticle(article *Article) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		article.WordNum = countWords(article.Content)
		tags, err := r.resolveTags(tx, article.CategoryID, article.TagNames)
		if err != nil {
			return err
		}
		article.Tags = tags
		if err := tx.Create(article).Error; err != nil {
			return fmt.Errorf("创建文章失败: %w", err)
		}
		return nil
	})
	if err == nil {
		r.clearArticleListCache()
	} else {
		logrus.Errorf("保存文章失败: %v", err)
	}
	return err
}

func (r *ArticleRepository) UpdateArticle(article *Article) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		article.WordNum = countWords(article.Content)
		tags, err := r.resolveTags(tx, article.CategoryID, article.TagNames)
		if err != nil {
			return err
		}
		article.Tags = tags
		if err := r.updateArticleTags(tx, article, tags); err != nil {
			return fmt.Errorf("更新文章标签关联失败: %w", err)
		}
		if err := tx.Save(article).Error; err != nil {
			return fmt.Errorf("更新文章失败: %w", err)
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("更新文章失败: %d, 错误: %v", article.ID, err)
		return err
	}

	cacheKey := fmt.Sprintf("%s%d", PrefixArticle, article.ID)
	if deleted := r.cache.Delete(cacheKey); !deleted {
		logrus.Warnf("清除文章缓存失败: %d, 缓存中未找到", article.ID)
	}
	r.clearArticleListCache()
	logrus.Debugf("已清除文章缓存: %d", article.ID)
	return nil
}

func (r *ArticleRepository) resolveTags(tx *gorm.DB, categoryID int, names []string) ([]*Tag, error) {
	tags, err := r.tagRepository.FindOrCreateByNames(tx, names)
	if err != nil {
		return nil, fmt.Errorf("查找或创建标签失败: %w", err)
	}
	if categoryID <= 0 || len(names) > 0 {
		return tags, nil
	}
	defaults, err := r.categoryRepository.getCategoryDefaultTags(tx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("获取分类默认标签失败: %w", err)
	}
	for i := range defaults {
		tags = append(tags, &defaults[i])
	}
	return tags, nil
}

func (r *ArticleRepository) GetArticlesByTagName(tagName string) ([]Article, error) {
	cacheKey := fmt.Sprintf("%stag:%s", PrefixArticleList, tagName)
	if cached, found := r.cache.Get(cacheKey); found {
		if articles, ok := cached.([]Article); ok {
			logrus.Debugf("从缓存获取标签文章列表: %s", tagName)
			return articles, nil
		}
		logrus.Warnf("标签文章列表缓存类型转换失败: %s, 将从数据库重新获取", tagName)
	}

	var articles []Article
	err := r.db.Joins("JOIN article_tags ON article_tags.article_id = articles.id").
		Joins("JOIN tags ON tags.id = article_tags.tag_id").
		Where("tags.name = ?", tagName).
		Find(&articles).Error
	if err != nil {
		return nil, fmt.Errorf("获取标签文章列表失败: %s, 错误: %w", tagName, err)
	}
	_ = r.cache.Set(cacheKey, articles, ExpireShort)
	logrus.Debugf("标签文章列表已缓存: %s", tagName)
	return articles, nil
}

func (r *ArticleRepository) FindByTagName(_ context.Context, tagName string) ([]*Article, error) {
	articles, err := r.GetArticlesByTagName(tagName)
	if err != nil {
		return nil, err
	}
	result := make([]*Article, len(articles))
	for i := range articles {
		result[i] = &articles[i]
	}
	return result, nil
}

func (r *ArticleRepository) CountArticlesByTagName(ctx context.Context, tagName string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("articles a").
		Joins("JOIN article_tags at ON a.id = at.article_id").
		Joins("JOIN tags t ON at.tag_id = t.id").
		Where("t.name = ? AND a.deleted_at IS NULL", tagName).
		Count(&count).Error
	return count, err
}

func (r *ArticleRepository) CountArticlesByCategoryName(ctx context.Context, categoryName string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("articles a").
		Joins("JOIN categories c ON a.category_id = c.id").
		Where("c.name = ? AND a.deleted_at IS NULL", categoryName).
		Count(&count).Error
	return count, err
}

func (r *ArticleRepository) GetArticlesByCategoryName(categoryName string) ([]Article, error) {
	cacheKey := fmt.Sprintf("%scategory:%s", PrefixArticleList, categoryName)
	if cached, found := r.cache.Get(cacheKey); found {
		if articles, ok := cached.([]Article); ok {
			logrus.Debugf("从缓存获取分类文章列表: %s", categoryName)
			return articles, nil
		}
		logrus.Warnf("分类文章列表缓存类型转换失败: %s, 将从数据库重新获取", categoryName)
	}

	var articles []Article
	err := r.db.Joins("JOIN categories ON categories.id = articles.category_id").
		Where("categories.name = ? OR categories.slug = ?", categoryName, categoryName).
		Find(&articles).Error
	if err != nil {
		return nil, fmt.Errorf("获取分类文章列表失败: %s, 错误: %w", categoryName, err)
	}
	_ = r.cache.Set(cacheKey, articles, ExpireShort)
	logrus.Debugf("分类文章列表已缓存: %s", categoryName)
	return articles, nil
}

func (r *ArticleRepository) FindByCategoryName(_ context.Context, categoryName string) ([]*Article, error) {
	articles, err := r.GetArticlesByCategoryName(categoryName)
	if err != nil {
		return nil, err
	}
	result := make([]*Article, len(articles))
	for i := range articles {
		result[i] = &articles[i]
	}
	return result, nil
}

func (r *ArticleRepository) updateArticleTags(tx *gorm.DB, article *Article, newTags []*Tag) error {
	var currentTags []Tag
	if err := tx.Model(article).Association("Tags").Find(&currentTags); err != nil {
		return fmt.Errorf("获取当前标签失败: %w", err)
	}
	current := make(map[int]bool, len(currentTags))
	for _, tag := range currentTags {
		current[tag.ID] = true
	}
	next := make(map[int]bool, len(newTags))
	for _, tag := range newTags {
		if tag.ID > 0 {
			next[tag.ID] = true
		}
	}
	for _, tag := range currentTags {
		if !next[tag.ID] {
			if err := tx.Model(article).Association("Tags").Delete(&Tag{BaseModel: tag.BaseModel}); err != nil {
				logrus.Warnf("删除标签关联失败: %d, 错误: %v", tag.ID, err)
			}
		}
	}
	var toAdd []*Tag
	for _, tag := range newTags {
		if tag.ID > 0 && !current[tag.ID] {
			toAdd = append(toAdd, &Tag{BaseModel: tag.BaseModel})
		}
	}
	if len(toAdd) > 0 {
		if err := tx.Model(article).Association("Tags").Append(toAdd); err != nil {
			return fmt.Errorf("添加标签关联失败: %w", err)
		}
	}
	return nil
}

// AppendGeneratedTags applies AI-generated tags without replacing manually
// selected tags. This preserves the historical background-task semantics.
func (r *ArticleRepository) AppendGeneratedTags(ctx context.Context, articleID int, names []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var article Article
		if err := tx.First(&article, articleID).Error; err != nil {
			return fmt.Errorf("查找文章失败: %w", err)
		}
		var currentTags []*Tag
		if err := tx.Model(&article).Association("Tags").Find(&currentTags); err != nil {
			return fmt.Errorf("获取文章当前标签失败: %w", err)
		}
		currentNames := make(map[string]bool, len(currentTags))
		for _, tag := range currentTags {
			currentNames[tag.Name] = true
		}
		newNames := make([]string, 0, len(names))
		for _, name := range names {
			if name != "" && !currentNames[name] {
				newNames = append(newNames, name)
			}
		}
		if len(newNames) == 0 {
			logrus.Infof("文章 %d 没有新的标签需要添加", articleID)
			return nil
		}
		newTags, err := r.tagRepository.FindOrCreateByNames(tx, newNames)
		if err != nil {
			return fmt.Errorf("查找或创建标签失败: %w", err)
		}
		logrus.Infof("将为文章 %d 添加 %d 个新标签", articleID, len(newTags))
		if err := tx.Model(&article).Association("Tags").Append(newTags); err != nil {
			return fmt.Errorf("添加文章标签关联失败: %w", err)
		}
		return nil
	})
}

func (r *ArticleRepository) UpdateArticleViewCount(id int) {
	if err := r.db.Model(&Article{}).Where("id = ?", id).Update("views", gorm.Expr("views + 1")).Error; err != nil {
		logrus.Errorf("更新文章浏览次数失败: %d, 错误: %v", id, err)
		return
	}
	cacheKey := fmt.Sprintf("%s%d", PrefixArticle, id)
	if cached, found := r.cache.Get(cacheKey); found {
		if article, ok := cached.(Article); ok {
			article.Views++
			_ = r.cache.Set(cacheKey, article, ExpireShort)
			logrus.Debugf("更新缓存中的文章浏览次数: %d, 新浏览次数: %d", id, article.Views)
			return
		}
		logrus.Warnf("文章缓存类型转换失败: %d, 无法更新缓存中的浏览次数", id)
	} else {
		logrus.Debugf("缓存中未找到文章: %d, 跳过缓存更新", id)
	}
}

func (r *ArticleRepository) FindPage(ctx context.Context, page, pageSize int) ([]Article, int64, error) {
	cacheKey := fmt.Sprintf("%spage:%d:%d", PrefixArticleList, page, pageSize)
	cacheCountKey := fmt.Sprintf("%scount:%d:%d", PrefixArticleList, page, pageSize)
	if cached, found := r.cache.Get(cacheKey); found {
		if articles, ok := cached.([]Article); ok {
			if cachedTotal, foundTotal := r.cache.Get(cacheCountKey); foundTotal {
				if total, ok := cachedTotal.(int64); ok {
					logrus.Debugf("从缓存获取文章分页: %d, %d", page, pageSize)
					return articles, total, nil
				}
				logrus.Warn("文章总数缓存类型转换失败，将从数据库重新获取")
			}
		} else {
			logrus.Warn("文章列表缓存类型转换失败，将从数据库重新获取")
		}
	}
	articles, total, err := r.gormRepository.FindPage(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("获取文章分页失败: %w", err)
	}
	_ = r.cache.Set(cacheKey, articles, ExpireShort)
	_ = r.cache.Set(cacheCountKey, total, ExpireLong)
	logrus.Debugf("文章分页已缓存: %d, %d", page, pageSize)
	return articles, total, nil
}

func (r *ArticleRepository) Delete(ctx context.Context, id int) error {
	err := r.gormRepository.Delete(ctx, id)
	if err != nil {
		logrus.Errorf("删除文章失败: %d, 错误: %v", id, err)
		return err
	}
	cacheKey := fmt.Sprintf("%s%d", PrefixArticle, id)
	if deleted := r.cache.Delete(cacheKey); !deleted {
		logrus.Warnf("清除文章缓存失败: %d, 错误: 缓存中未找到", id)
	}
	r.clearArticleListCache()
	logrus.Debugf("已清除已删除文章的缓存: %d", id)
	return nil
}

func (r *ArticleRepository) clearArticleListCache() {
	cacheKey := fmt.Sprintf("%scount", PrefixArticleList)
	if deleted := r.cache.Delete(cacheKey); !deleted {
		logrus.Warn("清除文章列表缓存失败: 缓存中未找到")
	} else {
		logrus.Debug("已清除文章列表缓存")
	}
}

func countWords(content string) int { return len(strings.Fields(content)) }
