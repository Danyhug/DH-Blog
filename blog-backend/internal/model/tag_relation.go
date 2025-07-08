package model

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
