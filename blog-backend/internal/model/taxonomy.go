package model

// Tag 对应于数据库中的 `tags` 表
type Tag struct {
	BaseModel `gorm:"embedded"`
	Name      string     `gorm:"column:name;not null;uniqueIndex" json:"name"`      // 标签名，唯一
	Articles  []*Article `gorm:"many2many:article_tags;" json:"articles,omitempty"` // 关联的文章
}

// TableName 指定 GORM 使用的表名
func (Tag) TableName() string {
	return "tags"
}

// TagRelation 是标签和其他模型（如文章、分类）的通用关联表
type TagRelation struct {
	TagID        int    `gorm:"primaryKey"`
	RelatedID    int    `gorm:"primaryKey"`
	RelationType string `gorm:"primaryKey"` // e.g., "article", "category"
}

// TableName 指定 GORM 使用的表名
func (TagRelation) TableName() string {
	return "tag_relations"
}

// Category 对应于数据库中的 `categories` 表
type Category struct {
	BaseModel `gorm:"embedded"`
	Name      string `gorm:"column:name;not null;uniqueIndex" json:"name"` // 分类名称，唯一
	Slug      string `gorm:"column:slug;not null;uniqueIndex" json:"slug"` // URL 友好的别名，唯一
	TagIDs    []int  `gorm:"-" json:"tagIds"`                              // 关联的标签ID列表，仅用于数据传输
}

// TableName 指定 GORM 使用的表名
func (Category) TableName() string {
	return "categories"
}
