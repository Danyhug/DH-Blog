package article

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetOverview(c *gin.Context) {
	articleCount, err := h.articleRepository.Count(c.Request.Context())
	if err != nil {
		h.Error(c, fmt.Errorf("%w: %v", ErrGetArticleCount, err))
		return
	}
	tagCount, err := h.tagRepository.Count(c.Request.Context())
	if err != nil {
		h.Error(c, fmt.Errorf("%w: %v", ErrGetTagCount, err))
		return
	}
	commentCount, err := h.commentCounter.Count(c.Request.Context())
	if err != nil {
		h.Error(c, fmt.Errorf("%w: %v", ErrGetCommentCount, err))
		return
	}
	categoryCount, err := h.categoryRepository.Count(c.Request.Context())
	if err != nil {
		h.Error(c, fmt.Errorf("%w: %v", ErrGetCategoryCount, err))
		return
	}
	h.SuccessWithData(c, struct {
		ArticleCount  int64 `json:"articleCount"`
		TagCount      int64 `json:"tagCount"`
		CommentCount  int64 `json:"commentCount"`
		CategoryCount int64 `json:"categoryCount"`
	}{articleCount, tagCount, commentCount, categoryCount})
}

func (h *Handler) GenerateTags(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}
	article, err := h.articleRepository.FindByID(c.Request.Context(), id)
	if err != nil {
		h.Error(c, err)
		return
	}
	if h.tasks != nil {
		h.tasks.SubmitTagGeneration(article.ID, article.Content)
	}
	h.SuccessWithMessage(c, "标签生成任务已提交，稍后将自动更新")
}
