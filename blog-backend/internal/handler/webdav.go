package handler

import (
	"net/http"

	"dh-blog/internal/repository"
	"dh-blog/internal/service"
	"dh-blog/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/webdav"
)

// writeMethods 会修改文件系统的 WebDAV/HTTP 方法
var writeMethods = map[string]bool{
	"PUT":       true,
	"DELETE":    true,
	"MKCOL":     true,
	"COPY":      true,
	"MOVE":      true,
	"PROPPATCH": true,
	"POST":      true,
}

// WebDAVHandler WebDAV 协议处理器
type WebDAVHandler struct {
	userRepo    *repository.UserRepository
	fileService service.IFileService
	prefix      string
	lockSystem  webdav.LockSystem // 全局共享的锁系统，保证 LOCK/UNLOCK 正常工作
}

// NewWebDAVHandler 创建 WebDAV 处理器
func NewWebDAVHandler(userRepo *repository.UserRepository, fileService service.IFileService, prefix string) *WebDAVHandler {
	return &WebDAVHandler{
		userRepo:    userRepo,
		fileService: fileService,
		prefix:      prefix,
		lockSystem:  webdav.NewMemLS(),
	}
}

// ServeWebDAV 返回处理 WebDAV 请求的 gin.HandlerFunc
func (h *WebDAVHandler) ServeWebDAV() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Basic Auth 认证
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="DH-Blog WebDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 验证用户名密码
		user, err := h.userRepo.GetUserByUsername(username)
		if err != nil {
			logrus.Debugf("WebDAV 认证失败，用户不存在: %s", username)
			c.Header("WWW-Authenticate", `Basic realm="DH-Blog WebDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if !utils.CheckPasswordHash(password, user.Password) {
			logrus.Debugf("WebDAV 认证失败，密码错误: %s", username)
			c.Header("WWW-Authenticate", `Basic realm="DH-Blog WebDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 获取存储路径
		storagePath := h.fileService.GetStoragePath()
		if storagePath == "" {
			logrus.Error("WebDAV 存储路径为空")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// 使用共享的 LockSystem，FileSystem 根据当前存储路径动态创建
		davHandler := &webdav.Handler{
			Prefix:     h.prefix,
			FileSystem: webdav.Dir(storagePath),
			LockSystem: h.lockSystem,
			Logger: func(r *http.Request, err error) {
				if err != nil {
					logrus.Debugf("WebDAV %s %s: %v", r.Method, r.URL.Path, err)
				}
			},
		}

		davHandler.ServeHTTP(c.Writer, c.Request)

		// 写操作完成后，触发防抖同步（将磁盘变更同步到数据库）
		if writeMethods[c.Request.Method] && c.Writer.Status() < 400 {
			h.fileService.SyncFilesFromDiskDebounced()
		}
	}
}
