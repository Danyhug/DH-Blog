package user

import (
	"dh-blog/internal/router"

	"gorm.io/gorm"
)

// Module owns the user feature's persistence, HTTP handler, and routes.
type Module struct {
	repository *Repository
	handler    *Handler
}

func New(db *gorm.DB, tokens TokenGenerator) *Module {
	repository := NewRepository(db)
	return &Module{
		repository: repository,
		handler:    NewHandler(repository, tokens),
	}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	routes.PublicAPI.POST("/user/login", m.handler.Login)
	routes.PublicAPI.POST("/user/check", m.handler.Check)
	routes.PublicAPI.GET("/user/heart", m.handler.Heart)
}

func (m *Module) Authenticate(username, password string) bool {
	return m.repository.Authenticate(username, password)
}

// MigrationModels declares the database tables owned by this module.
func MigrationModels() []any {
	return []any{&User{}}
}
