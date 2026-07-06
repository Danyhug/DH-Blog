package modules

import (
	"dh-blog/internal/controller"
	"dh-blog/internal/router"
)

// CommentModule 注册评论相关路由。
type CommentModule struct {
	commentController *controller.CommentController
}

func NewCommentModule(commentController *controller.CommentController) *CommentModule {
	return &CommentModule{commentController: commentController}
}

func (m *CommentModule) RegisterRoutes(routes *router.Routes) {
	publicAPI := routes.PublicAPI
	publicAPI.POST("/comment", m.commentController.AddComment)
	publicAPI.GET("/comment/:articleId", m.commentController.GetCommentsByArticleID)

	adminAPI := routes.AdminAPI
	adminAPI.GET("/comment/:pageSize/:pageNum", m.commentController.GetAllComments)
	adminAPI.PUT("/comment", m.commentController.UpdateComment)
	adminAPI.POST("/comment/reply", m.commentController.ReplyComment)
	adminAPI.DELETE("/comment/:id", m.commentController.DeleteComment)
}
