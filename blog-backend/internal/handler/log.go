package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
		adminRouter.GET("/overview/visitLog", h.GetVisitLogs)
		adminRouter.GET("/stats/daily", h.GetDailyStats)
		adminRouter.GET("/stats/visits", h.GetVisitStatistics)                 // 新增访问统计接口
		adminRouter.GET("/stats/monthly", h.GetMonthlyVisitStats)              // 新增月度统计接口
		adminRouter.GET("/stats/daily-chart", h.GetDailyVisitStatsForLastDays) // 新增每日图表统计接口
	}

	// 额外添加IP封禁/解封路由，与前端一致
	ipRouter := router.Group("/admin/ip")
	{
		ipRouter.POST("/ban/:ip/:status", h.BanIP)
	}
}

// GetVisitStatistics 获取访问统计信息
func (h *LogHandler) GetVisitStatistics(c *gin.Context) {
	stats, err := h.logRepo.GetVisitStatistics()
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, stats)
}

func (h *LogHandler) GetVisitLogs(c *gin.Context) {
	page := h.GetQueryInt(c, "page", 1)
	pageSize := h.GetQueryInt(c, "pageSize", 10)

	// 获取日期范围参数
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// 如果没有提供日期范围，默认为过去7天
	var startDate, endDate time.Time
	var err error

	if startDateStr == "" || endDateStr == "" {
		// 默认为过去7天
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -7)
	} else {
		// 尝试多种日期格式解析
		formats := []string{
			"2006-1-2",
			"2006-01-02",
			"2006/1/2",
			"2006/01/02",
		}

		startDateParsed := false
		for _, format := range formats {
			startDate, err = time.Parse(format, startDateStr)
			if err == nil {
				startDateParsed = true
				break
			}
		}

		if !startDateParsed {
			logrus.Warnf("解析开始日期失败: %s，使用默认值", startDateStr)
			startDate = time.Now().AddDate(0, 0, -7)
		}

		endDateParsed := false
		for _, format := range formats {
			endDate, err = time.Parse(format, endDateStr)
			if err == nil {
				endDateParsed = true
				break
			}
		}

		if !endDateParsed {
			logrus.Warnf("解析结束日期失败: %s，使用默认值", endDateStr)
			endDate = time.Now()
		}

		// 确保结束日期包含整天
		endDate = endDate.Add(24*time.Hour - 1*time.Second)
	}

	// 获取按IP分组的访问统计
	stats, total, err := h.logRepo.GetIPVisitStats(page, pageSize, startDate, endDate)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.SuccessWithPage(c, stats, total, page)
}

func (h *LogHandler) BanIP(c *gin.Context) {
	// 从URL参数获取IP和状态
	ip := c.Param("ip")
	statusStr := c.Param("status")

	if ip == "" {
		c.JSON(http.StatusBadRequest, response.Error("IP参数不能为空"))
		return
	}

	// 检查IP当前是否被封禁
	isBanned, err := h.logRepo.IsIPBanned(ip)
	if err != nil {
		h.Error(c, err)
		return
	}

	// 如果status是undefined或空字符串，则根据当前状态取反操作
	if statusStr == "undefined" || statusStr == "" {
		// 如果当前已封禁，则解封；如果当前未封禁，则封禁
		if isBanned {
			// 已封禁，执行解封
			if err := h.logRepo.UnbanIP(ip); err != nil {
				h.Error(c, err)
				return
			}
			h.Success(c)
			return
		} else {
			// 未封禁，执行封禁
			expireTime := time.Now().Add(24 * time.Hour)
			reason := "管理员操作"
			if err := h.logRepo.BanIP(ip, reason, expireTime); err != nil {
				h.Error(c, err)
				return
			}
			h.Success(c)
			return
		}
	}

	// 如果提供了明确的status值，则按照status值操作
	status, err := strconv.Atoi(statusStr)
	if err != nil {
		// 无法解析为整数，忽略status参数，根据当前状态取反
		if isBanned {
			if err := h.logRepo.UnbanIP(ip); err != nil {
				h.Error(c, err)
				return
			}
		} else {
			expireTime := time.Now().Add(24 * time.Hour)
			reason := "管理员操作"
			if err := h.logRepo.BanIP(ip, reason, expireTime); err != nil {
				h.Error(c, err)
				return
			}
		}
		h.Success(c)
		return
	}

	// 根据status值执行相应操作
	if status == 1 {
		// status=1表示当前已封禁，需要解封
		if err := h.logRepo.UnbanIP(ip); err != nil {
			h.Error(c, err)
			return
		}
	} else {
		// status=0或其他值表示当前未封禁，需要封禁
		expireTime := time.Now().Add(24 * time.Hour)
		reason := "管理员操作"
		if err := h.logRepo.BanIP(ip, reason, expireTime); err != nil {
			h.Error(c, err)
			return
		}
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

// GetMonthlyVisitStats 获取月度访问统计数据
func (h *LogHandler) GetMonthlyVisitStats(c *gin.Context) {
	yearStr := c.Query("year")
	var year int
	var err error

	if yearStr != "" {
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Error("无效的年份格式"))
			return
		}
	} else {
		year = time.Now().Year()
	}

	stats, err := h.logRepo.GetMonthlyVisitStats(year)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.SuccessWithData(c, stats)
}

// GetDailyVisitStatsForLastDays 获取最近几天的访问统计数据
func (h *LogHandler) GetDailyVisitStatsForLastDays(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 30 // 默认获取最近30天的数据
	}

	stats, err := h.logRepo.GetDailyVisitStatsForLastDays(days)
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
