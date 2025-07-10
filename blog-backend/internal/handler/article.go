package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"
	"dh-blog/internal/response"
	"dh-blog/internal/service"
	"dh-blog/internal/task"
	"github.com/gin-gonic/gin"
)

// 文章相关错误常量定义
var (
	ErrGetArticleCount  = errors.New("获取文章总数失败")
	ErrGetTagCount      = errors.New("获取标签总数失败")
	ErrGetCategoryCount = errors.New("获取分类总数失败")
	ErrUpdateArticle    = errors.New("更新文章失败")
)

// ArticleHandler 封装文章相关的处理器方法
type ArticleHandler struct {
	BaseHandler
	articleRepo  *repository.ArticleRepository
	tagRepo      *repository.TagRepository
	categoryRepo *repository.CategoryRepository
	aiService    service.AIService
	taskManager  *task.TaskManager
}

// NewArticleHandler 创建文章处理器
func NewArticleHandler(
	articleRepo *repository.ArticleRepository,
	tagRepo *repository.TagRepository,
	categoryRepo *repository.CategoryRepository,
	aiService service.AIService,
	taskManager *task.TaskManager,
) *ArticleHandler {
	return &ArticleHandler{
		articleRepo:  articleRepo,
		tagRepo:      tagRepo,
		categoryRepo: categoryRepo,
		aiService:    aiService,
		taskManager:  taskManager,
	}
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
		h.Error(c, ErrPasswordIncorrect)
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

	// 创建AI生成标签的异步任务
	aiTagTask := task.NewAiGenTask(article.ID, article.Content)
	// 提交到任务队列
	h.taskManager.SubmitTask(aiTagTask)

	// 立即返回响应，不等待AI生成标签
	c.JSON(http.StatusCreated, response.Success())
}

func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	var article model.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(ErrInvalidParams.Error()))
		return
	}

	if err := h.articleRepo.UpdateArticle(&article); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(ErrUpdateArticle.Error()))
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

func (h *ArticleHandler) GetOverview(c *gin.Context) {
	articleCount, err := h.articleRepo.Count(c.Request.Context())
	if err != nil {
		h.Error(c, fmt.Errorf("%w: %v", ErrGetArticleCount, err))
		return
	}
	tagCount, err := h.tagRepo.Count(c.Request.Context())
	if err != nil {
		h.Error(c, fmt.Errorf("%w: %v", ErrGetTagCount, err))
		return
	}
	// Note: Comment count is not available in the new structure
	commentCount := int64(0)
	categoryCount, err := h.categoryRepo.Count(c.Request.Context())
	if err != nil {
		h.Error(c, fmt.Errorf("%w: %v", ErrGetCategoryCount, err))
		return
	}

	type OverviewCount struct {
		ArticleCount  int64 `json:"articleCount"`  // 文章总数
		TagCount      int64 `json:"tagCount"`      // 标签总数
		CommentCount  int64 `json:"commentCount"`  // 评论总数
		CategoryCount int64 `json:"categoryCount"` // 分类总数
	}
	overview := OverviewCount{
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
		return 0, fmt.Errorf("%w: %v", ErrInvalidID, err)
	}
	return id, nil
}

func (h *ArticleHandler) bindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return fmt.Errorf("%w: %v", ErrParamBinding, err)
	}
	return nil
}

func (h *ArticleHandler) getPageRequest(c *gin.Context) (*model.PageRequest, error) {
	var req model.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPageParamBinding, err)
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
	// 根据错误类型设置不同的HTTP状态码
	statusCode := http.StatusInternalServerError

	// 检查特定的错误类型
	switch {
	case errors.Is(err, ErrInvalidID) ||
		errors.Is(err, ErrParamBinding) ||
		errors.Is(err, ErrPageParamBinding):
		statusCode = http.StatusBadRequest
	case errors.Is(err, ErrPasswordIncorrect):
		statusCode = http.StatusForbidden
	default:
		statusCode = http.StatusInternalServerError
	}

	c.JSON(statusCode, gin.H{"error": err.Error()})
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
