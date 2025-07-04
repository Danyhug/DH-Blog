package model

// Article 对应于数据库中的 `articles` 表
type Article struct {
	BaseModel    `gorm:"embedded"`
	Title        string `gorm:"column:title;not null" json:"title"`             // 文章标题
	Content      string `gorm:"column:content;not null" json:"content"`         // 文章内容
	CategoryID   int    `gorm:"column:category_id" json:"categoryId"`           // 分类ID，关联到 categories 表
	Views        int    `gorm:"column:views;default:0" json:"views"`            // 浏览次数
	WordNum      int    `gorm:"column:word_num" json:"wordNum"`                 // 文章字数
	ThumbnailURL string `gorm:"column:thumbnail_url" json:"thumbnailUrl"`       // 缩略图 URL
	IsLocked     bool   `gorm:"column:is_locked;default:false" json:"isLocked"` // 是否锁定
	LockPassword string `gorm:"column:lock_password" json:"lockPassword"`       // 锁定密码

	Tags     []*Tag   `gorm:"many2many:article_tags;" json:"-"` // 文章关联的标签
	TagNames []string `gorm:"-" json:"tags,omitempty"`          // 接收前端传来的标签名称
}

// TableName 指定 GORM 使用的表名
func (*Article) TableName() string {
	return "articles"
}
