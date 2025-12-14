package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"dh-blog/internal/response"
	"dh-blog/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ShareHandler 分享处理器
type ShareHandler struct {
	shareService service.IShareService
}

// NewShareHandler 创建分享处理器
func NewShareHandler(shareService service.IShareService) *ShareHandler {
	return &ShareHandler{
		shareService: shareService,
	}
}

// CreateShareRequest 创建分享请求
type CreateShareRequest struct {
	FileKey          string `json:"file_key" binding:"required"`  // 文件ID
	Password         string `json:"password,omitempty"`           // 访问密码（可选）
	ExpireDays       *int   `json:"expire_days,omitempty"`        // 过期天数（可选）
	MaxDownloadCount *int   `json:"max_download_count,omitempty"` // 最大下载次数（可选）
}

// VerifyPasswordRequest 验证密码请求
type VerifyPasswordRequest struct {
	Password string `json:"password"`
}

// ========== 管理接口（需要JWT认证） ==========

// CreateShare 创建分享
// @Summary 创建文件分享链接
// @Description 为指定文件创建分享链接
// @Tags 分享管理
// @Accept json
// @Produce json
// @Param request body CreateShareRequest true "创建分享请求"
// @Success 200 {object} response.AjaxResult "分享信息"
// @Failure 400 {object} response.AjaxResult "参数错误"
// @Failure 401 {object} response.AjaxResult "未授权"
// @Failure 500 {object} response.AjaxResult "服务器错误"
// @Router /api/files/share [post]
func (h *ShareHandler) CreateShare(c *gin.Context) {
	var req CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 构建服务层请求
	serviceReq := &service.CreateShareRequest{
		FileKey:          req.FileKey,
		Password:         req.Password,
		MaxDownloadCount: req.MaxDownloadCount,
	}

	// 处理过期时间
	if req.ExpireDays != nil && *req.ExpireDays > 0 {
		expireAt := time.Now().AddDate(0, 0, *req.ExpireDays)
		serviceReq.ExpireAt = &expireAt
	}

	share, err := h.shareService.CreateShare(c.Request.Context(), serviceReq)
	if err != nil {
		logrus.Errorf("创建分享失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(share))
}

// ListShares 获取分享列表
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
func (h *ShareHandler) ListShares(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	shares, total, err := h.shareService.ListShares(c.Request.Context(), page, pageSize)
	if err != nil {
		logrus.Errorf("获取分享列表失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "获取分享列表失败")
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), shares)))
}

// GetShareDetail 获取分享详情（管理）
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
func (h *ShareHandler) GetShareDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "无效的分享ID")
		return
	}

	share, err := h.shareService.GetShareDetail(c.Request.Context(), id)
	if err != nil {
		response.FailWithCode(c, http.StatusNotFound, "分享不存在")
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(share))
}

// DeleteShare 删除分享
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
func (h *ShareHandler) DeleteShare(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "无效的分享ID")
		return
	}

	if err := h.shareService.DeleteShare(c.Request.Context(), id); err != nil {
		logrus.Errorf("删除分享失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "删除分享失败")
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

// GetAccessLogs 获取分享访问日志
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
func (h *ShareHandler) GetAccessLogs(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "无效的分享ID")
		return
	}

	// 先获取分享详情以获得shareID
	share, err := h.shareService.GetShareDetail(c.Request.Context(), id)
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

	logs, total, err := h.shareService.GetShareAccessLogs(c.Request.Context(), share.ShareID, page, pageSize)
	if err != nil {
		logrus.Errorf("获取访问日志失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "获取访问日志失败")
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), logs)))
}

// ========== 公开接口（无需认证） ==========

// GetShareInfo 获取分享信息（公开）
// @Summary 获取分享信息
// @Description 根据分享ID获取分享信息（公开访问）
// @Tags 分享访问
// @Accept json
// @Produce json
// @Param shareId path string true "分享短链ID"
// @Success 200 {object} response.AjaxResult "分享信息"
// @Failure 404 {object} response.AjaxResult "分享不存在"
// @Router /api/share/{shareId} [get]
func (h *ShareHandler) GetShareInfo(c *gin.Context) {
	shareID := c.Param("shareId")
	if shareID == "" {
		response.FailWithCode(c, http.StatusBadRequest, "分享ID不能为空")
		return
	}

	// 记录访问日志
	go func() {
		if err := h.shareService.RecordAccess(
			c.Request.Context(),
			shareID,
			"view",
			c.ClientIP(),
			c.Request.UserAgent(),
			c.Request.Referer(),
		); err != nil {
			logrus.Warnf("记录访问日志失败: %v", err)
		}
	}()

	shareInfo, err := h.shareService.GetShareInfo(c.Request.Context(), shareID)
	if err != nil {
		response.FailWithCode(c, http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(shareInfo))
}

// VerifyPassword 验证分享密码并获取下载令牌
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
func (h *ShareHandler) VerifyPassword(c *gin.Context) {
	shareID := c.Param("shareId")
	if shareID == "" {
		response.FailWithCode(c, http.StatusBadRequest, "分享ID不能为空")
		return
	}

	var req VerifyPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空密码（无密码分享的情况）
		req.Password = ""
	}

	result, err := h.shareService.VerifyPassword(c.Request.Context(), shareID, req.Password)
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

// Download 下载分享文件
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
func (h *ShareHandler) Download(c *gin.Context) {
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

	// 使用令牌下载
	file, err := h.shareService.DownloadWithToken(
		c.Request.Context(),
		shareID,
		token,
		c.ClientIP(),
		c.Request.UserAgent(),
		c.Request.Referer(),
	)
	if err != nil {
		response.FailWithCode(c, http.StatusUnauthorized, err.Error())
		return
	}

	// 设置响应头
	fileName := file.Name
	contentType := file.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", contentType)
	c.File(file.StoragePath)
}
