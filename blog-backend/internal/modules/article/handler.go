package article

import (
	"errors"
)

var (
	ErrGetArticleCount   = errors.New("获取文章总数失败")
	ErrGetTagCount       = errors.New("获取标签总数失败")
	ErrGetCategoryCount  = errors.New("获取分类总数失败")
	ErrGetCommentCount   = errors.New("获取评论总数失败")
	ErrUpdateArticle     = errors.New("更新文章失败")
	ErrInvalidID         = errors.New("无效的ID")
	ErrInvalidParams     = errors.New("无效的请求参数")
	ErrParamBinding      = errors.New("请求参数绑定失败")
	ErrPageParamBinding  = errors.New("分页参数绑定失败")
	ErrPasswordIncorrect = errors.New("密码错误")
)

type Handler struct {
	articleRepository  *ArticleRepository
	tagRepository      *TagRepository
	categoryRepository *CategoryRepository
	commentCounter     CommentCounter
	ai                 AIService
	tasks              TagTaskScheduler
}

func NewHandler(
	articleRepository *ArticleRepository,
	tagRepository *TagRepository,
	categoryRepository *CategoryRepository,
	commentCounter CommentCounter,
	ai AIService,
	tasks TagTaskScheduler,
) *Handler {
	return &Handler{
		articleRepository:  articleRepository,
		tagRepository:      tagRepository,
		categoryRepository: categoryRepository,
		commentCounter:     commentCounter,
		ai:                 ai,
		tasks:              tasks,
	}
}
