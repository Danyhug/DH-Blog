package model

// Tag 对应于数据库中的 `tags` 表
type Tag struct {
	BaseModel `gorm:"embedded"`
	Name      string     `gorm:"column:name;not null;uniqueIndex" json:"name"`      // 标签名，唯一
	Slug      string     `gorm:"column:slug;not null;uniqueIndex" json:"slug"`      // URL 友好的别名，唯一
	Articles  []*Article `gorm:"many2many:article_tags;" json:"articles,omitempty"` // 关联的文章
}

// TableName 指定 GORM 使用的表名
func (Tag) TableName() string {
	return "tags"
}
