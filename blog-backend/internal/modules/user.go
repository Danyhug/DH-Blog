package modules

import (
	"dh-blog/internal/controller"
	"dh-blog/internal/router"
)

// UserModule 注册用户认证相关路由。
type UserModule struct {
	userController *controller.UserController
}

func NewUserModule(userController *controller.UserController) *UserModule {
	return &UserModule{userController: userController}
}

func (m *UserModule) RegisterRoutes(routes *router.Routes) {
	publicAPI := routes.PublicAPI
	publicAPI.POST("/user/login", m.userController.Login)
	publicAPI.POST("/user/check", m.userController.Check)
	publicAPI.GET("/user/heart", m.userController.Heart)
}
