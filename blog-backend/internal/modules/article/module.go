package article

import (
	"context"
	"fmt"
	"time"

	"dh-blog/internal/router"

	"gorm.io/gorm"
)

// Cache is the subset of cache operations owned by article persistence.
type Cache interface {
	Set(key string, value interface{}, duration ...time.Duration) error
	Get(key string) (interface{}, bool)
	Delete(key string) bool
}

// AIService is the article module's narrow AI dependency.
type AIService interface {
	GenerateTags(text string, existingTags []string) ([]string, error)
}

// CommentCounter provides the cross-module statistic used by the overview.
type CommentCounter interface {
	Count(ctx context.Context) (int64, error)
}

// TagGenerationHandler is registered with the background task scheduler.
type TagGenerationHandler = func(ctx context.Context, articleID int, content string) error

// TagTaskScheduler is the narrow scheduling port used by the article module.
// The scheduler owns queueing and retries; the article module owns the work.
type TagTaskScheduler interface {
	RegisterTagGenerationHandler(handler TagGenerationHandler)
	SubmitTagGeneration(articleID int, content string)
}

// Dependencies contains only infrastructure and cross-module ports.
type Dependencies struct {
	DB             *gorm.DB
	Cache          Cache
	AI             AIService
	CommentCounter CommentCounter
	Tasks          TagTaskScheduler
}

// Module owns article, category, and tag persistence, handlers, and routes.
type Module struct {
	handler *Handler
}

// New assembles all repositories and handlers inside the vertical module.
func New(deps Dependencies) (*Module, error) {
	if deps.DB == nil {
		return nil, fmt.Errorf("article: DB is required")
	}
	if err := ensureDefaults(deps.DB); err != nil {
		return nil, err
	}
	tagRepository := NewTagRepository(deps.DB, deps.Cache)
	categoryRepository := NewCategoryRepository(deps.DB)
	articleRepository := NewArticleRepository(deps.DB, categoryRepository, tagRepository, deps.Cache)
	handler := NewHandler(articleRepository, tagRepository, categoryRepository, deps.CommentCounter, deps.AI, deps.Tasks)

	if deps.Tasks != nil {
		deps.Tasks.RegisterTagGenerationHandler(handler.ProcessTagGeneration)
	}

	return &Module{handler: handler}, nil
}

func ensureDefaults(db *gorm.DB) error {
	category := Category{Name: "默认分类", Slug: "default"}
	if err := db.Where("slug = ?", category.Slug).FirstOrCreate(&category).Error; err != nil {
		return fmt.Errorf("article: ensure default category: %w", err)
	}
	return nil
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	publicAPI := routes.PublicAPI
	publicAPI.GET("/article/:id", m.handler.GetArticleDetail)
	publicAPI.GET("/article/title/:id", m.handler.GetArticleTitle)
	publicAPI.GET("/article/unlock/:id/:password", m.handler.UnlockArticle)
	publicAPI.POST("/article/list", m.handler.GetPublicArticleList)
	publicAPI.GET("/article/overview", m.handler.GetOverview)
	publicAPI.GET("/article/tag", m.handler.GetAllTags)
	publicAPI.GET("/article/category", m.handler.GetAllCategories)
	publicAPI.GET("/article/taxonomies", m.handler.GetAllTaxonomies)
	publicAPI.GET("/article/taxonomy/articles", m.handler.GetArticlesByTaxonomy)

	adminAPI := routes.AdminAPI
	adminAPI.GET("/article/:id", m.handler.GetArticleDetail)
	adminAPI.POST("/article", m.handler.SaveArticle)
	adminAPI.PUT("/article", m.handler.UpdateArticle)
	adminAPI.POST("/article/list", m.handler.GetArticleList)
	adminAPI.POST("/article/:id/generate-tags", m.handler.GenerateTags)
	adminAPI.POST("/tag", m.handler.CreateTag)
	adminAPI.PUT("/tag", m.handler.UpdateTag)
	adminAPI.DELETE("/tag/:id", m.handler.DeleteTag)
	adminAPI.POST("/category", m.handler.CreateCategory)
	adminAPI.GET("/category/:id", m.handler.GetCategoryByID)
	adminAPI.PUT("/category", m.handler.UpdateCategory)
	adminAPI.DELETE("/category/:id", m.handler.DeleteCategory)
	adminAPI.GET("/category/:id/tags", m.handler.GetCategoryDefaultTags)
	adminAPI.POST("/category/:id/tags", m.handler.SaveCategoryDefaultTags)
}

// MigrationModels declares the tables owned by the article module.
func MigrationModels() []any {
	return []any{&Article{}, &Category{}, &Tag{}, &TagRelation{}}
}
