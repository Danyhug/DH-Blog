package webdav

import (
	"context"
	"net/http"
	"os"
	"strings"

	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/webdav"
)

const basicAuthRealm = `Basic realm="DH-Blog WebDAV"`

// tempDirName 是 files 模块分片上传的临时目录，必须对 WebDAV 客户端隐藏，
// 否则其他用户的上传分片与会话信息可被任意浏览、篡改或删除。
const tempDirName = "temp"

// tempFilterFS 包装存储根目录，屏蔽分片上传的 temp 目录。
type tempFilterFS struct {
	webdav.Dir
}

// blockedTempPath 判断 WebDAV 路径（'/' 分隔）是否指向 temp 目录或其内部。
func blockedTempPath(name string) bool {
	clean := strings.Trim(name, "/")
	return clean == tempDirName || strings.HasPrefix(clean, tempDirName+"/")
}

func (f tempFilterFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if blockedTempPath(name) {
		return os.ErrNotExist
	}
	return f.Dir.Mkdir(ctx, name, perm)
}

func (f tempFilterFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if blockedTempPath(name) {
		return nil, os.ErrNotExist
	}
	return f.Dir.OpenFile(ctx, name, flag, perm)
}

func (f tempFilterFS) RemoveAll(ctx context.Context, name string) error {
	if blockedTempPath(name) {
		return os.ErrNotExist
	}
	return f.Dir.RemoveAll(ctx, name)
}

func (f tempFilterFS) Rename(ctx context.Context, oldName, newName string) error {
	if blockedTempPath(oldName) || blockedTempPath(newName) {
		return os.ErrNotExist
	}
	return f.Dir.Rename(ctx, oldName, newName)
}

func (f tempFilterFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if blockedTempPath(name) {
		return nil, os.ErrNotExist
	}
	return f.Dir.Stat(ctx, name)
}

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
			FileSystem: tempFilterFS{Dir: webdav.Dir(storagePath)},
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

// Ensure the wrapper satisfies the FileSystem contract at compile time.
var _ webdav.FileSystem = tempFilterFS{}
