package app

import "dh-blog/internal/model"

// SchemaModels lists the GORM models owned by the application.
func SchemaModels() []any {
	return []any{
		&model.AccessLog{},
		&model.Article{},
		&model.Category{},
		&model.Comment{},
		&model.Tag{},
		&model.User{},
		&model.SystemSetting{},
		&model.IPBlacklist{},
		&model.TagRelation{},
		&model.File{},
		&model.Share{},
		&model.ShareAccessLog{},
	}
}
