package model

// OverviewCount 网站总览统计数据
type OverviewCount struct {
	ArticleCount  int64 `json:"articleCount"`  // 文章总数
	TagCount      int64 `json:"tagCount"`      // 标签总数
	CommentCount  int64 `json:"commentCount"`  // 评论总数
	CategoryCount int64 `json:"categoryCount"` // 分类总数
}
