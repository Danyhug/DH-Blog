package comment

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"dh-blog/internal/response"
	"dh-blog/internal/utils"

	"github.com/gin-gonic/gin"
)

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
	ErrInvalidPagination   = errors.New("无效的分页参数")
)

type handler struct {
	repo *Repository
}

func newHandler(repo *Repository) *handler {
	return &handler{repo: repo}
}

func (h *handler) AddComment(c *gin.Context) {
	var comment Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	os, browser := utils.ParseUserAgent(c.Request.Header.Get("User-Agent"))
	comment.UA = os + "; " + browser
	comment.IsAdmin = false
	if err := h.repo.AddComment(&comment); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrAddCommentFailed.Error(), err)))
		return
	}
	c.JSON(http.StatusCreated, response.Success())
}

func (h *handler) GetCommentsByArticleID(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("articleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(fmt.Sprintf("%s: %v", ErrInvalidArticleID.Error(), err)))
		return
	}

	comments, total, err := h.repo.GetCommentsByArticleID(int(articleID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrGetCommentsFailed.Error(), err)))
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(10), comments)))
}

func (h *handler) DeleteComment(c *gin.Context) {
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

func (h *handler) GetAllComments(c *gin.Context) {
	pageSize, pageSizeErr := strconv.Atoi(c.Param("pageSize"))
	page, pageErr := strconv.Atoi(c.Param("pageNum"))
	if pageSizeErr != nil || pageErr != nil || pageSize <= 0 || page <= 0 {
		c.JSON(http.StatusBadRequest, response.Error(ErrInvalidPagination.Error()))
		return
	}

	groups, total, err := h.repo.GetCommentGroups(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(fmt.Sprintf("%s: %v", ErrGetCommentsFailed.Error(), err)))
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), groups)))
}

func (h *handler) UpdateComment(c *gin.Context) {
	var comment Comment
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

func (h *handler) ReplyComment(c *gin.Context) {
	var comment Comment
	if err := c.ShouldBindJSON(&comment); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数: "+err.Error()))
		return
	}

	os, browser := utils.ParseUserAgent(c.Request.Header.Get("User-Agent"))
	comment.UA = os + "; " + browser
	comment.IsAdmin = true
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
