package comment

import (
	"context"
	"testing"

	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestModule(t *testing.T) *Module {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	module := New(db)
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate comment model: %v", err)
	}
	return module
}

func TestModuleRegistersCommentRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	module := newTestModule(t)
	engine := gin.New()
	routes := &router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
	}
	module.RegisterRoutes(routes)

	want := map[string]bool{
		"POST /api/comment":                         false,
		"GET /api/comment/:articleId":               false,
		"GET /api/admin/comment/:pageSize/:pageNum": false,
		"PUT /api/admin/comment":                    false,
		"POST /api/admin/comment/reply":             false,
		"DELETE /api/admin/comment/:id":             false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("route %s was not registered", route)
		}
	}
}

func TestMigrationModels(t *testing.T) {
	models := MigrationModels()
	if len(models) != 1 {
		t.Fatalf("MigrationModels() len = %d, want 1", len(models))
	}
	if _, ok := models[0].(*Comment); !ok {
		t.Fatalf("MigrationModels()[0] type = %T, want *Comment", models[0])
	}
	if got := (Comment{}).TableName(); got != "comments" {
		t.Fatalf("TableName() = %q, want comments", got)
	}
}

func TestRepositoryBuildsPublicCommentTree(t *testing.T) {
	repository := newTestModule(t).repository
	root := &Comment{ArticleID: 7, Author: "root", Email: "root@example.com", Content: "root", IsPublic: true, UA: "test"}
	if err := repository.AddComment(root); err != nil {
		t.Fatalf("add root comment: %v", err)
	}
	parentID := root.ID
	child := &Comment{ArticleID: 7, Author: "child", Email: "child@example.com", Content: "child", IsPublic: true, ParentID: &parentID, UA: "test"}
	if err := repository.AddComment(child); err != nil {
		t.Fatalf("add child comment: %v", err)
	}
	private := &Comment{ArticleID: 7, Author: "private", Email: "private@example.com", Content: "private", IsPublic: false, UA: "test"}
	if err := repository.AddComment(private); err != nil {
		t.Fatalf("add private comment: %v", err)
	}
	private.IsPublic = false
	if err := repository.UpdateComment(private); err != nil {
		t.Fatalf("make comment private: %v", err)
	}

	comments, total, err := repository.GetCommentsByArticleID(7)
	if err != nil {
		t.Fatalf("get article comments: %v", err)
	}
	if total != 2 {
		t.Fatalf("public comment total = %d, want 2", total)
	}
	if len(comments) != 1 || comments[0].ID != root.ID {
		t.Fatalf("root comments = %#v, want root ID %d", comments, root.ID)
	}
	if len(comments[0].Children) != 1 || comments[0].Children[0].ID != child.ID {
		t.Fatalf("root children = %#v, want child ID %d", comments[0].Children, child.ID)
	}

	count, err := repository.Count(context.Background())
	if err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if count != 3 {
		t.Fatalf("comment count = %d, want 3", count)
	}
}

func TestRepositoryDeleteCommentDeletesDescendants(t *testing.T) {
	repository := newTestModule(t).repository
	root := &Comment{ArticleID: 1, Author: "root", Email: "root@example.com", Content: "root", IsPublic: true, UA: "test"}
	if err := repository.AddComment(root); err != nil {
		t.Fatalf("add root comment: %v", err)
	}
	parentID := root.ID
	child := &Comment{ArticleID: 1, Author: "child", Email: "child@example.com", Content: "child", IsPublic: true, ParentID: &parentID, UA: "test"}
	if err := repository.AddComment(child); err != nil {
		t.Fatalf("add child comment: %v", err)
	}

	if err := repository.DeleteComment(root.ID); err != nil {
		t.Fatalf("delete root comment: %v", err)
	}
	count, err := repository.Count(context.Background())
	if err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if count != 0 {
		t.Fatalf("comment count after delete = %d, want 0", count)
	}
}
