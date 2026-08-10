package article

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type testAI struct {
	tags    []string
	summary string
}

func (a testAI) GenerateTags(string, []string) ([]string, error) { return a.tags, nil }
func (a testAI) GenerateSummary(string) (string, error)          { return a.summary, nil }

type testTasks struct {
	handler          TagGenerationHandler
	summaryHandler   SummaryGenerationHandler
	articleID        int
	content          string
	submitted        bool
	summaryArticleID int
	summarySubmitted bool
}

func (t *testTasks) RegisterTagGenerationHandler(handler TagGenerationHandler) { t.handler = handler }
func (t *testTasks) SubmitTagGeneration(articleID int, content string) {
	t.articleID, t.content, t.submitted = articleID, content, true
}
func (t *testTasks) RegisterSummaryGenerationHandler(handler SummaryGenerationHandler) {
	t.summaryHandler = handler
}
func (t *testTasks) SubmitSummaryGeneration(articleID int, _ string) {
	t.summaryArticleID, t.summarySubmitted = articleID, true
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
	if tasks.summaryHandler == nil {
		t.Fatal("New() did not register the article summary-generation handler")
	}

	engine := gin.New()
	routes := &router.Routes{Engine: engine, PublicAPI: engine.Group("/api"), AdminAPI: engine.Group("/api/admin")}
	module.RegisterRoutes(routes)
	want := map[string]bool{
		"GET /api/article/:id":                      false,
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
		"DELETE /api/admin/article/:id":             false,
		"POST /api/admin/article/:id/generate-tags": false,
		"POST /api/admin/article/summaries/batch":  false,
		"GET /api/admin/article/summaries/batch":   false,
		"POST /api/admin/tag":                       false,
		"PUT /api/admin/tag":                        false,
		"DELETE /api/admin/tag/:id":                 false,
		"POST /api/admin/category":                  false,
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

func TestDeleteTagRemovesRelationsWithoutDeletingArticles(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	tags := NewTagRepository(db, cache)

	tag := Tag{Name: "shared"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	category := Category{Name: "Backend", Slug: "backend"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := db.Create(&TagRelation{TagID: tag.ID, RelatedID: category.ID, RelationType: "category"}).Error; err != nil {
		t.Fatalf("create category tag relation: %v", err)
	}
	articles := []Article{
		{Title: "A", Content: "first", Tags: []*Tag{&tag}},
		{Title: "B", Content: "second", Tags: []*Tag{&tag}},
	}
	if err := db.Create(&articles).Error; err != nil {
		t.Fatalf("create tagged articles: %v", err)
	}
	for i := range articles {
		if err := cache.Set(fmt.Sprintf("%s%d", PrefixArticle, articles[i].ID), articles[i]); err != nil {
			t.Fatalf("cache article: %v", err)
		}
	}

	if err := tags.Delete(context.Background(), tag.ID); err != nil {
		t.Fatalf("delete tag: %v", err)
	}

	var articleCount int64
	if err := db.Model(&Article{}).Count(&articleCount).Error; err != nil {
		t.Fatalf("count articles: %v", err)
	}
	if articleCount != 2 {
		t.Fatalf("article count = %d, want 2", articleCount)
	}
	var articleTagCount int64
	if err := db.Table("article_tags").Where("tag_id = ?", tag.ID).Count(&articleTagCount).Error; err != nil {
		t.Fatalf("count article tag relations: %v", err)
	}
	if articleTagCount != 0 {
		t.Fatalf("article tag relation count = %d, want 0", articleTagCount)
	}
	var defaultRelationCount int64
	if err := db.Model(&TagRelation{}).Where("tag_id = ?", tag.ID).Count(&defaultRelationCount).Error; err != nil {
		t.Fatalf("count category tag relations: %v", err)
	}
	if defaultRelationCount != 0 {
		t.Fatalf("category tag relation count = %d, want 0", defaultRelationCount)
	}
	var deletedTagCount int64
	if err := db.Unscoped().Model(&Tag{}).Where("id = ?", tag.ID).Count(&deletedTagCount).Error; err != nil {
		t.Fatalf("count deleted tag: %v", err)
	}
	if deletedTagCount != 0 {
		t.Fatalf("deleted tag count = %d, want 0", deletedTagCount)
	}
	for i := range articles {
		if _, found := cache.Get(fmt.Sprintf("%s%d", PrefixArticle, articles[i].ID)); found {
			t.Fatalf("article %d cache was not cleared", articles[i].ID)
		}
	}
	recreated := Tag{Name: tag.Name}
	if err := tags.Create(context.Background(), &recreated); err != nil {
		t.Fatalf("recreate deleted tag name: %v", err)
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
	if publicPage[0].Content != "" || publicPage[0].Summary != "" || publicPage[0].LockPassword != "" {
		t.Fatalf("locked article leaked protected fields: %#v", publicPage[0])
	}
	if publicPage[0].CanAccess {
		t.Fatalf("visitor unexpectedly received access to locked article: %#v", publicPage[0])
	}
	if len(publicPage[0].Tags) != 0 {
		t.Fatalf("homepage article unexpectedly included tags: %#v", publicPage[0].Tags)
	}
	// 首页只展示摘要：正文不再下发，没有 AI 摘要时用正文截断兜底
	if publicPage[1].ID != publicArticle.ID || publicPage[1].Content != "" {
		t.Fatalf("public article = %#v; want article %d without content", publicPage[1], publicArticle.ID)
	}
	if publicPage[1].Summary != publicArticle.Content {
		t.Fatalf("public article summary = %q; want excerpt %q", publicPage[1].Summary, publicArticle.Content)
	}
	if !publicPage[1].CanAccess {
		t.Fatalf("public article should be accessible: %#v", publicPage[1])
	}

	authenticatedPage, authenticatedTotal, err := articles.FindPublicPage(context.Background(), 1, 10, true)
	if err != nil {
		t.Fatalf("find authenticated homepage: %v", err)
	}
	if authenticatedTotal != 2 || authenticatedPage[0].Summary != lockedArticle.Content || !authenticatedPage[0].CanAccess {
		t.Fatalf("authenticated homepage did not expose locked article preview: %#v", authenticatedPage)
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

func TestExcerptStripsMarkdownAndTruncates(t *testing.T) {
	cases := []struct {
		name    string
		content string
		limit   int
		want    string
	}{
		{name: "去掉标题与强调", content: "# 标题\n**加粗**正文", limit: 50, want: "标题 加粗正文"},
		{name: "链接只保留文字", content: "见 [官方文档](https://example.com) 说明", limit: 50, want: "见 官方文档 说明"},
		{name: "图片整体丢弃", content: "![封面](https://example.com/a.png) 正文", limit: 50, want: "正文"},
		{name: "超长截断并加省略号", content: "一二三四五六七八九十", limit: 4, want: "一二三四…"},
		{name: "空正文", content: "   \n\t ", limit: 10, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := excerpt(tc.content, tc.limit); got != tc.want {
				t.Fatalf("excerpt() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSaveGeneratedSummaryPersistsAndClearsCache(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	articles := NewArticleRepository(db, NewCategoryRepository(db), NewTagRepository(db, cache), cache)

	article := Article{Title: "标题", Content: "正文"}
	if err := db.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}
	_ = cache.Set(fmt.Sprintf("%s%d", PrefixArticle, article.ID), article)

	if err := articles.SaveGeneratedSummary(context.Background(), article.ID, "本文讲述了摘要"); err != nil {
		t.Fatalf("save summary: %v", err)
	}

	var stored Article
	if err := db.First(&stored, article.ID).Error; err != nil {
		t.Fatalf("load article: %v", err)
	}
	if stored.Summary != "本文讲述了摘要" {
		t.Fatalf("stored summary = %q", stored.Summary)
	}
	if _, found := cache.Get(fmt.Sprintf("%s%d", PrefixArticle, article.ID)); found {
		t.Fatal("article cache was not cleared after saving summary")
	}
}

func TestSaveGeneratedSummaryRejectsMissingArticle(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	articles := NewArticleRepository(db, NewCategoryRepository(db), NewTagRepository(db, cache), cache)

	if err := articles.SaveGeneratedSummary(context.Background(), 4242, "摘要"); err == nil {
		t.Fatal("expected an error when the article does not exist")
	}
}

func TestUpdateArticleKeepsStoredSummaryWhenPayloadOmitsIt(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	articles := NewArticleRepository(db, NewCategoryRepository(db), NewTagRepository(db, cache), cache)

	article := Article{Title: "标题", Content: "正文", Summary: "已生成的摘要"}
	if err := db.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}

	// 模拟编辑器提交了不含 summary 的旧草稿
	update := Article{Title: "新标题", Content: "新正文"}
	update.ID = article.ID
	if err := articles.UpdateArticle(&update); err != nil {
		t.Fatalf("update article: %v", err)
	}

	var stored Article
	if err := db.First(&stored, article.ID).Error; err != nil {
		t.Fatalf("load article: %v", err)
	}
	if stored.Summary != "已生成的摘要" {
		t.Fatalf("stored summary = %q, want the previously generated summary", stored.Summary)
	}
	if stored.Title != "新标题" {
		t.Fatalf("stored title = %q, want the updated title", stored.Title)
	}
}

func TestUpdateArticleAcceptsManuallyEditedSummary(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	articles := NewArticleRepository(db, NewCategoryRepository(db), NewTagRepository(db, cache), cache)

	article := Article{Title: "标题", Content: "正文", Summary: "旧摘要"}
	if err := db.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}

	update := Article{Title: "标题", Content: "正文", Summary: "手写摘要"}
	update.ID = article.ID
	if err := articles.UpdateArticle(&update); err != nil {
		t.Fatalf("update article: %v", err)
	}

	var stored Article
	if err := db.First(&stored, article.ID).Error; err != nil {
		t.Fatalf("load article: %v", err)
	}
	if stored.Summary != "手写摘要" {
		t.Fatalf("stored summary = %q, want the manually edited summary", stored.Summary)
	}
}

func TestProcessSummaryGenerationStoresSummary(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	tags := NewTagRepository(db, cache)
	categories := NewCategoryRepository(db)
	articles := NewArticleRepository(db, categories, tags, cache)

	article := Article{Title: "标题", Content: "正文"}
	if err := db.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}

	handler := NewHandler(articles, tags, categories, testComments{}, testAI{summary: "本文讲述了要点"}, nil)
	if err := handler.ProcessSummaryGeneration(context.Background(), article.ID, article.Content); err != nil {
		t.Fatalf("process summary: %v", err)
	}

	var stored Article
	if err := db.First(&stored, article.ID).Error; err != nil {
		t.Fatalf("load article: %v", err)
	}
	if stored.Summary != "本文讲述了要点" {
		t.Fatalf("stored summary = %q", stored.Summary)
	}
}

func TestProcessSummaryGenerationSkipsEmptyContent(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	tags := NewTagRepository(db, cache)
	categories := NewCategoryRepository(db)
	articles := NewArticleRepository(db, categories, tags, cache)

	handler := NewHandler(articles, tags, categories, testComments{}, testAI{summary: "不应写入"}, nil)
	if err := handler.ProcessSummaryGeneration(context.Background(), 1, "   "); err != nil {
		t.Fatalf("empty content should be skipped without error, got %v", err)
	}
}

func TestProcessSummaryGenerationRejectsEmptyAIResult(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	tags := NewTagRepository(db, cache)
	categories := NewCategoryRepository(db)
	articles := NewArticleRepository(db, categories, tags, cache)

	article := Article{Title: "标题", Content: "正文"}
	if err := db.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}

	handler := NewHandler(articles, tags, categories, testComments{}, testAI{summary: "  "}, nil)
	if err := handler.ProcessSummaryGeneration(context.Background(), article.ID, article.Content); err == nil {
		t.Fatal("expected an error when the AI returns an empty summary")
	}

	var stored Article
	if err := db.First(&stored, article.ID).Error; err != nil {
		t.Fatalf("load article: %v", err)
	}
	if stored.Summary != "" {
		t.Fatalf("stored summary = %q, want it left untouched", stored.Summary)
	}
}

func TestDeleteArticleRemovesItFromListings(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	tags := NewTagRepository(db, cache)
	categories := NewCategoryRepository(db)
	articles := NewArticleRepository(db, categories, tags, cache)

	keep := Article{Title: "保留", Content: "正文"}
	drop := Article{Title: "删除", Content: "正文"}
	if err := db.Create(&keep).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}
	if err := db.Create(&drop).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}
	_ = cache.Set(fmt.Sprintf("%s%d", PrefixArticle, drop.ID), drop)

	if err := articles.Delete(context.Background(), drop.ID); err != nil {
		t.Fatalf("delete article: %v", err)
	}

	if _, found := cache.Get(fmt.Sprintf("%s%d", PrefixArticle, drop.ID)); found {
		t.Fatal("deleted article cache was not cleared")
	}

	// 软删除：默认作用域下查不到，后台与首页列表都不应再出现
	if err := db.First(&Article{}, drop.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted article is still reachable: %v", err)
	}
	adminPage, adminTotal, err := articles.FindPage(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("find admin page: %v", err)
	}
	if adminTotal != 1 || len(adminPage) != 1 || adminPage[0].ID != keep.ID {
		t.Fatalf("admin page = %#v (total %d), want only article %d", adminPage, adminTotal, keep.ID)
	}
	publicPage, publicTotal, err := articles.FindPublicPage(context.Background(), 1, 10, false)
	if err != nil {
		t.Fatalf("find public page: %v", err)
	}
	if publicTotal != 1 || len(publicPage) != 1 || publicPage[0].ID != keep.ID {
		t.Fatalf("public page = %#v (total %d), want only article %d", publicPage, publicTotal, keep.ID)
	}
}

func TestDeleteArticleRejectsUnknownID(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	articles := NewArticleRepository(db, NewCategoryRepository(db), NewTagRepository(db, cache), cache)

	err := articles.Delete(context.Background(), 4242)
	if !errors.Is(err, ErrArticleNotFound) {
		t.Fatalf("delete unknown article err = %v, want ErrArticleNotFound", err)
	}
}

func TestDeleteArticleIsNotRepeatable(t *testing.T) {
	db := openArticleTestDB(t)
	cache := newTestCache()
	articles := NewArticleRepository(db, NewCategoryRepository(db), NewTagRepository(db, cache), cache)

	article := Article{Title: "标题", Content: "正文"}
	if err := db.Create(&article).Error; err != nil {
		t.Fatalf("create article: %v", err)
	}
	if err := articles.Delete(context.Background(), article.ID); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	// 重复删除必须报错，否则前端会对已删除的文章提示成功
	if err := articles.Delete(context.Background(), article.ID); !errors.Is(err, ErrArticleNotFound) {
		t.Fatalf("second delete err = %v, want ErrArticleNotFound", err)
	}
}
