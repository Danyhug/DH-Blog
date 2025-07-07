package handler

import (
	"errors"
	"net/http"
	"strconv"

	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"dh-blog/internal/utils"
	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	repo *repository.CommentRepository
}

func NewCommentHandler(repo *repository.CommentRepository) *CommentHandler {
	return &CommentHandler{repo: repo}
}

// AddComment 添加评论
func (h *CommentHandler) AddComment(c *gin.Context) {
	var comment model.Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	// 获取 User-Agent
	os, browser := utils.ParseUserAgent(c.Request.Header.Get("User-Agent")) // 忽略 os，只获取 browser
	comment.UA = os + "; " + browser
	comment.IsAdmin = false // 默认非管理员评论

	if err := h.repo.AddComment(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("添加评论失败", err).Error()))
		return
	}

	c.JSON(http.StatusCreated, response.Success())
}

// GetCommentsByArticleID 根据文章 ID 获取评论列表（带分页）
func (h *CommentHandler) GetCommentsByArticleID(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("articleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的文章 ID", err).Error()))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	comments, total, err := h.repo.GetCommentsByArticleID(int(articleID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取评论列表失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), comments)))
}

// DeleteComment 删除评论
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的评论 ID", err).Error()))
		return
	}

	if err := h.repo.DeleteComment(int(id)); err != nil {
		if errors.Is(err, errs.ErrCommentNotFound) {
			c.JSON(http.StatusNotFound, response.Error(errs.NotFound(err.Error(), nil).Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("删除评论失败", err).Error()))
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
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取评论列表失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), comments)))
}

// UpdateComment 更新评论
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	var comment model.Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	if comment.ID == 0 {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("评论 ID 不能为空", nil).Error()))
		return
	}

	if err := h.repo.UpdateComment(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("更新评论失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

// ReplyComment 回复评论
func (h *CommentHandler) ReplyComment(c *gin.Context) {
	var comment model.Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	// 获取 User-Agent
	os, browser := utils.ParseUserAgent(c.Request.Header.Get("User-Agent")) // 忽略 os，只获取 browser
	comment.UA = os + "; " + browser
	comment.IsAdmin = true // 回复评论通常是管理员行为

	// 确保 ParentID 被设置
	if comment.ParentID == nil || *comment.ParentID == 0 {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("ParentID is required for reply", nil).Error()))
		return
	}

	if err := h.repo.AddComment(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("回复评论失败", err).Error()))
		return
	}

	c.JSON(http.StatusCreated, response.Success())
}
