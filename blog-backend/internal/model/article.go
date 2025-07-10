package model

// Article 对应于数据库中的 `articles` 表
import "encoding/json"

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

	Tags     []*Tag   `gorm:"many2many:article_tags;" json:"tags"` // 文章关联的标签
	TagNames []string `gorm:"-" json:"tagNames,omitempty"`         // 接收前端传来的标签名数组
}

// UnmarshalJSON 自定义 JSON 反序列化逻辑
func (a *Article) UnmarshalJSON(data []byte) error {
	type Alias Article // 创建一个别名类型，避免无限递归调用 UnmarshalJSON

	aux := &struct {
		TagNames []string `json:"tags"` // 将传入的 "tags" 字段解析到 TagNames
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 将解析到的 TagNames 赋值给 Article 结构体的 TagNames 字段
	a.TagNames = aux.TagNames
	return nil
}

// TableName 指定 GORM 使用的表名
func (*Article) TableName() string {
	return "articles"
}
