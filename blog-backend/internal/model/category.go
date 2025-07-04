package model

// Category 对应于数据库中的 `categories` 表
type Category struct {
	BaseModel `gorm:"embedded"`
	Name      string `gorm:"column:name;not null;uniqueIndex" json:"name"` // 分类名称，唯一
	Slug      string `gorm:"column:slug;not null;uniqueIndex" json:"slug"` // URL 友好的别名，唯一
	TagIDs    []uint `gorm:"-" json:"tagIds"`                              // 关联的标签ID列表，仅用于数据传输
}

// TableName 指定 GORM 使用的表名
func (Category) TableName() string {
	return "categories"
}
