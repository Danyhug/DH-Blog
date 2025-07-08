package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"dh-blog/internal/errs"
	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"github.com/gin-gonic/gin"
)

// ArticleHandler 封装文章相关的处理器方法
type ArticleHandler struct {
	BaseHandler
	articleRepo  *repository.ArticleRepository
	tagRepo      *repository.TagRepository
	categoryRepo *repository.CategoryRepository
}

// NewArticleHandler 创建文章处理器
func NewArticleHandler(articleRepo *repository.ArticleRepository, tagRepo *repository.TagRepository, categoryRepo *repository.CategoryRepository) *ArticleHandler {
	return &ArticleHandler{articleRepo: articleRepo, tagRepo: tagRepo, categoryRepo: categoryRepo}
}

func (h *ArticleHandler) GetAllTags(c *gin.Context) {
	tags, err := h.tagRepo.FindAll(c.Request.Context())
	if err != nil {
		h.Error(c, err)
		return
	}

	h.SuccessWithData(c, tags)
}

func (h *ArticleHandler) CreateTag(c *gin.Context) {
	var tag model.Tag
	if err := h.bindJSON(c, &tag); err != nil {
		h.Error(c, err)
		return
	}

	if err := h.tagRepo.Create(c.Request.Context(), &tag); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *ArticleHandler) UpdateTag(c *gin.Context) {
	var tag model.Tag
	if err := h.bindJSON(c, &tag); err != nil {
		h.Error(c, err)
		return
	}

	if err := h.tagRepo.Update(c.Request.Context(), &tag); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *ArticleHandler) DeleteTag(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	if err := h.tagRepo.Delete(c.Request.Context(), id); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *ArticleHandler) GetCategoryByID(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	category, err := h.categoryRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, category)
}

func (h *ArticleHandler) UpdateCategory(c *gin.Context) {
	var category model.Category
	if err := h.bindJSON(c, &category); err != nil {
		h.Error(c, err)
		return
	}

	// 先更新分类基本信息
	if err := h.categoryRepo.Update(c.Request.Context(), &category); err != nil {
		h.Error(c, err)
		return
	}

	// 如果提供了标签ID，则更新分类的默认标签
	if len(category.TagIDs) > 0 {
		if err := h.categoryRepo.SaveCategoryDefaultTags(category.ID, category.TagIDs); err != nil {
			h.Error(c, fmt.Errorf("更新分类基本信息成功，但更新默认标签失败: %w", err))
			return
		}
	}

	h.Success(c)
}

func (h *ArticleHandler) GetCategoryDefaultTags(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	tags, err := h.categoryRepo.GetCategoryDefaultTags(id)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.SuccessWithData(c, tags)
}

func (h *ArticleHandler) SaveCategoryDefaultTags(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的分类 ID"))
		return
	}

	// 支持多种格式的请求体
	var req struct {
		TagIDs []int `json:"tag_ids"`
		TagIds []int `json:"tagIds"` // 兼容驼峰命名
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数"))
		return
	}

	// 使用非空的标签ID列表
	var tagIDs []int
	if len(req.TagIDs) > 0 {
		tagIDs = req.TagIDs
	} else {
		tagIDs = req.TagIds
	}

	if err := h.categoryRepo.SaveCategoryDefaultTags(int(id), tagIDs); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error("保存分类默认标签失败"))
		return
	}

	c.JSON(http.StatusOK, response.Success())
}

func (h *ArticleHandler) GetAllCategories(c *gin.Context) {
	categories, err := h.categoryRepo.FindAll(c.Request.Context())
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, categories)
}

func (h *ArticleHandler) CreateCategory(c *gin.Context) {
	var category model.Category
	if err := h.bindJSON(c, &category); err != nil {
		h.Error(c, err)
		return
	}

	// 先创建分类基本信息
	if err := h.categoryRepo.Create(c.Request.Context(), &category); err != nil {
		h.Error(c, err)
		return
	}

	// 如果提供了标签ID，则保存分类的默认标签
	if len(category.TagIDs) > 0 {
		if err := h.categoryRepo.SaveCategoryDefaultTags(category.ID, category.TagIDs); err != nil {
			h.Error(c, fmt.Errorf("创建分类基本信息成功，但保存默认标签失败: %w", err))
			return
		}
	}

	h.Success(c)
}

func (h *ArticleHandler) DeleteCategory(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	if err := h.categoryRepo.Delete(c.Request.Context(), id); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *ArticleHandler) GetArticleDetail(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	// 异步增加文章浏览次数
	go h.articleRepo.UpdateArticleViewCount(id)

	// 使用预加载获取文章及其标签
	article, err := h.articleRepo.GetArticleById(id)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.SuccessWithData(c, article)
}

func (h *ArticleHandler) GetArticleTitle(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	article, err := h.articleRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, article.Title)
}

func (h *ArticleHandler) UnlockArticle(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}
	password := c.Param("password")

	article, err := h.articleRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		h.Error(c, err)
		return
	}

	if !article.IsLocked || article.LockPassword != password {
		h.Error(c, errs.Forbidden("密码错误", nil))
		return
	}

	h.SuccessWithData(c, article)
}

func (h *ArticleHandler) SaveArticle(c *gin.Context) {
	var article model.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("参数错误"))
		return
	}

	err := h.articleRepo.SaveArticle(&article)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error("保存文章失败"))
		return
	}
	c.JSON(http.StatusCreated, response.Success())
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
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}

	if err := h.articleRepo.Delete(c.Request.Context(), id); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *ArticleHandler) GetArticleList(c *gin.Context) {
	pageReq, err := h.getPageRequest(c)
	if err != nil {
		h.Error(c, err)
		return
	}

	articles, total, err := h.articleRepo.FindPage(c.Request.Context(), pageReq.Page, pageReq.PageSize)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.SuccessWithPage(c, articles, total, pageReq.Page)
}

// RegisterRoutes 注册路由
func (h *ArticleHandler) RegisterRoutes(router *gin.RouterGroup) {
	// 文章公共 API
	router.GET("/article/:id", h.GetArticleDetail)
	router.GET("/article/title/:id", h.GetArticleTitle)
	router.GET("/article/unlock/:id/:password", h.UnlockArticle)
	router.POST("/article/list", h.GetArticleList)
	router.GET("/article/overview", h.GetOverview)
	router.GET("/article/tag", h.GetAllTags)
	router.GET("/article/category", h.GetAllCategories)

	// 管理员 API
	adminRouter := router.Group("/admin")
	{
		adminRouter.GET("/article/:id", h.GetArticleDetail)
		adminRouter.POST("/article", h.SaveArticle)
		adminRouter.PUT("/article", h.UpdateArticle)
		adminRouter.POST("/article/list", h.GetArticleList)

		// 标签管理
		adminRouter.POST("/tag", h.CreateTag)
		adminRouter.PUT("/tag", h.UpdateTag)
		adminRouter.DELETE("/tag/:id", h.DeleteTag)

		// 分类管理
		adminRouter.POST("/category", h.CreateCategory)
		adminRouter.GET("/category/:id", h.GetCategoryByID)
		adminRouter.PUT("/category", h.UpdateCategory)
		adminRouter.DELETE("/category/:id", h.DeleteCategory)
		adminRouter.GET("/category/:id/tags", h.GetCategoryDefaultTags)
		adminRouter.POST("/category/:id/tags", h.SaveCategoryDefaultTags)
	}
}

func (h *ArticleHandler) GetOverview(c *gin.Context) {
	articleCount, err := h.articleRepo.Count(c.Request.Context())
	if err != nil {
		h.Error(c, errs.InternalServerError("获取文章总数失败", err))
		return
	}
	tagCount, err := h.tagRepo.Count(c.Request.Context())
	if err != nil {
		h.Error(c, errs.InternalServerError("获取标签总数失败", err))
		return
	}
	// Note: Comment count is not available in the new structure
	commentCount := int64(0)
	categoryCount, err := h.categoryRepo.Count(c.Request.Context())
	if err != nil {
		h.Error(c, errs.InternalServerError("获取分类总数失败", err))
		return
	}

	overview := model.OverviewCount{
		ArticleCount:  articleCount,
		TagCount:      tagCount,
		CommentCount:  commentCount,
		CategoryCount: categoryCount,
	}

	h.SuccessWithData(c, overview)
}

func (h *ArticleHandler) getID(c *gin.Context, key string) (int, error) {
	idStr := c.Param(key)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, errs.BadRequest("无效的ID", err)
	}
	return id, nil
}

func (h *ArticleHandler) bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return errs.BadRequest("请求参数绑定失败", err)
	}
	return nil
}

func (h *ArticleHandler) getPageRequest(c *gin.Context) (*model.PageRequest, error) {
	var req model.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		return nil, errs.BadRequest("分页参数绑定失败", err)
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}
	return &req, nil
}

func (h *ArticleHandler) Error(c *gin.Context, err error) {
	var appErr *errs.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
}

func (h *ArticleHandler) Success(c *gin.Context) {
	c.JSON(http.StatusOK, response.Success())
}

func (h *ArticleHandler) SuccessWithData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response.SuccessWithData(data))
}

func (h *ArticleHandler) SuccessWithPage(c *gin.Context, data interface{}, total int64, page int) {
	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), data)))
}
