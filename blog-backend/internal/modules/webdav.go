package modules

import (
	"dh-blog/internal/controller"
	"dh-blog/internal/router"

	"github.com/sirupsen/logrus"
)

// WebDAVModule 注册 WebDAV 路由。
type WebDAVModule struct {
	enabled          bool
	prefix           string
	webDAVController *controller.WebDAVController
}

func NewWebDAVModule(enabled bool, prefix string, webDAVController *controller.WebDAVController) *WebDAVModule {
	return &WebDAVModule{
		enabled:          enabled,
		prefix:           prefix,
		webDAVController: webDAVController,
	}
}

func (m *WebDAVModule) RegisterRoutes(routes *router.Routes) {
	if !m.enabled {
		return
	}

	davHandler := m.webDAVController.ServeWebDAV()
	routes.Engine.Any(m.prefix+"/*path", davHandler)
	routes.Engine.Any(m.prefix, davHandler)

	webdavMethods := []string{"PROPFIND", "PROPPATCH", "MKCOL", "COPY", "MOVE", "LOCK", "UNLOCK"}
	for _, method := range webdavMethods {
		routes.Engine.Handle(method, m.prefix+"/*path", davHandler)
		routes.Engine.Handle(method, m.prefix, davHandler)
	}

	logrus.Infof("WebDAV 服务已启用，路径前缀: %s", m.prefix)
}
