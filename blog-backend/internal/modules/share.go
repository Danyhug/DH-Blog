package modules

import (
	"dh-blog/internal/controller"
	"dh-blog/internal/router"
)

// ShareModule 注册公开分享访问和分享管理路由。
type ShareModule struct {
	shareController *controller.ShareController
}

func NewShareModule(shareController *controller.ShareController) *ShareModule {
	return &ShareModule{shareController: shareController}
}

func (m *ShareModule) RegisterRoutes(routes *router.Routes) {
	sharePublicRoutes := routes.Engine.Group("/api/share")
	sharePublicRoutes.GET("/:shareId", m.shareController.GetShareInfo)
	sharePublicRoutes.POST("/:shareId/verify", m.shareController.VerifyPassword)
	sharePublicRoutes.GET("/:shareId/download", m.shareController.Download)

	fileAPI := routes.FileAPI
	fileAPI.POST("/share", m.shareController.CreateShare)
	fileAPI.GET("/share", m.shareController.ListShares)
	fileAPI.GET("/share/:id", m.shareController.GetShareDetail)
	fileAPI.DELETE("/share/:id", m.shareController.DeleteShare)
	fileAPI.GET("/share/:id/logs", m.shareController.GetAccessLogs)
}
