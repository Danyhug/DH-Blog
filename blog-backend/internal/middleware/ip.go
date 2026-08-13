package middleware

import (
	"strings"
	"time"

	"dh-blog/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AccessRecord is the transport-neutral request data persisted by the logging
// module. Keeping it here prevents the global middleware from depending on a
// business module's database model. Geo-location is resolved by the logging
// module itself, not here.
type AccessRecord struct {
	IPAddress    string
	AccessDate   time.Time
	UserAgent    string
	RequestURL   string
	ResourceType string
}

// IPService is the narrow port required by IPMiddleware.
type IPService interface {
	RecordRequest(AccessRecord) error
	IsIPBanned(ip string) (bool, error)
}

// skippedResourceTypes are request kinds that must not reach the access log.
// Heartbeats are pure noise, and AI gateway traffic is machine-to-machine: it
// would drown the blog's visitor statistics and trigger one geo-IP lookup per
// agent call. The gateway keeps its own, richer request log instead.
var skippedResourceTypes = map[string]struct{}{
	"heartbeat": {},
	"gateway":   {},
}

func skipAccessLog(resourceType string) bool {
	_, skipped := skippedResourceTypes[resourceType]
	return skipped
}

func getResourceType(path string) string {
	if strings.HasPrefix(path, "/api/user/heart") {
		return "heartbeat"
	}
	// 确保路径长度足够并且包含/api/前缀
	if len(path) < 5 || !strings.HasPrefix(path, "/api/") {
		return ""
	}

	// 从/api/之后开始查找下一个/
	resourcePath := path[5:]
	idx := strings.Index(resourcePath, "/")

	if idx == -1 {
		// 如果没有后续的/，整个剩余部分就是资源类型
		return resourcePath
	}

	// 提取/api/和下一个/之间的部分
	return resourcePath[:idx]
}

// IPMiddleware 客户端 IP 中间件
func IPMiddleware(ipService IPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端IP
		ip := utils.GetClientIP(c.Request)
		resourceType := getResourceType(c.Request.URL.Path)
		userAgent := c.Request.UserAgent()
		requestURL := c.Request.URL.String()

		go func() {
			if skipAccessLog(resourceType) {
				return
			}
			os, browser := utils.ParseUserAgent(userAgent)
			ua := os + "; " + browser

			// 创建访问日志。City 归属地由 logging 模块负责解析与缓存
			record := AccessRecord{
				IPAddress:    ip,
				AccessDate:   time.Now(),
				UserAgent:    ua,
				RequestURL:   requestURL,
				ResourceType: resourceType,
			}

			if err := ipService.RecordRequest(record); err != nil {
				logrus.Errorf("保存访问日志时出错: %v", err)
				return
			}
		}()

		// 检查该IP是否被封禁
		banned, err := ipService.IsIPBanned(ip)
		if err != nil {
			logrus.Errorf("检查IP封禁状态时出错: %v", err)
			c.JSON(403, "未知的错误")
			c.Abort()
			return
		}

		if banned {
			c.JSON(403, "您已被封禁")
			c.Abort()
			return
		}

		// 继续执行下一个处理器
		c.Next()
	}
}
