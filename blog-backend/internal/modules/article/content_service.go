package article

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dh-blog/internal/model"

	"gorm.io/gorm"
)

// ContentService 是把 article 仓储能力以接口形式暴露给跨模块使用方（agentapi）的端口。
// 使用方只需要这套方法，不知道 article 内部 repository 的存在。
type ContentService interface {
	List(ctx context.Context, keyword string, page, pageSize int) ([]ArticleBrief, int64, error)
	Get(ctx context.Context, id int) (*ArticleDetail, error)
	Create(ctx context.Context, input CreateInput) (int, error)
	Update(ctx context.Context, input UpdateInput) error
}

// ArticleBrief 是列表页的轻量视图，分类名与标签名一次批量解析，避免 N+1。
type ArticleBrief struct {
	ID           int
	Title        string
	CategoryName string
	Tags         []string
	Summary      string
	WordNum      int
	CreatedAt    model.JSONTime
	AuthorType   string
	AuthorName   string
	AuthorKeyID  int
}

// ArticleDetail 是单篇文章的完整视图，正文、分类名、标签与作者信息都解析好。
type ArticleDetail struct {
	ID           int
	Title        string
	Content      string
	Summary      string
	CategoryID   int
	CategoryName string
	Tags         []string
	WordNum      int
	ThumbnailURL string
	IsLocked     bool
	CreatedAt    model.JSONTime
	AuthorType   string
	AuthorName   string
	AuthorKeyID  int
}

// CreateInput 携带新建文章所需的全部信息。CategoryName 为空表示不归类。
type CreateInput struct {
	Title        string
	Content      string
	Summary      string
	CategoryName string
	Tags         []string
	ThumbnailURL string
	AuthorType   string
	AuthorName   string
	AuthorKeyID  int
}

// UpdateInput 只携带要改的字段，nil 表示不改。作者身份在创建时固化，不走这里。
type UpdateInput struct {
	ID           int
	Title        *string
	Content      *string
	Summary      *string
	CategoryName *string
	ThumbnailURL *string
	Tags         *[]string
}

// contentService 持有 repository 指针与 db：标签 preload 走 repository 的缓存链路，
// 分类名解析与列表的批量查询走 db，避免把查询细节塞进导出接口。
type contentService struct {
	repository *ArticleRepository
	db         *gorm.DB
}

func newContentService(repository *ArticleRepository, db *gorm.DB) *contentService {
	return &contentService{repository: repository, db: db}
}

func (s *contentService) Create(ctx context.Context, input CreateInput) (int, error) {
	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)
	if title == "" {
		return 0, errors.New("文章标题不能为空")
	}
	if content == "" {
		return 0, errors.New("文章内容不能为空")
	}

	// 重复保护：同标题文章（未删除的）已存在时不静默重复创建，
	// 让 Agent 改用 update_article 继续。
	var count int64
	if err := s.db.WithContext(ctx).Model(&Article{}).Where("title = ?", title).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("检查文章标题重复失败: %w", err)
	}
	if count > 0 {
		return 0, fmt.Errorf("已存在同标题文章（%s），请改用 update_article 更新", title)
	}

	var categoryID int
	if strings.TrimSpace(input.CategoryName) != "" {
		category, err := s.findCategory(input.CategoryName)
		if err != nil {
			return 0, err
		}
		categoryID = category.ID
	}

	article := &Article{
		Title:        title,
		Content:      content,
		Summary:      strings.TrimSpace(input.Summary),
		CategoryID:   categoryID,
		ThumbnailURL: strings.TrimSpace(input.ThumbnailURL),
		TagNames:     input.Tags,
		AuthorType:   input.AuthorType,
		AuthorName:   input.AuthorName,
		AuthorKeyID:  input.AuthorKeyID,
	}
	if err := s.repository.SaveArticle(article); err != nil {
		return 0, err
	}
	return article.ID, nil
}

func (s *contentService) Get(ctx context.Context, id int) (*ArticleDetail, error) {
	article, err := s.repository.GetArticleById(id)
	if err != nil {
		if errors.Is(err, ErrArticleNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	categoryName := ""
	if article.CategoryID > 0 {
		category, err := s.categoryByID(article.CategoryID)
		if err != nil {
			return nil, err
		}
		categoryName = category.Name
	}

	return &ArticleDetail{
		ID:           article.ID,
		Title:        article.Title,
		Content:      article.Content,
		Summary:      article.Summary,
		CategoryID:   article.CategoryID,
		CategoryName: categoryName,
		Tags:         tagNames(article.Tags),
		WordNum:      article.WordNum,
		ThumbnailURL: article.ThumbnailURL,
		IsLocked:     article.IsLocked,
		CreatedAt:    article.CreatedAt,
		AuthorType:   article.AuthorType,
		AuthorName:   article.AuthorName,
		AuthorKeyID:  article.AuthorKeyID,
	}, nil
}

func (s *contentService) Update(ctx context.Context, input UpdateInput) error {
	// GetArticleById 的加载顺带刷掉可能存在的文章缓存，而后 UpdateArticle 全量
	// Save 只在旧数据之上覆盖传进来的字段，未传的字段自然保持原值。
	current, err := s.repository.GetArticleById(input.ID)
	if err != nil {
		if errors.Is(err, ErrArticleNotFound) {
			return ErrArticleNotFound
		}
		return err
	}

	next := current
	if input.Title != nil {
		next.Title = *input.Title
	}
	if input.Content != nil {
		next.Content = *input.Content
	}
	if input.Summary != nil {
		next.Summary = *input.Summary
	}
	if input.ThumbnailURL != nil {
		next.ThumbnailURL = *input.ThumbnailURL
	}
	if input.CategoryName != nil {
		category, err := s.findCategory(*input.CategoryName)
		if err != nil {
			return err
		}
		next.CategoryID = category.ID
	}
	if input.Tags != nil {
		next.TagNames = *input.Tags
	} else {
		next.TagNames = tagNames(current.Tags)
	}

	if err := s.repository.UpdateArticle(&next); err != nil {
		return err
	}
	return nil
}

func (s *contentService) List(ctx context.Context, keyword string, page, pageSize int) ([]ArticleBrief, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}

	query := s.db.WithContext(ctx).Model(&Article{})
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计文章总数失败: %w", err)
	}

	var articles []Article
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&articles).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章列表失败: %w", err)
	}

	tagByArticle := s.tagsByArticleIDs(ctx, articleIDs(articles))
	categoryByID := s.categoriesByIDs(ctx, articleCategoryIDs(articles))

	briefs := make([]ArticleBrief, 0, len(articles))
	for i := range articles {
		a := &articles[i]
		briefs = append(briefs, ArticleBrief{
			ID:           a.ID,
			Title:        a.Title,
			CategoryName: categoryByID[a.CategoryID].Name,
			Tags:         tagByArticle[a.ID],
			Summary:      a.Summary,
			WordNum:      a.WordNum,
			CreatedAt:    a.CreatedAt,
			AuthorType:   a.AuthorType,
			AuthorName:   a.AuthorName,
			AuthorKeyID:  a.AuthorKeyID,
		})
	}
	return briefs, total, nil
}

// findCategory 按 name 或 slug 解析分类，未命中时报错并列出按名字排序的可用分类。
func (s *contentService) findCategory(nameOrSlug string) (*Category, error) {
	var category Category
	err := s.db.Where("name = ? OR slug = ?", nameOrSlug, nameOrSlug).First(&category).Error
	if err == nil {
		return &category, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询分类失败: %w", err)
	}

	available, err := s.availableCategoryNames()
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("分类不存在: %s。可用分类：%s", nameOrSlug, strings.Join(available, "、"))
}

func (s *contentService) availableCategoryNames() ([]string, error) {
	var categories []Category
	if err := s.db.Order("name").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("查询可用分类失败: %w", err)
	}
	names := make([]string, 0, len(categories))
	for i := range categories {
		if strings.TrimSpace(categories[i].Name) != "" {
			names = append(names, categories[i].Name)
		}
	}
	return names, nil
}

func (s *contentService) categoryByID(id int) (*Category, error) {
	var category Category
	if err := s.db.First(&category, id).Error; err != nil {
		return nil, fmt.Errorf("查询分类失败: %w", err)
	}
	return &category, nil
}

// tagsByArticleIDs 一次 IN 查询把所有文章的标签带回，避免逐篇 N+1。
func (s *contentService) tagsByArticleIDs(ctx context.Context, ids []int) map[int][]string {
	result := make(map[int][]string, len(ids))
	if len(ids) == 0 {
		return result
	}
	for _, id := range ids {
		result[id] = []string{}
	}
	var rows []struct {
		ArticleID int
		Name      string
	}
	if err := s.db.WithContext(ctx).
		Table("tags").
		Joins("JOIN article_tags ON article_tags.tag_id = tags.id").
		Where("article_tags.article_id IN ?", ids).
		Select("article_tags.article_id, tags.name").
		Scan(&rows).Error; err != nil {
		return result
	}
	for _, row := range rows {
		names := result[row.ArticleID]
		names = append(names, row.Name)
		result[row.ArticleID] = names
	}
	return result
}

// categoriesByIDs 一次 IN 查询把文章涉及的分类上下文带回；
// 查不到的（分类被删 / 无分类）留空，列表不因此报错。
func (s *contentService) categoriesByIDs(ctx context.Context, ids []int) map[int]Category {
	result := make(map[int]Category, len(ids))
	if len(ids) == 0 {
		return result
	}
	var categories []Category
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&categories).Error; err != nil {
		return result
	}
	for i := range categories {
		result[categories[i].ID] = categories[i]
	}
	return result
}

func articleIDs(articles []Article) []int {
	ids := make([]int, 0, len(articles))
	for i := range articles {
		ids = append(ids, articles[i].ID)
	}
	return ids
}

func articleCategoryIDs(articles []Article) []int {
	ids := make([]int, 0, len(articles))
	for i := range articles {
		if articles[i].CategoryID > 0 {
			ids = append(ids, articles[i].CategoryID)
		}
	}
	return ids
}

func tagNames(tags []*Tag) []string {
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag != nil {
			names = append(names, tag.Name)
		}
	}
	return names
}