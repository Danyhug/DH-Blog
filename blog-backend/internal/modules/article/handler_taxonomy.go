package article

import (
	"fmt"
	"net/http"
	"strconv"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetAllTags(c *gin.Context) {
	tags, err := h.tagRepository.FindAll(c.Request.Context())
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, tags)
}

func (h *Handler) CreateTag(c *gin.Context) {
	var tag Tag
	if err := h.bindJSON(c, &tag); err != nil {
		h.Error(c, err)
		return
	}
	if err := h.tagRepository.Create(c.Request.Context(), &tag); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *Handler) UpdateTag(c *gin.Context) {
	var tag Tag
	if err := h.bindJSON(c, &tag); err != nil {
		h.Error(c, err)
		return
	}
	if err := h.tagRepository.Update(c.Request.Context(), &tag); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *Handler) DeleteTag(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}
	if err := h.tagRepository.Delete(c.Request.Context(), id); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}

func (h *Handler) GetCategoryByID(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}
	category, err := h.categoryRepository.FindByID(c.Request.Context(), id)
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, category)
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	var category Category
	if err := h.bindJSON(c, &category); err != nil {
		h.Error(c, err)
		return
	}
	if err := h.categoryRepository.Update(c.Request.Context(), &category); err != nil {
		h.Error(c, err)
		return
	}
	if len(category.TagIDs) > 0 {
		if err := h.categoryRepository.SaveCategoryDefaultTags(category.ID, category.TagIDs); err != nil {
			h.Error(c, fmt.Errorf("更新分类基本信息成功，但更新默认标签失败: %w", err))
			return
		}
	}
	h.Success(c)
}

func (h *Handler) GetCategoryDefaultTags(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}
	tags, err := h.categoryRepository.GetCategoryDefaultTags(id)
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, tags)
}

func (h *Handler) SaveCategoryDefaultTags(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的分类 ID"))
		return
	}
	var req struct {
		TagIDs []int `json:"tag_ids"`
		TagIds []int `json:"tagIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error("无效的请求参数"))
		return
	}
	tagIDs := req.TagIDs
	if len(tagIDs) == 0 {
		tagIDs = req.TagIds
	}
	if err := h.categoryRepository.SaveCategoryDefaultTags(int(id), tagIDs); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error("保存分类默认标签失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success())
}

func (h *Handler) GetAllCategories(c *gin.Context) {
	categories, err := h.categoryRepository.FindAll(c.Request.Context())
	if err != nil {
		h.Error(c, err)
		return
	}
	h.SuccessWithData(c, categories)
}

func (h *Handler) GetAllTaxonomies(c *gin.Context) {
	ctx := c.Request.Context()
	tags, err := h.tagRepository.FindAll(ctx)
	if err != nil {
		h.Error(c, err)
		return
	}
	categories, err := h.categoryRepository.FindAll(ctx)
	if err != nil {
		h.Error(c, err)
		return
	}
	var result []map[string]interface{}
	for _, tag := range tags {
		count, _ := h.articleRepository.CountArticlesByTagName(ctx, tag.Name)
		result = append(result, map[string]interface{}{
			"name": tag.Name, "url": fmt.Sprintf("/tag/%s", tag.Name), "type": "tag", "count": count,
		})
	}
	for _, category := range categories {
		count, _ := h.articleRepository.CountArticlesByCategoryName(ctx, category.Name)
		result = append(result, map[string]interface{}{
			"name": category.Name, "url": fmt.Sprintf("/category/%s", category.Slug), "type": "category", "count": count,
		})
	}
	h.SuccessWithData(c, result)
}

func (h *Handler) GetArticlesByTaxonomy(c *gin.Context) {
	name, taxonomyType := c.Query("name"), c.Query("type")
	if name == "" || taxonomyType == "" {
		c.JSON(http.StatusBadRequest, response.Error("name和type参数不能为空"))
		return
	}
	var (
		articles []*Article
		err      error
	)
	switch taxonomyType {
	case "tag":
		articles, err = h.articleRepository.FindByTagName(c.Request.Context(), name)
	case "category":
		articles, err = h.articleRepository.FindByCategoryName(c.Request.Context(), name)
	default:
		c.JSON(http.StatusBadRequest, response.Error("type参数必须是tag或category"))
		return
	}
	if err != nil {
		h.Error(c, err)
		return
	}
	var result []map[string]interface{}
	for _, article := range articles {
		result = append(result, map[string]interface{}{
			"id": article.ID, "title": article.Title, "views": article.Views, "wordNum": article.WordNum,
			"createTime": article.CreatedAt, "updateTime": article.UpdatedAt,
		})
	}
	h.SuccessWithData(c, result)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var category Category
	if err := h.bindJSON(c, &category); err != nil {
		h.Error(c, err)
		return
	}
	category.ID = 0
	if err := h.categoryRepository.Create(c.Request.Context(), &category); err != nil {
		h.Error(c, err)
		return
	}
	if len(category.TagIDs) > 0 {
		if err := h.categoryRepository.SaveCategoryDefaultTags(category.ID, category.TagIDs); err != nil {
			h.Error(c, fmt.Errorf("创建分类基本信息成功，但保存默认标签失败: %w", err))
			return
		}
	}
	h.Success(c)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id, err := h.getID(c, "id")
	if err != nil {
		h.Error(c, err)
		return
	}
	if err := h.categoryRepository.Delete(c.Request.Context(), id); err != nil {
		h.Error(c, err)
		return
	}
	h.Success(c)
}
