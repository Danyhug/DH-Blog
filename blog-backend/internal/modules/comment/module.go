package comment

import (
	"context"

	"dh-blog/internal/router"

	"gorm.io/gorm"
)

// Module 装配评论模块并注册其 HTTP 路由。
type Module struct {
	repository *Repository
	handler    *handler
}

// New 使用数据库连接完成模块内部装配。
func New(db *gorm.DB) *Module {
	repository := newRepository(db)
	return &Module{
		repository: repository,
		handler:    newHandler(repository),
	}
}

// Count exposes the narrow statistic consumed by the article module.
func (m *Module) Count(ctx context.Context) (int64, error) {
	return m.repository.Count(ctx)
}

// MigrationModels declares the database tables owned by this module.
func MigrationModels() []any {
	return []any{&Comment{}}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	routes.PublicAPI.POST("/comment", m.handler.AddComment)
	routes.PublicAPI.GET("/comment/:articleId", m.handler.GetCommentsByArticleID)

	routes.AdminAPI.GET("/comment/:pageSize/:pageNum", m.handler.GetAllComments)
	routes.AdminAPI.PUT("/comment", m.handler.UpdateComment)
	routes.AdminAPI.POST("/comment/reply", m.handler.ReplyComment)
	routes.AdminAPI.DELETE("/comment/:id", m.handler.DeleteComment)
}
