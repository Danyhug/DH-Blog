package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"dh-blog/internal/utils"
	"github.com/gin-gonic/gin"
)

// 评论相关错误常量
var (
	ErrCommentNotFound     = errors.New("评论不存在")
	ErrInvalidCommentID    = errors.New("无效的评论 ID")
	ErrInvalidArticleID    = errors.New("无效的文章 ID")
	ErrAddCommentFailed    = errors.New("添加评论失败")
	ErrGetCommentsFailed   = errors.New("获取评论列表失败")
	ErrDeleteCommentFailed = errors.New("删除评论失败")
	ErrUpdateCommentFailed = errors.New("更新评论失败")
	ErrReplyCommentFailed  = errors.New("回复评论失败")
	ErrEmptyCommentID      = errors.New("评论 ID 不能为空")
	ErrParentIDRequired    = errors.New("回复评论需要指定父评论ID")
)

type CommentHandler struct {
	BaseHandler
	repo *repository.CommentRepository
}

func NewCommentHandler(repo *repository.CommentRepository) *CommentHandler {
	return &CommentHandler{repo: repo}
}

// AddComment 添加评论
func (h *CommentHandler) AddComment(c *gin.Context) {
	var comment model.Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	// 获取 User-Agent
	os, browser := utils.ParseUserAgent(c.Request.Header.Get("User-Agent")) // 忽略 os，只获取 browser
	comment.UA = os + "; " + browser
	comment.IsAdmin = false // 默认非管理员评论

	if err := h.repo.AddComment(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrAddCommentFailed.Error(), err)))
		return
	}

	c.JSON(http.StatusCreated, response.Success())
}

// GetCommentsByArticleID 根据文章 ID 获取评论列表（带分页）
func (h *CommentHandler) GetCommentsByArticleID(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("articleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(fmt.Sprintf("%s: %v", ErrInvalidArticleID.Error(), err)))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	comments, total, err := h.repo.GetCommentsByArticleID(int(articleID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrGetCommentsFailed.Error(), err)))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), comments)))
}

// DeleteComment 删除评论
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(fmt.Sprintf("%s: %v", ErrInvalidCommentID.Error(), err)))
		return
	}

	if err := h.repo.DeleteComment(int(id)); err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, response.Error(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrDeleteCommentFailed.Error(), err)))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

// GetAllComments 获取所有评论列表（带分页）
func (h *CommentHandler) GetAllComments(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	comments, total, err := h.repo.GetAllComments(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrGetCommentsFailed.Error(), err)))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), comments)))
}

// UpdateComment 更新评论
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	var comment model.Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	if comment.ID == 0 {
		c.JSON(http.StatusBadRequest, response.Error(ErrEmptyCommentID.Error()))
		return
	}

	if err := h.repo.UpdateComment(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrUpdateCommentFailed.Error(), err)))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

// ReplyComment 回复评论
func (h *CommentHandler) ReplyComment(c *gin.Context) {
	var comment model.Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	// 获取 User-Agent
	os, browser := utils.ParseUserAgent(c.Request.Header.Get("User-Agent")) // 忽略 os，只获取 browser
	comment.UA = os + "; " + browser
	comment.IsAdmin = true // 回复评论通常是管理员行为

	// 确保 ParentID 被设置
	if comment.ParentID == nil || *comment.ParentID == 0 {
		c.JSON(http.StatusBadRequest, response.Error(ErrParentIDRequired.Error()))
		return
	}

	if err := h.repo.AddComment(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrReplyCommentFailed.Error(), err)))
		return
	}

	c.JSON(http.StatusCreated, response.Success())
}

// RegisterRoutes 注册路由
func (h *CommentHandler) RegisterRoutes(router *gin.RouterGroup) {
	// 评论公共 API
	router.POST("/comment", h.AddComment)
	router.GET("/comment/:articleId", h.GetCommentsByArticleID)

	// 评论管理 API
	adminRouter := router.Group("/admin")
	{
		adminRouter.GET("/comment/:pageSize/:pageNum", h.GetAllComments)
		adminRouter.PUT("/comment", h.UpdateComment)
		adminRouter.POST("/comment/reply", h.ReplyComment)
		adminRouter.DELETE("/comment/:id", h.DeleteComment)
	}
}
