package modules

import (
	"dh-blog/internal/controller"
	"dh-blog/internal/router"
)

// AdminModule 注册后台通用管理路由。
type AdminModule struct {
	adminController *controller.AdminController
}

func NewAdminModule(adminController *controller.AdminController) *AdminModule {
	return &AdminModule{adminController: adminController}
}

func (m *AdminModule) RegisterRoutes(routes *router.Routes) {
	routes.AdminAPI.POST("/upload/:type", m.adminController.UploadFile)
}
