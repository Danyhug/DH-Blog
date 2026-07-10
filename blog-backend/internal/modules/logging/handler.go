package logging

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var ErrInvalidDateFormat = errors.New("无效的日期格式")

type handler struct {
	repository *Repository
}

func newHandler(repository *Repository) *handler {
	return &handler{repository: repository}
}

func (h *handler) GetVisitStatistics(c *gin.Context) {
	stats, err := h.repository.GetVisitStatistics()
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, stats)
}

func (h *handler) GetVisitLogs(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "pageSize", 10)
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	var startDate, endDate time.Time
	var err error
	if startDateStr == "" || endDateStr == "" {
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -7)
	} else {
		formats := []string{"2006-1-2", "2006-01-02", "2006/1/2", "2006/01/02"}
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
		endDate = endDate.Add(24*time.Hour - time.Second)
	}

	stats, total, err := h.repository.GetIPVisitStats(page, pageSize, startDate, endDate)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), stats)))
}

func (h *handler) BanIP(c *gin.Context) {
	ip := c.Param("ip")
	statusStr := c.Param("status")
	if ip == "" {
		c.JSON(http.StatusBadRequest, response.Error("IP参数不能为空"))
		return
	}

	isBanned, err := h.repository.IsIPBanned(ip)
	if err != nil {
		respondError(c, err)
		return
	}

	if statusStr == "undefined" || statusStr == "" {
		h.toggleIPBan(c, ip, isBanned)
		return
	}
	status, err := strconv.Atoi(statusStr)
	if err != nil {
		h.toggleIPBan(c, ip, isBanned)
		return
	}
	if status == 1 {
		err = h.repository.UnbanIP(ip)
	} else {
		err = h.repository.BanIP(ip, "管理员操作", time.Now().Add(24*time.Hour))
	}
	if err != nil {
		respondError(c, err)
		return
	}
	respondSuccess(c)
}

func (h *handler) toggleIPBan(c *gin.Context, ip string, isBanned bool) {
	var err error
	if isBanned {
		err = h.repository.UnbanIP(ip)
	} else {
		err = h.repository.BanIP(ip, "管理员操作", time.Now().Add(24*time.Hour))
	}
	if err != nil {
		respondError(c, err)
		return
	}
	respondSuccess(c)
}

func (h *handler) GetDailyStats(c *gin.Context) {
	var request struct {
		StartDate string `form:"startDate" binding:"required"`
		EndDate   string `form:"endDate" binding:"required"`
	}
	if err := c.ShouldBindQuery(&request); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的查询参数: "+err.Error()))
		return
	}
	startDate, startErr := time.Parse("2006-01-02", request.StartDate)
	endDate, endErr := time.Parse("2006-01-02", request.EndDate)
	if startErr != nil || endErr != nil {
		c.JSON(http.StatusBadRequest, response.Error(ErrInvalidDateFormat.Error()))
		return
	}
	stats, err := h.repository.GetDailyVisitStats(startDate, endDate)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, stats)
}

func (h *handler) GetMonthlyVisitStats(c *gin.Context) {
	year := time.Now().Year()
	if yearStr := c.Query("year"); yearStr != "" {
		parsedYear, err := strconv.Atoi(yearStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.Error("无效的年份格式"))
			return
		}
		year = parsedYear
	}
	stats, err := h.repository.GetMonthlyVisitStats(year)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, stats)
}

func (h *handler) GetDailyVisitStatsForLastDays(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days <= 0 {
		days = 30
	}
	stats, err := h.repository.GetDailyVisitStatsForLastDays(days)
	if err != nil {
		respondError(c, err)
		return
	}
	respondData(c, stats)
}

// SaveAccessLog retains the former controller middleware behavior for callers
// that need request logging after a handler has run.
func (h *handler) SaveAccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		_ = h.repository.SaveAccessLog(&AccessLog{
			IPAddress:  c.ClientIP(),
			AccessDate: time.Now().Truncate(24 * time.Hour),
			UserAgent:  c.Request.UserAgent(),
			RequestURL: c.Request.URL.String(),
		})
	}
}

func queryInt(c *gin.Context, key string, defaultValue int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return defaultValue
	}
	return value
}

func respondError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, response.Error(err.Error()))
}

func respondSuccess(c *gin.Context) {
	c.JSON(http.StatusOK, response.Success())
}

func respondData(c *gin.Context, data any) {
	c.JSON(http.StatusOK, response.SuccessWithData(data))
}
