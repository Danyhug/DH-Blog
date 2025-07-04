package handler

import (
	"net/http"
	"strconv"
	"time"

	"dh-blog/internal/errs"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	logRepo       *repository.LogRepository
	dailyStatRepo *repository.DailyStatsRepository
}

func NewLogHandler(logRepo *repository.LogRepository, dailyStatRepo *repository.DailyStatsRepository) *LogHandler {
	return &LogHandler{logRepo: logRepo, dailyStatRepo: dailyStatRepo}
}

// GetAccessLogs 获取访问日志列表
func (h *LogHandler) GetAccessLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	logs, total, err := h.logRepo.GetAccessLogs(page, pageSize, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取访问日志失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), logs)))
}

// GetIPStats 获取 IP 统计列表
func (h *LogHandler) GetIPStats(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	stats, total, err := h.logRepo.GetIPStats(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取 IP 统计失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), stats)))
}

// BanIP 封禁或解封 IP
func (h *LogHandler) BanIP(c *gin.Context) {
	ipAddress := c.Param("ip")
	statusStr := c.Param("status")

	if statusStr == "" {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("状态不能为空", nil).Error()))
		return
	}

	status := statusStr == "1"

	if err := h.logRepo.UpdateIPBanStatus(ipAddress, status); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("IP 状态更新失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

// GetVisitLogs 获取访问日志概览（每日统计）
func (h *LogHandler) GetVisitLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "30"))

	var startDate *time.Time
	var endDate *time.Time

	startDateStr := c.Query("startDate")
	if startDateStr != "" {
		parsedDate, err := time.Parse("2006-1-2", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的开始日期格式，应为 YYYY-M-D", err).Error()))
			return
		}
		startDate = &parsedDate
	}

	endDateStr := c.Query("endDate")
	if endDateStr != "" {
		parsedDate, err := time.Parse("2006-1-2", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的结束日期格式，应为 YYYY-M-D", err).Error()))
			return
		}
		// 确保结束日期包含当天所有时间
		modifiedDate := parsedDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		endDate = &modifiedDate
	}

	logs, total, err := h.dailyStatRepo.GetDailyStatsByDateRange(page, pageSize, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取访问日志概览失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), logs)))
}
