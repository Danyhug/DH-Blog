package admin

import (
	filesmodule "dh-blog/internal/modules/files"
	"dh-blog/internal/router"
)

type Module struct {
	handler *handler
}

func New(fileService filesmodule.Service) *Module {
	return &Module{handler: newHandler(fileService)}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	routes.AdminAPI.POST("/upload/:type", m.handler.UploadFile)
}
