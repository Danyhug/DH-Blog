package app_test

import (
	"reflect"
	"testing"

	"dh-blog/internal/app"
	articlemodule "dh-blog/internal/modules/article"
	commentmodule "dh-blog/internal/modules/comment"
	filesmodule "dh-blog/internal/modules/files"
	loggingmodule "dh-blog/internal/modules/logging"
	sharemodule "dh-blog/internal/modules/share"
	systemmodule "dh-blog/internal/modules/system"
	usermodule "dh-blog/internal/modules/user"
)

func TestSchemaModelsContainModuleModelsWithoutDuplicates(t *testing.T) {
	models := app.SchemaModels()
	seen := make(map[reflect.Type]bool, len(models))
	for _, model := range models {
		modelType := reflect.TypeOf(model)
		if seen[modelType] {
			t.Errorf("duplicate migration model %v", modelType)
		}
		seen[modelType] = true
	}

	for _, modelType := range []reflect.Type{
		reflect.TypeOf(&articlemodule.Article{}),
		reflect.TypeOf(&articlemodule.Category{}),
		reflect.TypeOf(&articlemodule.Tag{}),
		reflect.TypeOf(&articlemodule.TagRelation{}),
		reflect.TypeOf(&commentmodule.Comment{}),
		reflect.TypeOf(&filesmodule.File{}),
		reflect.TypeOf(&loggingmodule.AccessLog{}),
		reflect.TypeOf(&loggingmodule.IPBlacklist{}),
		reflect.TypeOf(&sharemodule.Share{}),
		reflect.TypeOf(&sharemodule.ShareAccessLog{}),
		reflect.TypeOf(&systemmodule.Setting{}),
		reflect.TypeOf(&usermodule.User{}),
	} {
		if !seen[modelType] {
			t.Errorf("module migration model %v is not registered", modelType)
		}
	}
}
