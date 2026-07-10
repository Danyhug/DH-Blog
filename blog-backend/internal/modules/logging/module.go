package logging

import (
	"dh-blog/internal/dhcache"
	"dh-blog/internal/middleware"
	"dh-blog/internal/router"

	"gorm.io/gorm"
)

// Module assembles access logging, IP banning, and the related admin routes.
type Module struct {
	repository *Repository
	handler    *handler
	ipService  *ipService
}

// New constructs the complete logging module from infrastructure dependencies.
func New(db *gorm.DB, cache dhcache.Cache) *Module {
	repository := newRepository(db, cache)
	return &Module{
		repository: repository,
		handler:    newHandler(repository),
		ipService:  newIPService(repository),
	}
}

// IPService exposes only the operations required by the global IP middleware.
func (m *Module) IPService() middleware.IPService {
	return m.ipService
}

// MigrationModels declares the database tables owned by this module.
func MigrationModels() []any {
	return []any{&AccessLog{}, &IPBlacklist{}}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	adminAPI := routes.AdminAPI
	adminAPI.GET("/log/overview/visitLog", m.handler.GetVisitLogs)
	adminAPI.GET("/stats/daily", m.handler.GetDailyStats)
	adminAPI.GET("/log/stats/visits", m.handler.GetVisitStatistics)
	adminAPI.GET("/log/stats/monthly", m.handler.GetMonthlyVisitStats)
	adminAPI.GET("/log/stats/daily-chart", m.handler.GetDailyVisitStatsForLastDays)
	adminAPI.POST("/ip/ban/:ip/:status", m.handler.BanIP)
}
