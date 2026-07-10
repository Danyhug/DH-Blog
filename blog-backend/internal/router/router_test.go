package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"dh-blog/internal/config"
	"dh-blog/internal/middleware"
	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
)

type ipServiceStub struct{}

func (ipServiceStub) RecordRequest(middleware.AccessRecord) error { return nil }
func (ipServiceStub) IsIPBanned(string) (bool, error)             { return false, nil }

type routeModuleStub struct{}

func (routeModuleStub) RegisterRoutes(routes *router.Routes) {
	routes.PublicAPI.GET("/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	routes.AdminAPI.GET("/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	routes.AuthenticatedAPI("/api/private").GET("/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })
}

func TestPublicAndAuthenticatedRouteBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := router.Init(router.Options{Config: config.DefaultConfig(), IPService: ipServiceStub{}}, routeModuleStub{})

	for _, test := range []struct {
		path string
		want int
	}{
		{path: "/api/ping", want: http.StatusNoContent},
		{path: "/api/admin/ping", want: http.StatusUnauthorized},
		{path: "/api/private/ping", want: http.StatusUnauthorized},
	} {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		request.RemoteAddr = "127.0.0.1:1234"
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Errorf("GET %s status = %d, want %d", test.path, response.Code, test.want)
		}
	}
}
