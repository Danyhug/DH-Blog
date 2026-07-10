package webdav

import (
	"net/http"

	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/webdav"
)

const basicAuthRealm = `Basic realm="DH-Blog WebDAV"`

var writeMethods = map[string]bool{
	http.MethodPut:    true,
	http.MethodDelete: true,
	"MKCOL":           true,
	"COPY":            true,
	"MOVE":            true,
	"PROPPATCH":       true,
	http.MethodPost:   true,
}

// UserAuthenticator is the only user capability WebDAV consumes.
type UserAuthenticator interface {
	Authenticate(username, password string) bool
}

// FileService is the storage capability WebDAV consumes from files.
type FileService interface {
	GetStoragePath() string
	SyncFilesFromDiskDebounced()
}

// Dependencies are the application-owned settings and collaborators WebDAV needs.
type Dependencies struct {
	Enabled bool
	Prefix  string
	Users   UserAuthenticator
	Files   FileService
}

// Module owns WebDAV authentication, filesystem serving, locking, and routes.
type Module struct {
	enabled    bool
	prefix     string
	users      UserAuthenticator
	files      FileService
	lockSystem webdav.LockSystem
}

func New(deps Dependencies) *Module {
	return &Module{
		enabled:    deps.Enabled,
		prefix:     deps.Prefix,
		users:      deps.Users,
		files:      deps.Files,
		lockSystem: webdav.NewMemLS(),
	}
}

func (m *Module) RegisterRoutes(routes *router.Routes) {
	if !m.enabled {
		return
	}

	handler := m.serveHTTP()
	routes.Engine.Any(m.prefix+"/*path", handler)
	routes.Engine.Any(m.prefix, handler)

	for _, method := range []string{"PROPFIND", "PROPPATCH", "MKCOL", "COPY", "MOVE", "LOCK", "UNLOCK"} {
		routes.Engine.Handle(method, m.prefix+"/*path", handler)
		routes.Engine.Handle(method, m.prefix, handler)
	}

	logrus.Infof("WebDAV 服务已启用，路径前缀: %s", m.prefix)
}

func (m *Module) serveHTTP() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			abortUnauthorized(c)
			return
		}

		if m.users == nil || !m.users.Authenticate(username, password) {
			logrus.Debugf("WebDAV 认证失败: %s", username)
			abortUnauthorized(c)
			return
		}

		storagePath := m.files.GetStoragePath()
		if storagePath == "" {
			logrus.Error("WebDAV 存储路径为空")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		davHandler := &webdav.Handler{
			Prefix:     m.prefix,
			FileSystem: webdav.Dir(storagePath),
			LockSystem: m.lockSystem,
			Logger: func(r *http.Request, err error) {
				if err != nil {
					logrus.Debugf("WebDAV %s %s: %v", r.Method, r.URL.Path, err)
				}
			},
		}

		davHandler.ServeHTTP(c.Writer, c.Request)

		if writeMethods[c.Request.Method] && c.Writer.Status() < http.StatusBadRequest {
			m.files.SyncFilesFromDiskDebounced()
		}
	}
}

func abortUnauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", basicAuthRealm)
	c.AbortWithStatus(http.StatusUnauthorized)
}
