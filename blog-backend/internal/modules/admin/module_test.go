package admin

import (
	"testing"

	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
)

func TestModuleRegistersUploadRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	routes := &router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
	}
	New(nil).RegisterRoutes(routes)

	registered := engine.Routes()
	if len(registered) != 1 || registered[0].Method != "POST" || registered[0].Path != "/api/admin/upload/:type" {
		t.Fatalf("registered routes = %#v", registered)
	}
}

func TestUserIDFromContext(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
		want  uint64
	}{
		{name: "jwt number", value: float64(7), want: 7},
		{name: "uint64", value: uint64(9), want: 9},
		{name: "invalid", value: "10", want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(nil)
			context.Set("userID", test.value)
			if got := userIDFromContext(context); got != test.want {
				t.Fatalf("userIDFromContext() = %d, want %d", got, test.want)
			}
		})
	}
}
