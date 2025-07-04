package handler

import (
	"errors"
	"net/http"
	"strconv"

	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"github.com/gin-gonic/gin"
)

type ArticleHandler struct {
	articleRepo    *repository.ArticleRepository
	tagRepo        *repository.TagRepository
	categoryRepo   *repository.CategoryRepository
	dailyStatsRepo *repository.DailyStatsRepository
}

func NewArticleHandler(articleRepo *repository.ArticleRepository, tagRepo *repository.TagRepository, categoryRepo *repository.CategoryRepository, dailyStatsRepo *repository.DailyStatsRepository) *ArticleHandler {
	return &ArticleHandler{
		articleRepo:    articleRepo,
		tagRepo:        tagRepo,
		categoryRepo:   categoryRepo,
		dailyStatsRepo: dailyStatsRepo,
	}
}

func (h *ArticleHandler) GetAllTags(c *gin.Context) {
	tags, err := h.tagRepo.GetAllTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取所有标签失败", err).Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(tags))
}

func (h *ArticleHandler) CreateTag(c *gin.Context) {
	var tag model.Tag
	if err := c.ShouldBindJSON(&tag); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	if err := h.tagRepo.CreateTag(&tag); err != nil {
		// 检查是否是标签已存在的错误
		if errors.Is(err, errs.ErrTagAlreadyExists) {
			c.JSON(http.StatusConflict, response.Error(errs.ErrTagAlreadyExists.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("创建标签失败", err).Error()))
		return
	}

	c.JSON(http.StatusCreated, response.SuccessWithData(tag))
}

func (h *ArticleHandler) UpdateTag(c *gin.Context) {
	var tag model.Tag
	if err := c.ShouldBindJSON(&tag); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	if tag.ID == 0 {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("标签 ID 不能为空", nil).Error()))
		return
	}

	if err := h.tagRepo.UpdateTag(&tag); err != nil {
		// 检查是否是标签已存在的错误
		if errors.Is(err, errs.ErrTagAlreadyExists) {
			c.JSON(http.StatusConflict, response.Error(errs.ErrTagAlreadyExists.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("更新标签失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(tag))
}

func (h *ArticleHandler) DeleteTag(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的标签 ID", err).Error()))
		return
	}

	if err := h.tagRepo.DeleteTag(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("删除标签失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

func (h *ArticleHandler) GetCategoryByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的分类 ID", err).Error()))
		return
	}

	category, err := h.categoryRepo.GetCategoryByID(uint(id))
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			c.JSON(http.StatusNotFound, response.Error(errs.NotFound(err.Error(), nil).Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取分类失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(category))
}

func (h *ArticleHandler) UpdateCategory(c *gin.Context) {
	var category model.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	if category.ID == 0 {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("分类 ID 不能为空", nil).Error()))
		return
	}

	if err := h.categoryRepo.UpdateCategory(&category); err != nil {
		// 检查是否是分类已存在的错误
		if errors.Is(err, errs.ErrCategoryAlreadyExists) {
			c.JSON(http.StatusConflict, response.Error(errs.ErrCategoryAlreadyExists.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("更新分类失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(category))
}

// GetArticleTitle 获取文章标题
func (h *ArticleHandler) GetCategoryDefaultTags(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的分类 ID", err).Error()))
		return
	}

	tagIDs, err := h.categoryRepo.GetCategoryDefaultTagsByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取分类默认标签失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{"tag_ids": tagIDs}))
}

// UnlockArticle 获取需要解密的文章
func (h *ArticleHandler) GetOverview(c *gin.Context) {
	articleCount, err := h.dailyStatsRepo.CountArticles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取文章总数失败", err).Error()))
		return
	}
	tagCount, err := h.dailyStatsRepo.CountTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取标签总数失败", err).Error()))
		return
	}
	commentCount, err := h.dailyStatsRepo.CountComments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取评论总数失败", err).Error()))
		return
	}
	categoryCount, err := h.dailyStatsRepo.CountCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取分类总数失败", err).Error()))
		return
	}

	overview := model.OverviewCount{
		ArticleCount:  articleCount,
		TagCount:      tagCount,
		CommentCount:  commentCount,
		CategoryCount: categoryCount,
	}

	c.JSON(http.StatusOK, response.SuccessWithData(overview))
}

// SaveArticle 创建新文章
func (h *ArticleHandler) SaveCategoryDefaultTags(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的分类 ID", err).Error()))
		return
	}

	var req struct {
		TagIDs []uint `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	if err := h.categoryRepo.SaveCategoryDefaultTags(uint(id), req.TagIDs); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("保存分类默认标签失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

// UpdateArticle 更新文章
func (h *ArticleHandler) GetAllCategories(c *gin.Context) {
	categories, err := h.categoryRepo.GetAllCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取所有分类失败", err).Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(categories))
}

func (h *ArticleHandler) CreateCategory(c *gin.Context) {
	var category model.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	if err := h.categoryRepo.CreateCategory(&category); err != nil {
		// 检查是否是分类已存在的错误
		if errors.Is(err, errs.ErrCategoryAlreadyExists) {
			c.JSON(http.StatusConflict, response.Error(errs.ErrCategoryAlreadyExists.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("创建分类失败", err).Error()))
		return
	}

	c.JSON(http.StatusCreated, response.SuccessWithData(category))
}

// DeleteArticle 删除文章
func (h *ArticleHandler) DeleteCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的分类 ID", err).Error()))
		return
	}

	if err := h.categoryRepo.DeleteCategory(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("删除分类失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

// GetArticleList 获取文章列表（带分页）
func (h *ArticleHandler) GetArticleDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的文章 ID", err).Error()))
		return
	}
	article, err := h.articleRepo.GetArticleById(id)
	if err != nil {
		if errors.Is(err, errs.ErrArticleNotFound) {
			c.JSON(http.StatusNotFound, response.Error(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取文章详情失败", err).Error()))
		return
	}

	// 更新观看次数
	go h.articleRepo.AddViewCount(id)

	article.Views += 1
	c.JSON(http.StatusOK, response.SuccessWithData(article))
}

func (h *ArticleHandler) GetArticleTitle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的文章 ID", err).Error()))
		return
	}

	title, err := h.articleRepo.GetArticleTitleByID(id)
	if err != nil {
		if errors.Is(err, errs.ErrArticleNotFound) {
			c.JSON(http.StatusNotFound, response.Error(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取文章标题失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(title))
}

func (h *ArticleHandler) UnlockArticle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的文章 ID", err).Error()))
		return
	}
	password := c.Param("password")

	article, err := h.articleRepo.GetLockedArticle(id, password)
	if err != nil {
		if errors.Is(err, errs.ErrArticleNotFound) {
			c.JSON(http.StatusNotFound, response.Error(errors.New("文章不存在或密码错误").Error()))
			return
		}
		c.JSON(http.StatusUnauthorized, response.Error(errs.Unauthorized("密码错误或文章不存在", nil).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(article))
}

func (h *ArticleHandler) SaveArticle(c *gin.Context) {
	var article model.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	if err := h.articleRepo.SaveArticle(&article); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("保存文章失败", err).Error()))
		return
	}

	c.JSON(http.StatusCreated, response.SuccessWithData(article))
}

func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	var article model.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的请求参数", err).Error()))
		return
	}

	if err := h.articleRepo.UpdateArticle(&article); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("更新文章失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(article))
}

func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(errs.BadRequest("无效的文章 ID", err).Error()))
		return
	}

	if err := h.articleRepo.DeleteArticle(id); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("删除文章失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

func (h *ArticleHandler) GetArticleList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	articles, total, err := h.articleRepo.GetArticleList(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(errs.InternalServerError("获取文章列表失败", err).Error()))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), articles)))
}
