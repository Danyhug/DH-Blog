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

// TableName 保持原有数据库表名。
func (Comment) TableName() string {
	return "comments"
}
