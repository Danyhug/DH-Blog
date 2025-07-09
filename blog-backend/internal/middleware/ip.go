package middleware

import (
	"strings"
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/service"
	"dh-blog/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func getResourceType(path string) string {
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
func IPMiddleware(ipService service.IPService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端IP
		ip := utils.GetClientIP(c.Request)

		go func() {
			os, browser := utils.ParseUserAgent(c.Request.UserAgent())
			ua := os + "; " + browser

			// 获取IP所在城市
			city, err := utils.GetIPLocation(ip)
			if err != nil {
				logrus.Warnf("获取IP地理位置信息失败: %v", err)
				city = "未知/未知"
			}

			// 如果是本地网络，格式化为符合前端期望的格式
			if city == "本地网络" {
				city = "本地网络/本地/内网"
			}

			// 创建访问日志
			log := &model.AccessLog{
				IPAddress:    ip,
				AccessDate:   time.Now(),
				UserAgent:    ua,
				RequestURL:   c.Request.URL.String(),
				City:         city,
				ResourceType: getResourceType(c.Request.URL.Path),
			}

			err = ipService.RecordRequest(log)
			if err != nil {
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
