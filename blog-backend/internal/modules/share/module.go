package share

import (
	filesmodule "dh-blog/internal/modules/files"
	"dh-blog/internal/router"

	"gorm.io/gorm"
)

// Dependencies are the application-owned services needed by the share module.
type Dependencies struct {
	DB          *gorm.DB
	FileService filesmodule.Service
}

// Module owns share persistence, business logic, HTTP routes, and background cleanup.
type Module struct {
	repository          *Repository
	accessLogRepository *AccessLogRepository
	service             *shareService
	handler             *handler
}

func New(deps Dependencies) *Module {
	repository := newRepository(deps.DB)
	accessLogRepository := newAccessLogRepository(deps.DB)
	service := newService(repository, accessLogRepository, deps.FileService)
	return &Module{
		repository:          repository,
		accessLogRepository: accessLogRepository,
		service:             service,
		handler:             newHandler(service),
	}
}

// Service exposes share operations to application-level collaborators.
func (m *Module) Service() Service {
	return m.service
}

// MigrationModels declares the database tables owned by this module.
func MigrationModels() []any {
	return []any{&Share{}, &ShareAccessLog{}}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	sharePublicRoutes := routes.Engine.Group("/api/share")
	sharePublicRoutes.GET("/:shareId", m.handler.GetShareInfo)
	sharePublicRoutes.POST("/:shareId/verify", m.handler.VerifyPassword)
	sharePublicRoutes.GET("/:shareId/download", m.handler.Download)

	fileAPI := routes.AuthenticatedAPI("/api/files")
	fileAPI.POST("/share", m.handler.CreateShare)
	fileAPI.GET("/share", m.handler.ListShares)
	fileAPI.GET("/share/:id", m.handler.GetShareDetail)
	fileAPI.DELETE("/share/:id", m.handler.DeleteShare)
	fileAPI.GET("/share/:id/logs", m.handler.GetAccessLogs)
}

// Shutdown stops the share token cleanup worker. It is safe to call more than once.
func (m *Module) Shutdown() {
	if m != nil && m.service != nil {
		m.service.shutdown()
	}
}
