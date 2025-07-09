package handler

import (
	"errors"
	"net/http"
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"github.com/gin-gonic/gin"
)

// 日志相关错误
var (
	ErrInvalidDateFormat = errors.New("无效的日期格式")
)

type LogHandler struct {
	BaseHandler
	logRepo *repository.LogRepository
}

func NewLogHandler(logRepo *repository.LogRepository) *LogHandler {
	return &LogHandler{logRepo: logRepo}
}

func (h *LogHandler) RegisterRoutes(router *gin.RouterGroup) {
	adminRouter := router.Group("/admin/log")
	{
		adminRouter.GET("/visits", h.GetVisitLogs)
		adminRouter.POST("/ip/ban", h.BanIP)
		adminRouter.POST("/ip/unban", h.UnbanIP)
		adminRouter.GET("/stats/daily", h.GetDailyStats)
	}
}

func (h *LogHandler) GetVisitLogs(c *gin.Context) {
	page := h.GetQueryInt(c, "page", 1)
	pageSize := h.GetQueryInt(c, "pageSize", 10)

	logs, total, err := h.logRepo.GetVisitLogs(page, pageSize)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.SuccessWithPage(c, logs, total, page)
}

func (h *LogHandler) BanIP(c *gin.Context) {
	var req struct {
		IP       string `json:"ip" binding:"required"`
		Reason   string `json:"reason"`
		Duration int    `json:"duration"` // in hours
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	var expireTime time.Time
	if req.Duration > 0 {
		expireTime = time.Now().Add(time.Hour * time.Duration(req.Duration))
	}

	if err := h.logRepo.BanIP(req.IP, req.Reason, expireTime); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *LogHandler) UnbanIP(c *gin.Context) {
	var req struct {
		IP string `json:"ip" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	if err := h.logRepo.UnbanIP(req.IP); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *LogHandler) GetDailyStats(c *gin.Context) {
	var req struct {
		StartDate string `form:"startDate" binding:"required"`
		EndDate   string `form:"endDate" binding:"required"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的查询参数: "+err.Error()))
		return
	}

	startDate, err1 := time.Parse("2006-01-02", req.StartDate)
	endDate, err2 := time.Parse("2006-01-02", req.EndDate)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, response.Error(ErrInvalidDateFormat.Error()))
		return
	}

	stats, err := h.logRepo.GetDailyVisitStats(startDate, endDate)
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, stats)
}

// SaveAccessLog a middleware to save access log
func (h *LogHandler) SaveAccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		log := &model.AccessLog{
			IPAddress:  c.ClientIP(),
			AccessDate: time.Now().Truncate(24 * time.Hour),
			UserAgent:  c.Request.UserAgent(),
			RequestURL: c.Request.URL.String(),
		}

		// You can add more logic here to get city, resourceType, resourceId

		if err := h.logRepo.SaveAccessLog(log); err != nil {
			// Log error
		}
	}
}
