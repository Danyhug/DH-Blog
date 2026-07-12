package article

import (
	"encoding/json"

	"dh-blog/internal/model"
)

// Article 对应数据库中的 articles 表。
type Article struct {
	model.BaseModel `gorm:"embedded"`
	Title           string `gorm:"column:title;not null" json:"title"`
	Content         string `gorm:"column:content;not null" json:"content"`
	CategoryID      int    `gorm:"column:category_id" json:"categoryId"`
	Views           int    `gorm:"column:views;default:0" json:"views"`
	WordNum         int    `gorm:"column:word_num" json:"wordNum"`
	ThumbnailURL    string `gorm:"column:thumbnail_url" json:"thumbnailUrl"`
	IsLocked        bool   `gorm:"column:is_locked;default:false" json:"isLocked"`
	LockPassword    string `gorm:"column:lock_password" json:"lockPassword"`
	CanAccess       bool   `gorm:"-" json:"canAccess"`

	Tags     []*Tag   `gorm:"many2many:article_tags;" json:"tags"`
	TagNames []string `gorm:"-" json:"tagNames,omitempty"`
}

// UnmarshalJSON keeps accepting the historical request shape where tags is a
// list of tag names while responses expose the populated Tag objects.
func (a *Article) UnmarshalJSON(data []byte) error {
	type alias Article
	aux := &struct {
		TagNames []string `json:"tags"`
		*alias
	}{alias: (*alias)(a)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	a.TagNames = aux.TagNames
	return nil
}

func (Article) TableName() string { return "articles" }

// Tag 对应数据库中的 tags 表。
type Tag struct {
	model.BaseModel `gorm:"embedded"`
	Name            string     `gorm:"column:name;not null;uniqueIndex" json:"name"`
	Articles        []*Article `gorm:"many2many:article_tags;" json:"articles,omitempty"`
}

func (Tag) TableName() string { return "tags" }

// Category 对应数据库中的 categories 表。
type Category struct {
	model.BaseModel `gorm:"embedded"`
	Name            string `gorm:"column:name;not null;uniqueIndex" json:"name"`
	Slug            string `gorm:"column:slug;not null;uniqueIndex" json:"slug"`
	TagIDs          []int  `gorm:"-" json:"tagIds"`
}

func (Category) TableName() string { return "categories" }

// TagRelation 保存分类等实体的默认标签关系。
type TagRelation struct {
	TagID        int    `gorm:"primaryKey"`
	RelatedID    int    `gorm:"primaryKey"`
	RelationType string `gorm:"primaryKey"`
}

func (TagRelation) TableName() string { return "tag_relations" }
