package modules

import (
	"dh-blog/internal/controller"
	"dh-blog/internal/router"
)

// ArticleModule 注册文章、标签和分类相关路由。
type ArticleModule struct {
	articleController *controller.ArticleController
}

func NewArticleModule(articleController *controller.ArticleController) *ArticleModule {
	return &ArticleModule{articleController: articleController}
}

func (m *ArticleModule) RegisterRoutes(routes *router.Routes) {
	publicAPI := routes.PublicAPI
	publicAPI.GET("/article/:id", m.articleController.GetArticleDetail)
	publicAPI.GET("/article/title/:id", m.articleController.GetArticleTitle)
	publicAPI.GET("/article/unlock/:id/:password", m.articleController.UnlockArticle)
	publicAPI.POST("/article/list", m.articleController.GetArticleList)
	publicAPI.GET("/article/overview", m.articleController.GetOverview)
	publicAPI.GET("/article/tag", m.articleController.GetAllTags)
	publicAPI.GET("/article/category", m.articleController.GetAllCategories)
	publicAPI.GET("/article/taxonomies", m.articleController.GetAllTaxonomies)
	publicAPI.GET("/article/taxonomy/articles", m.articleController.GetArticlesByTaxonomy)

	adminAPI := routes.AdminAPI
	adminAPI.GET("/article/:id", m.articleController.GetArticleDetail)
	adminAPI.POST("/article", m.articleController.SaveArticle)
	adminAPI.PUT("/article", m.articleController.UpdateArticle)
	adminAPI.POST("/article/list", m.articleController.GetArticleList)
	adminAPI.POST("/article/:id/generate-tags", m.articleController.GenerateTags)

	adminAPI.POST("/tag", m.articleController.CreateTag)
	adminAPI.PUT("/tag", m.articleController.UpdateTag)
	adminAPI.DELETE("/tag/:id", m.articleController.DeleteTag)

	adminAPI.POST("/category", m.articleController.CreateCategory)
	adminAPI.GET("/category/:id", m.articleController.GetCategoryByID)
	adminAPI.PUT("/category", m.articleController.UpdateCategory)
	adminAPI.DELETE("/category/:id", m.articleController.DeleteCategory)
	adminAPI.GET("/category/:id/tags", m.articleController.GetCategoryDefaultTags)
	adminAPI.POST("/category/:id/tags", m.articleController.SaveCategoryDefaultTags)
}
