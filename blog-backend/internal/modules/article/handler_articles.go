package article

import (
	"errors"
	"net/http"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetArticleDetail(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}
	article, err := h.articleRepository.GetArticleById(id)
	if err != nil {
		h.Error(c, err)
		return
	}
	isLogin, _ := c.Get("isLogin")
	loggedIn, _ := isLogin.(bool)
	if article.IsLocked && !loggedIn {
		h.Error(c, errors.New("加密文章，请输入密码后访问"))
		return
	}
	go h.articleRepository.UpdateArticleViewCount(id)
	h.SuccessWithData(c, article)
}

func (h *Handler) GetArticleTitle(c *gin.Context) {
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
	h.SuccessWithData(c, article.Title)
}

func (h *Handler) UnlockArticle(c *gin.Context) {
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
	if !article.IsLocked || article.LockPassword != c.Param("password") {
		h.Error(c, ErrPasswordIncorrect)
		return
	}
	h.SuccessWithData(c, article)
}

func (h *Handler) SaveArticle(c *gin.Context) {
	var article Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("参数错误"))
		return
	}
	if err := h.articleRepository.SaveArticle(&article); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error("保存文章失败"))
		return
	}
	if h.tasks != nil {
		h.tasks.SubmitTagGeneration(article.ID, article.Content)
	}
	c.JSON(http.StatusCreated, response.Success())
}

func (h *Handler) UpdateArticle(c *gin.Context) {
	var article Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(ErrInvalidParams.Error()))
		return
	}
	if err := h.articleRepository.UpdateArticle(&article); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(ErrUpdateArticle.Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(article))
}

func (h *Handler) DeleteArticle(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}
	if err := h.articleRepository.Delete(c.Request.Context(), id); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *Handler) GetArticleList(c *gin.Context) {
	pageRequest, err := h.getPageRequest(c)
	if err != nil {
		h.Error(c, err)
		return
	}
	articles, total, err := h.articleRepository.FindPage(c.Request.Context(), pageRequest.PageNum, pageRequest.PageSize)
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithPage(c, articles, total, pageRequest.PageNum)
}
