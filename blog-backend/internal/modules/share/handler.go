package share

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type handler struct {
	service Service
}

func newHandler(service Service) *handler {
	return &handler{service: service}
}

type CreateShareHTTPReq struct {
	FileKey          string `json:"file_key" binding:"required"`
	Password         string `json:"password,omitempty"`
	ExpireDays       *int   `json:"expire_days,omitempty"`
	MaxDownloadCount *int   `json:"max_download_count,omitempty"`
}

type VerifyPasswordRequest struct {
	Password string `json:"password"`
}

// CreateShare creates a file share.
// @Summary 创建文件分享链接
// @Description 为指定文件创建分享链接
// @Tags 分享管理
// @Accept json
// @Produce json
// @Param request body CreateShareHTTPReq true "创建分享请求"
// @Success 200 {object} response.AjaxResult "分享信息"
// @Failure 400 {object} response.AjaxResult "参数错误"
// @Failure 401 {object} response.AjaxResult "未授权"
// @Failure 500 {object} response.AjaxResult "服务器错误"
// @Router /api/files/share [post]
func (h *handler) CreateShare(c *gin.Context) {
	var req CreateShareHTTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	serviceReq := &CreateShareRequest{
		FileKey:          req.FileKey,
		Password:         req.Password,
		MaxDownloadCount: req.MaxDownloadCount,
	}
	if req.ExpireDays != nil && *req.ExpireDays > 0 {
		expireAt := time.Now().AddDate(0, 0, *req.ExpireDays)
		serviceReq.ExpireAt = &expireAt
	}

	share, err := h.service.CreateShare(c.Request.Context(), serviceReq)
	if err != nil {
		logrus.Errorf("创建分享失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(share))
}

// ListShares returns the paginated share list.
// @Summary 获取分享列表
// @Description 分页获取所有分享记录
// @Tags 分享管理
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认10"
// @Success 200 {object} response.AjaxResult "分享列表"
// @Failure 401 {object} response.AjaxResult "未授权"
// @Failure 500 {object} response.AjaxResult "服务器错误"
// @Router /api/files/share [get]
func (h *handler) ListShares(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	shares, total, err := h.service.ListShares(c.Request.Context(), page, pageSize)
	if err != nil {
		logrus.Errorf("获取分享列表失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "获取分享列表失败")
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), shares)))
}

// GetShareDetail returns one managed share.
// @Summary 获取分享详情
// @Description 根据ID获取分享详细信息
// @Tags 分享管理
// @Accept json
// @Produce json
// @Param id path int true "分享ID"
// @Success 200 {object} response.AjaxResult "分享详情"
// @Failure 400 {object} response.AjaxResult "参数错误"
// @Failure 401 {object} response.AjaxResult "未授权"
// @Failure 404 {object} response.AjaxResult "分享不存在"
// @Router /api/files/share/{id} [get]
func (h *handler) GetShareDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "无效的分享ID")
		return
	}
	share, err := h.service.GetShareDetail(c.Request.Context(), id)
	if err != nil {
		response.FailWithCode(c, http.StatusNotFound, "分享不存在")
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(share))
}

// DeleteShare deletes one managed share.
// @Summary 删除分享
// @Description 根据ID删除分享
// @Tags 分享管理
// @Accept json
// @Produce json
// @Param id path int true "分享ID"
// @Success 200 {object} response.AjaxResult "删除成功"
// @Failure 400 {object} response.AjaxResult "参数错误"
// @Failure 401 {object} response.AjaxResult "未授权"
// @Failure 500 {object} response.AjaxResult "服务器错误"
// @Router /api/files/share/{id} [delete]
func (h *handler) DeleteShare(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "无效的分享ID")
		return
	}
	if err := h.service.DeleteShare(c.Request.Context(), id); err != nil {
		logrus.Errorf("删除分享失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "删除分享失败")
		return
	}
	c.JSON(http.StatusOK, response.Success())
}

// GetAccessLogs returns access logs for one managed share.
// @Summary 获取分享访问日志
// @Description 根据分享ID获取访问日志
// @Tags 分享管理
// @Accept json
// @Produce json
// @Param id path int true "分享ID"
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认10"
// @Success 200 {object} response.AjaxResult "访问日志"
// @Failure 400 {object} response.AjaxResult "参数错误"
// @Failure 401 {object} response.AjaxResult "未授权"
// @Failure 500 {object} response.AjaxResult "服务器错误"
// @Router /api/files/share/{id}/logs [get]
func (h *handler) GetAccessLogs(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "无效的分享ID")
		return
	}
	share, err := h.service.GetShareDetail(c.Request.Context(), id)
	if err != nil {
		response.FailWithCode(c, http.StatusNotFound, "分享不存在")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	logs, total, err := h.service.GetShareAccessLogs(c.Request.Context(), share.ShareID, page, pageSize)
	if err != nil {
		logrus.Errorf("获取访问日志失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "获取访问日志失败")
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), logs)))
}

// GetShareInfo returns public share metadata.
// @Summary 获取分享信息
// @Description 根据分享ID获取分享信息（公开访问）
// @Tags 分享访问
// @Accept json
// @Produce json
// @Param shareId path string true "分享短链ID"
// @Success 200 {object} response.AjaxResult "分享信息"
// @Failure 404 {object} response.AjaxResult "分享不存在"
// @Router /api/share/{shareId} [get]
func (h *handler) GetShareInfo(c *gin.Context) {
	shareID := c.Param("shareId")
	if shareID == "" {
		response.FailWithCode(c, http.StatusBadRequest, "分享ID不能为空")
		return
	}

	go func() {
		if err := h.service.RecordAccess(
			c.Request.Context(),
			shareID,
			ShareActionView,
			c.ClientIP(),
			c.Request.UserAgent(),
			c.Request.Referer(),
		); err != nil {
			logrus.Warnf("记录访问日志失败: %v", err)
		}
	}()

	shareInfo, err := h.service.GetShareInfo(c.Request.Context(), shareID)
	if err != nil {
		response.FailWithCode(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(shareInfo))
}

// VerifyPassword verifies a public share password.
// @Summary 验证分享密码
// @Description 验证分享链接的访问密码，成功后返回临时下载令牌
// @Tags 分享访问
// @Accept json
// @Produce json
// @Param shareId path string true "分享短链ID"
// @Param request body VerifyPasswordRequest true "密码验证请求"
// @Success 200 {object} response.AjaxResult "验证结果和下载令牌"
// @Failure 400 {object} response.AjaxResult "参数错误"
// @Failure 401 {object} response.AjaxResult "密码错误"
// @Failure 404 {object} response.AjaxResult "分享不存在"
// @Router /api/share/{shareId}/verify [post]
func (h *handler) VerifyPassword(c *gin.Context) {
	shareID := c.Param("shareId")
	if shareID == "" {
		response.FailWithCode(c, http.StatusBadRequest, "分享ID不能为空")
		return
	}

	var req VerifyPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Password = ""
	}
	result, err := h.service.VerifyPassword(c.Request.Context(), shareID, req.Password)
	if err != nil {
		response.FailWithCode(c, http.StatusNotFound, err.Error())
		return
	}
	if !result.Valid {
		response.FailWithCode(c, http.StatusUnauthorized, "密码错误")
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(result))
}

// Download streams a public shared file.
// @Summary 下载分享文件
// @Description 使用下载令牌下载分享链接中的文件
// @Tags 分享访问
// @Produce octet-stream
// @Param shareId path string true "分享短链ID"
// @Param token query string true "下载令牌"
// @Success 200 {file} file "文件内容"
// @Failure 400 {object} response.AjaxResult "参数错误"
// @Failure 401 {object} response.AjaxResult "令牌无效或已过期"
// @Failure 404 {object} response.AjaxResult "分享不存在"
// @Router /api/share/{shareId}/download [get]
func (h *handler) Download(c *gin.Context) {
	shareID := c.Param("shareId")
	if shareID == "" {
		response.FailWithCode(c, http.StatusBadRequest, "分享ID不能为空")
		return
	}
	token := c.Query("token")
	if token == "" {
		response.FailWithCode(c, http.StatusBadRequest, "下载令牌不能为空")
		return
	}

	preview := c.Query("preview") == "true"
	file, err := h.service.DownloadWithToken(
		c.Request.Context(),
		shareID,
		token,
		c.ClientIP(),
		c.Request.UserAgent(),
		c.Request.Referer(),
		preview,
	)
	if err != nil {
		response.FailWithCode(c, http.StatusUnauthorized, err.Error())
		return
	}

	contentType := file.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	disposition := "attachment"
	if preview {
		disposition = "inline"
	}
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, file.Name))
	c.Header("Content-Type", contentType)
	c.File(file.StoragePath)
}
