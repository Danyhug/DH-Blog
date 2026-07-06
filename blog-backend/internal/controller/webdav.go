package controller

import (
	"net/http"

	"dh-blog/internal/repository"
	"dh-blog/internal/service"
	"dh-blog/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/webdav"
)

// WebDAVWriteMethods 会修改文件系统的 WebDAV/HTTP 方法。
var WebDAVWriteMethods = map[string]bool{
	"PUT":       true,
	"DELETE":    true,
	"MKCOL":     true,
	"COPY":      true,
	"MOVE":      true,
	"PROPPATCH": true,
	"POST":      true,
}

type WebDAVController struct {
	userRepo    *repository.UserRepository
	fileService service.IFileService
	prefix      string
	lockSystem  webdav.LockSystem
}

func NewWebDAVController(userRepo *repository.UserRepository, fileService service.IFileService, prefix string) *WebDAVController {
	return &WebDAVController{
		userRepo:    userRepo,
		fileService: fileService,
		prefix:      prefix,
		lockSystem:  webdav.NewMemLS(),
	}
}

func (h *WebDAVController) ServeWebDAV() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="DH-Blog WebDAV"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

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

		storagePath := h.fileService.GetStoragePath()
		if storagePath == "" {
			logrus.Error("WebDAV 存储路径为空")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

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

		if WebDAVWriteMethods[c.Request.Method] && c.Writer.Status() < 400 {
			h.fileService.SyncFilesFromDiskDebounced()
		}
	}
}
