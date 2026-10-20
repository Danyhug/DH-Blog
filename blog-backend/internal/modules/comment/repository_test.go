package comment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type testArticle struct {
	ID        int `gorm:"primaryKey"`
	Title     string
	DeletedAt gorm.DeletedAt
}

func (testArticle) TableName() string { return "articles" }

// stubPolicy 让测试可以固定评论开关的取值。
type stubPolicy struct {
	open bool
	err  error
}

func (s stubPolicy) CommentsOpen(context.Context) (bool, error) { return s.open, s.err }

func newTestModule(t *testing.T) *Module {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	module := New(db, stubPolicy{open: true})
	if err := db.AutoMigrate(append(MigrationModels(), &testArticle{})...); err != nil {
		t.Fatalf("migrate comment model: %v", err)
	}
	return module
}

func TestRepositoryGroupsCommentsByArticle(t *testing.T) {
	repository := newTestModule(t).repository
	if err := repository.db.Create(&[]testArticle{
		{ID: 1, Title: "第一篇文章"},
		{ID: 2, Title: "第二篇文章"},
	}).Error; err != nil {
		t.Fatalf("create articles: %v", err)
	}

	firstRoot := &Comment{ArticleID: 1, Author: "root", Email: "root@example.com", Content: "root", IsPublic: true, UA: "test"}
	if err := repository.AddComment(firstRoot); err != nil {
		t.Fatalf("add first root: %v", err)
	}
	parentID := firstRoot.ID
	if err := repository.AddComment(&Comment{ArticleID: 1, Author: "child", Email: "child@example.com", Content: "child", IsPublic: true, ParentID: &parentID, UA: "test"}); err != nil {
		t.Fatalf("add child: %v", err)
	}
	if err := repository.AddComment(&Comment{ArticleID: 2, Author: "other", Email: "other@example.com", Content: "other", IsPublic: true, UA: "test"}); err != nil {
		t.Fatalf("add second article comment: %v", err)
	}

	groups, total, err := repository.GetCommentGroups(1, 10)
	if err != nil {
		t.Fatalf("get comment groups: %v", err)
	}
	if total != 2 {
		t.Fatalf("article group total = %d, want 2", total)
	}
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}

	groupByArticle := make(map[int]*ArticleCommentGroup, len(groups))
	for _, group := range groups {
		groupByArticle[group.ArticleID] = group
	}
	firstGroup := groupByArticle[1]
	if firstGroup == nil || firstGroup.ArticleTitle != "第一篇文章" || firstGroup.CommentCount != 2 {
		t.Fatalf("first article group = %#v", firstGroup)
	}
	if len(firstGroup.Children) != 1 || len(firstGroup.Children[0].Children) != 1 {
		t.Fatalf("first article comment tree = %#v", firstGroup.Children)
	}
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

	comments, rootTotal, commentTotal, err := repository.GetCommentsByArticleID(7, 1, 10)
	if err != nil {
		t.Fatalf("get article comments: %v", err)
	}
	if commentTotal != 2 {
		t.Fatalf("public comment total = %d, want 2", commentTotal)
	}
	if rootTotal != 1 {
		t.Fatalf("root comment total = %d, want 1", rootTotal)
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

// 分页切的是顶层评论：第二页要能拿到剩下的顶层评论，而回复始终跟着自己的父评论走。
func TestRepositoryPaginatesRootComments(t *testing.T) {
	repository := newTestModule(t).repository

	// 12 条顶层评论，每条带 2 条回复，正好跨两页（每页 10 条）
	for i := 0; i < 12; i++ {
		root := &Comment{ArticleID: 42, Author: "root", Email: "root@example.com", Content: "root", IsPublic: true, UA: "test"}
		if err := repository.AddComment(root); err != nil {
			t.Fatalf("add root %d: %v", i, err)
		}
		for j := 0; j < 2; j++ {
			parentID := root.ID
			child := &Comment{ArticleID: 42, Author: "child", Email: "child@example.com", Content: "child", IsPublic: true, ParentID: &parentID, UA: "test"}
			if err := repository.AddComment(child); err != nil {
				t.Fatalf("add child %d-%d: %v", i, j, err)
			}
		}
	}

	firstPage, rootTotal, commentTotal, err := repository.GetCommentsByArticleID(42, 1, 10)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if rootTotal != 12 || commentTotal != 36 {
		t.Fatalf("totals = (%d roots, %d comments), want (12, 36)", rootTotal, commentTotal)
	}
	if len(firstPage) != 10 {
		t.Fatalf("first page roots = %d, want 10", len(firstPage))
	}
	for _, root := range firstPage {
		if len(root.Children) != 2 {
			t.Fatalf("root %d children = %d, want 2 (回复必须跟父评论同页)", root.ID, len(root.Children))
		}
	}

	secondPage, _, _, err := repository.GetCommentsByArticleID(42, 2, 10)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(secondPage) != 2 {
		t.Fatalf("second page roots = %d, want 2", len(secondPage))
	}
	for _, root := range secondPage {
		if len(root.Children) != 2 {
			t.Fatalf("second page root %d children = %d, want 2", root.ID, len(root.Children))
		}
	}

	// 两页不能有重叠
	seen := make(map[int]bool, 12)
	for _, page := range [][]*Comment{firstPage, secondPage} {
		for _, root := range page {
			if seen[root.ID] {
				t.Fatalf("root %d appears on both pages", root.ID)
			}
			seen[root.ID] = true
		}
	}
	if len(seen) != 12 {
		t.Fatalf("distinct roots across pages = %d, want 12", len(seen))
	}

	// 越界的页码应当是空列表，不是报错
	beyond, _, _, err := repository.GetCommentsByArticleID(42, 99, 10)
	if err != nil {
		t.Fatalf("page beyond end: %v", err)
	}
	if len(beyond) != 0 {
		t.Fatalf("page beyond end = %d roots, want 0", len(beyond))
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

// 前端分页器读的是 rootTotal（顶层评论数），「N 条评论」读的是 total。
// 这条响应契约不能被悄悄改回单一的 total，否则评论一多就会多出一堆空页。
func TestGetCommentsEndpointReturnsPageShape(t *testing.T) {
	module := newTestModule(t)
	for i := 0; i < 12; i++ {
		if err := module.repository.AddComment(&Comment{
			ArticleID: 9, Author: "root", Email: "root@example.com", Content: "root", IsPublic: true, UA: "test",
		}); err != nil {
			t.Fatalf("add comment %d: %v", i, err)
		}
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	module.RegisterRoutes(&router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/comment/9?page=2&pageSize=10", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", recorder.Code, recorder.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			Total     int64     `json:"total"`
			RootTotal int64     `json:"rootTotal"`
			List      []Comment `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 1 {
		t.Fatalf("code = %d, want 1 (body: %s)", body.Code, recorder.Body.String())
	}
	if body.Data.Total != 12 || body.Data.RootTotal != 12 {
		t.Fatalf("totals = (%d total, %d root), want (12, 12)", body.Data.Total, body.Data.RootTotal)
	}
	if len(body.Data.List) != 2 {
		t.Fatalf("page 2 of 12 roots should hold 2 items, got %d", len(body.Data.List))
	}
}

func postComment(t *testing.T, module *Module) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	routes := &router.Routes{
		Engine:    engine,
		PublicAPI: engine.Group("/api"),
		AdminAPI:  engine.Group("/api/admin"),
	}
	module.RegisterRoutes(routes)

	body := `{"articleId":1,"author":"访客","email":"guest@example.com","content":"你好","isPublic":true}`
	request := httptest.NewRequest(http.MethodPost, "/api/comment", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}

func TestAddCommentRejectedWhenCommentsClosed(t *testing.T) {
	module := newTestModule(t)
	module.handler.policy = stubPolicy{open: false}

	recorder := postComment(t, module)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), ErrCommentClosed.Error()) {
		t.Fatalf("body = %q, want it to mention %q", recorder.Body.String(), ErrCommentClosed.Error())
	}
	var count int64
	if err := module.repository.db.Model(&Comment{}).Count(&count).Error; err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if count != 0 {
		t.Fatalf("comment count = %d, want 0 when comments are closed", count)
	}
}

func TestAddCommentAcceptedWhenCommentsOpen(t *testing.T) {
	module := newTestModule(t)
	module.handler.policy = stubPolicy{open: true}

	recorder := postComment(t, module)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	var count int64
	if err := module.repository.db.Model(&Comment{}).Count(&count).Error; err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if count != 1 {
		t.Fatalf("comment count = %d, want 1", count)
	}
}

func TestAddCommentSurfacesPolicyError(t *testing.T) {
	module := newTestModule(t)
	module.handler.policy = stubPolicy{err: errors.New("配置不可用")}

	recorder := postComment(t, module)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
