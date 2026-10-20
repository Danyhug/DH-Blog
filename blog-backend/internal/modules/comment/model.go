package comment

import "dh-blog/internal/model"

// Comment 对应数据库中的 comments 表。
type Comment struct {
	model.BaseModel `gorm:"embedded"`
	ArticleID       int        `gorm:"column:article_id;not null" json:"articleId"`
	Author          string     `gorm:"column:author;not null" json:"author"`
	Email           string     `gorm:"column:email;not null" json:"email"`
	Content         string     `gorm:"column:content;not null" json:"content"`
	IsPublic        bool       `gorm:"column:is_public;default:true" json:"isPublic"`
	ParentID        *int       `gorm:"column:parent_id" json:"parentId"`
	UA              string     `gorm:"column:ua;not null" json:"ua"`
	IsAdmin         bool       `gorm:"column:is_admin;default:false" json:"isAdmin"`
	Children        []*Comment `gorm:"-" json:"children,omitempty"`
}

// ArticleCommentGroup 是后台评论管理使用的文章评论分组。
type ArticleCommentGroup struct {
	ArticleID         int            `json:"articleId"`
	ArticleTitle      string         `json:"articleTitle"`
	CommentCount      int64          `json:"commentCount"`
	LatestCommentTime model.JSONTime `json:"latestCommentTime"`
	Children          []*Comment     `json:"children"`
}

// CommentPage 是文章详情页的评论分页结果。
//
// 分页切的是**顶层评论**，因为一条对话连同它的回复必须留在同一页，切成两段读者接不上。
// 所以这里有两个总数：
//   - Total：这篇文章的全部公开评论数（父子都算），给「N 条评论」用；
//   - RootTotal：顶层评论数，给分页器算页数用。
type CommentPage struct {
	Total     int64      `json:"total"`
	RootTotal int64      `json:"rootTotal"`
	List      []*Comment `json:"list"`
}

// TableName 保持原有数据库表名。
func (Comment) TableName() string {
	return "comments"
}
