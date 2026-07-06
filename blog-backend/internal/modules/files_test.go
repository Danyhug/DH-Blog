package modules_test

import (
	"testing"

	"dh-blog/internal/controller"
	"dh-blog/internal/modules"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
)

func TestFileAndShareRoutesCanRegisterTogether(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	routes := &router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
		FileAPI:   engine.Group("/api/files"),
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("registering file and share routes panicked: %v", recovered)
		}
	}()

	modules.NewFilesModule(
		controller.NewFileController(nil),
		controller.NewChunkUploadController(nil, nil, nil),
		t.TempDir(),
		nil,
	).RegisterRoutes(routes)
	modules.NewShareModule(controller.NewShareController(nil)).RegisterRoutes(routes)
}
