package modules

import (
	"dh-blog/internal/controller"
	"dh-blog/internal/router"
)

// LoggingModule 注册访问日志和统计相关路由。
type LoggingModule struct {
	logController *controller.LogController
}

func NewLoggingModule(logController *controller.LogController) *LoggingModule {
	return &LoggingModule{logController: logController}
}

func (m *LoggingModule) RegisterRoutes(routes *router.Routes) {
	adminAPI := routes.AdminAPI
	adminAPI.GET("/log/overview/visitLog", m.logController.GetVisitLogs)
	adminAPI.GET("/stats/daily", m.logController.GetDailyStats)
	adminAPI.GET("/log/stats/visits", m.logController.GetVisitStatistics)
	adminAPI.GET("/log/stats/monthly", m.logController.GetMonthlyVisitStats)
	adminAPI.GET("/log/stats/daily-chart", m.logController.GetDailyVisitStatsForLastDays)
	adminAPI.POST("/ip/ban/:ip/:status", m.logController.BanIP)
}
