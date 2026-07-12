package article

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"dh-blog/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type testCache struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

func newTestCache() *testCache { return &testCache{items: make(map[string]interface{})} }

func (c *testCache) Set(key string, value interface{}, _ ...time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
	return nil
}

func (c *testCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.items[key]
	return value, ok
}

func (c *testCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[key]; !ok {
		return false
	}
	delete(c.items, key)
	return true
}

type testComments struct{ count int64 }

func (c testComments) Count(context.Context) (int64, error) { return c.count, nil }

type testAI struct{ tags []string }

func (a testAI) GenerateTags(string, []string) ([]string, error) { return a.tags, nil }

type testTasks struct {
	handler   TagGenerationHandler
	articleID int
	content   string
	submitted bool
}

func (t *testTasks) RegisterTagGenerationHandler(handler TagGenerationHandler) { t.handler = handler }
func (t *testTasks) SubmitTagGeneration(articleID int, content string) {
	t.articleID, t.content, t.submitted = articleID, content, true
}

func openArticleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate article models: %v", err)
	}
	return db
}

func TestModuleOwnsModelsRoutesAndTaskRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openArticleTestDB(t)
	tasks := &testTasks{}
	module, err := New(Dependencies{
		DB: db, Cache: newTestCache(), AI: testAI{}, CommentCounter: testComments{}, Tasks: tasks,
	})
	if err != nil {
		t.Fatal(err)
	}
	var defaultCategory Category
	if err := db.Where("slug = ?", "default").First(&defaultCategory).Error; err != nil {
		t.Fatalf("default category was not created: %v", err)
	}
	if tasks.handler == nil {
		t.Fatal("New() did not register the article tag-generation handler")
	}

	engine := gin.New()
	routes := &router.Routes{Engine: engine, PublicAPI: engine.Group("/api"), AdminAPI: engine.Group("/api/admin")}
	module.RegisterRoutes(routes)
	want := map[string]bool{
		"GET /api/article/:id":                      false,
		"GET /api/article/title/:id":                false,
		"GET /api/article/unlock/:id/:password":     false,
		"POST /api/article/list":                    false,
		"GET /api/article/overview":                 false,
		"GET /api/article/tag":                      false,
		"GET /api/article/category":                 false,
		"GET /api/article/taxonomies":               false,
		"GET /api/article/taxonomy/articles":        false,
		"GET /api/admin/article/:id":                false,
		"POST /api/admin/article":                   false,
		"PUT /api/admin/article":                    false,
		"POST /api/admin/article/list":              false,
		"POST /api/admin/article/:id/generate-tags": false,
		"POST /api/admin/tag":                       false,
		"PUT /api/admin/tag":                        false,
		"DELETE /api/admin/tag/:id":                 false,
		"POST /api/admin/category":                  false,
		"GET /api/admin/category/:id":               false,
		"PUT /api/admin/category":                   false,
		"DELETE /api/admin/category/:id":            false,
		"GET /api/admin/category/:id/tags":          false,
		"POST /api/admin/category/:id/tags":         false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, exists := want[key]; exists {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("route %s was not registered", route)
		}
	}
}

func TestArticleJSONAndTableNamesStayCompatible(t *testing.T) {
	var article Article
	if err := json.Unmarshal([]byte(`{"title":"hello","content":"body","tags":["go","vue"]}`), &article); err != nil {
		t.Fatalf("unmarshal article: %v", err)
	}
	if len(article.TagNames) != 2 || article.TagNames[0] != "go" || article.TagNames[1] != "vue" {
		t.Fatalf("tag names = %#v, want [go vue]", article.TagNames)
	}
	if (Article{}).TableName() != "articles" || (Tag{}).TableName() != "tags" || (Category{}).TableName() != "categories" || (TagRelation{}).TableName() != "tag_relations" {
		t.Fatal("article module table names changed")
	}
}

func TestRepositoryUsesCategoryDefaultsAndAppendsGeneratedTags(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	tags := NewTagRepository(db, cache)
	categories := NewCategoryRepository(db)
	articles := NewArticleRepository(db, categories, tags, cache)

	defaultTag := Tag{Name: "default"}
	if err := db.Create(&defaultTag).Error; err != nil {
		t.Fatalf("create default tag: %v", err)
	}
	category := Category{Name: "Backend", Slug: "backend"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := categories.SaveCategoryDefaultTags(category.ID, []int{defaultTag.ID}); err != nil {
		t.Fatalf("save category defaults: %v", err)
	}
	article := Article{Title: "A", Content: "one two three", CategoryID: category.ID}
	if err := articles.SaveArticle(&article); err != nil {
		t.Fatalf("save article: %v", err)
	}
	if article.WordNum != 3 {
		t.Fatalf("word count = %d, want 3", article.WordNum)
	}

	handler := NewHandler(articles, tags, categories, testComments{}, testAI{tags: []string{"default", "generated"}}, nil)
	if err := handler.ProcessTagGeneration(context.Background(), article.ID, article.Content); err != nil {
		t.Fatalf("process generated tags: %v", err)
	}
	var stored Article
	if err := db.Preload("Tags").First(&stored, article.ID).Error; err != nil {
		t.Fatalf("load stored article: %v", err)
	}
	if len(stored.Tags) != 2 {
		t.Fatalf("stored tags = %#v, want default and generated", stored.Tags)
	}
}

func TestPublicArticlePageRedactsLockedArticleSecrets(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	tags := NewTagRepository(db, cache)
	categories := NewCategoryRepository(db)
	articles := NewArticleRepository(db, categories, tags, cache)

	publicArticle := Article{Title: "public", Content: "visible"}
	lockedArticle := Article{Title: "private", Content: "secret", IsLocked: true, LockPassword: "password"}
	if err := db.Create(&publicArticle).Error; err != nil {
		t.Fatalf("create public article: %v", err)
	}
	if err := db.Create(&lockedArticle).Error; err != nil {
		t.Fatalf("create locked article: %v", err)
	}
	previewTag := Tag{Name: "private-tag"}
	if err := db.Create(&previewTag).Error; err != nil {
		t.Fatalf("create preview tag: %v", err)
	}
	if err := db.Model(&lockedArticle).Association("Tags").Append(&previewTag); err != nil {
		t.Fatalf("attach preview tag: %v", err)
	}

	publicPage, publicTotal, err := articles.FindPublicPage(context.Background(), 1, 10, false)
	if err != nil {
		t.Fatalf("find public page: %v", err)
	}
	if publicTotal != 2 || len(publicPage) != 2 {
		t.Fatalf("public page length = %d, total = %d; want both articles", len(publicPage), publicTotal)
	}
	if publicPage[0].ID != lockedArticle.ID || publicPage[0].Title != lockedArticle.Title {
		t.Fatalf("locked article metadata = %#v; want visible title for article %d", publicPage[0], lockedArticle.ID)
	}
	if publicPage[0].Content != "" || publicPage[0].LockPassword != "" {
		t.Fatalf("locked article leaked protected fields: %#v", publicPage[0])
	}
	if publicPage[0].CanAccess {
		t.Fatalf("visitor unexpectedly received access to locked article: %#v", publicPage[0])
	}
	if len(publicPage[0].Tags) != 0 {
		t.Fatalf("homepage article unexpectedly included tags: %#v", publicPage[0].Tags)
	}
	if publicPage[1].ID != publicArticle.ID || publicPage[1].Content != publicArticle.Content {
		t.Fatalf("public article = %#v; want unchanged article %d", publicPage[1], publicArticle.ID)
	}
	if !publicPage[1].CanAccess {
		t.Fatalf("public article should be accessible: %#v", publicPage[1])
	}

	authenticatedPage, authenticatedTotal, err := articles.FindPublicPage(context.Background(), 1, 10, true)
	if err != nil {
		t.Fatalf("find authenticated homepage: %v", err)
	}
	if authenticatedTotal != 2 || authenticatedPage[0].Content != lockedArticle.Content || !authenticatedPage[0].CanAccess {
		t.Fatalf("authenticated homepage did not expose locked content: %#v", authenticatedPage)
	}
	if authenticatedPage[0].LockPassword != "" {
		t.Fatalf("authenticated homepage leaked lock password: %#v", authenticatedPage[0])
	}

	adminPage, adminTotal, err := articles.FindPage(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("find admin page: %v", err)
	}
	if adminTotal != 2 || len(adminPage) != 2 {
		t.Fatalf("admin page length = %d, total = %d; want both articles", len(adminPage), adminTotal)
	}
}
