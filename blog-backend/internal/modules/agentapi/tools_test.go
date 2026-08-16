package agentapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	article "dh-blog/internal/modules/article"
	"dh-blog/internal/platform/mcp"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// tinyPNG is a valid 1x1 transparent PNG, used where the tool must accept a
// real image payload.
const tinyPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

// testIdentity is a fixed gateway identity: deterministic key id, author name
// and scope set, with no aigateway dependency.
type testIdentity struct {
	keyID  int
	author string
	scopes map[string]bool
}

func (i testIdentity) KeyID() int                 { return i.keyID }
func (i testIdentity) AuthorName() string         { return i.author }
func (i testIdentity) HasScope(scope string) bool { return i.scopes[scope] }

func identity(keyID int, scopes ...string) testIdentity {
	allowed := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		allowed[scope] = true
	}
	return testIdentity{keyID: keyID, author: "测试 Agent", scopes: allowed}
}

// testCache mirrors the article module's test cache: a thread-safe map that
// satisfies the Cache port the article repositories need.
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

// stubImages records what upload_image handed it and answers a fixed URL. File
// persistence itself belongs to the files module's tests; here the upload
// tool's base64 / size / MIME / name-safety checks run against the real path.
type stubImages struct {
	url       string
	savedName string
	savedData []byte
	calls     int
}

func (s *stubImages) SaveBlogImage(_ context.Context, fileName string, data []byte) (string, error) {
	s.savedName = fileName
	s.savedData = data
	s.calls++
	return s.url, nil
}

// recordingTasks captures background task submissions instead of running them.
type recordingTasks struct {
	tagIDs      []int
	tagContents []string
	summaryIDs  []int
}

func (t *recordingTasks) SubmitTagGeneration(articleID int, content string) {
	t.tagIDs = append(t.tagIDs, articleID)
	t.tagContents = append(t.tagContents, content)
}

func (t *recordingTasks) SubmitSummaryGeneration(articleID int, content string) {
	t.summaryIDs = append(t.summaryIDs, articleID)
}

// recordingEvents captures ContentReporter calls for the audit assertions.
type recordingEvents struct {
	created []struct {
		agent, title string
		id           int
	}
	updated []struct {
		agent, title string
		id           int
		viaGrant     bool
	}
	denied []struct {
		agent, title, reason string
		id                   int
	}
}

func (e *recordingEvents) ArticleCreated(agent, title string, articleID int) {
	e.created = append(e.created, struct {
		agent, title string
		id           int
	}{agent, title, articleID})
}

func (e *recordingEvents) ArticleUpdated(agent, title string, articleID int, viaGrant bool) {
	e.updated = append(e.updated, struct {
		agent, title string
		id           int
		viaGrant     bool
	}{agent, title, articleID, viaGrant})
}

func (e *recordingEvents) ArticleUpdateDenied(agent, title string, articleID int, reason string) {
	e.denied = append(e.denied, struct {
		agent, title, reason string
		id                   int
	}{agent, title, reason, articleID})
}

// toolFixture assembles the tool layer with real article and grant services
// over one in-memory SQLite database, plus stubs for the parts that do not
// belong here (images, tasks, events).
type toolFixture struct {
	db       *gorm.DB
	articles article.ContentService
	grants   *grantService
	images   *stubImages
	tasks    *recordingTasks
	events   *recordingEvents
	tools    []mcp.Tool
}

func newToolFixture(t *testing.T) *toolFixture {
	return newToolFixtureWithEvents(t, &recordingEvents{})
}

func newToolFixtureWithEvents(t *testing.T, events *recordingEvents) *toolFixture {
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
	if err := db.AutoMigrate(append(MigrationModels(), article.MigrationModels()...)...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	module, err := article.New(article.Dependencies{DB: db, Cache: newTestCache()})
	if err != nil {
		t.Fatalf("build article module: %v", err)
	}
	images := &stubImages{url: "/api/uploads/blog/x.png"}
	tasks := &recordingTasks{}
	// A nil *recordingEvents must reach New() as an untyped nil interface;
	// assigning the typed nil straight into the interface field would produce
	// a non-nil interface and bypass the noop fallback.
	var reporter ContentReporter
	if events != nil {
		reporter = events
	}

	agentModule, err := New(Dependencies{
		DB:       db,
		Articles: module.ContentService(),
		Images:   images,
		Tasks:    tasks,
		Events:   reporter,
	})
	if err != nil {
		t.Fatalf("build agentapi module: %v", err)
	}
	// The tool layer validates through the module's own grant service; tests
	// pin its clock so expiry assertions run against a known instant.
	grants := agentModule.service
	grants.now = func() time.Time { return fixedNow() }
	return &toolFixture{
		db:       db,
		articles: module.ContentService(),
		grants:   grants,
		images:   images,
		tasks:    tasks,
		events:   events,
		tools:    agentModule.MCPTools(),
	}
}

// tool finds a tool by name in the fixture.
func (f *toolFixture) tool(t *testing.T, name string) mcp.Tool {
	t.Helper()
	for _, tool := range f.tools {
		if tool.Name() == name {
			return tool
		}
	}
	t.Fatalf("tool %q not found", name)
	return nil
}

// callTool runs a tool with the given identity in the context. On success the
// parsed JSON text is returned; on failure text is returned as-is.
func callTool(t *testing.T, tool mcp.Tool, id Identity, args any) (result mcp.Result, data map[string]any, text string) {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	ctx := context.Background()
	if id != nil {
		ctx = IdentityContext(ctx, id)
	}
	result = tool.Call(ctx, raw)
	text = result.Content[0].Text
	if result.IsError {
		return result, nil, text
	}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("success text is not JSON: %v", err)
	}
	return result, data, text
}

// createArticle seeds an article through the real ContentService and returns
// its id, mirroring what an admin-created article looks like.
func (f *toolFixture) createArticle(t *testing.T, title string, authorKeyID int) int {
	t.Helper()
	id, err := f.articles.Create(context.Background(), article.CreateInput{
		Title:       title,
		Content:     "正文 " + title,
		AuthorType:  "agent",
		AuthorName:  "写手",
		AuthorKeyID: authorKeyID,
	})
	if err != nil {
		t.Fatalf("seed article: %v", err)
	}
	return id
}

func TestIdentityContextRoundTrip(t *testing.T) {
	expected := identity(7, scopeContentRead)
	ctx := IdentityContext(context.Background(), expected)
	got, ok := IdentityFrom(ctx)
	if !ok {
		t.Fatal("IdentityFrom lost the identity")
	}
	if got.KeyID() != 7 || got.AuthorName() != "测试 Agent" || !got.HasScope(scopeContentRead) {
		t.Fatalf("IdentityFrom = %#v", got)
	}
}

func TestIdentityFromMissingContext(t *testing.T) {
	if _, ok := IdentityFrom(context.Background()); ok {
		t.Fatal("IdentityFrom on a bare context must not report an identity")
	}
}

func TestModuleMCPToolsAreFiveAndScoped(t *testing.T) {
	fixture := newToolFixture(t)
	if len(fixture.tools) != 5 {
		t.Fatalf("MCPTools() returned %d tools, want 5", len(fixture.tools))
	}
	seen := make(map[string]bool)
	for _, tool := range fixture.tools {
		scoped, ok := tool.(interface{ Scope() string })
		if !ok {
			t.Fatalf("tool %s lacks Scope()", tool.Name())
		}
		if scoped.Scope() == "" {
			t.Fatalf("tool %s has an empty scope", tool.Name())
		}
		seen[tool.Name()] = true
	}
	for _, name := range []string{"list_articles", "get_article", "create_article", "update_article", "upload_image"} {
		if !seen[name] {
			t.Fatalf("MCPTools() missing %s", name)
		}
	}
}

func TestToolDefinitionsAdvertiseContractParameters(t *testing.T) {
	fixture := newToolFixture(t)
	cases := map[string]struct {
		required     []string
		optional     []string
		editTokenDoc bool
	}{
		"list_articles":  {optional: []string{"keyword", "page", "page_size"}},
		"get_article":    {required: []string{"id"}},
		"create_article": {required: []string{"title", "content"}, optional: []string{"summary", "category", "tags", "thumbnail_url"}},
		"update_article": {required: []string{"id"}, optional: []string{"edit_token", "title", "content", "summary", "category", "tags"}, editTokenDoc: true},
		"upload_image":   {required: []string{"file_name", "data"}},
	}
	for name, want := range cases {
		tool := fixture.tool(t, name)
		def := tool.Definition(context.Background())
		if def.Name != name {
			t.Fatalf("%s Definition.Name = %q", name, def.Name)
		}
		schema, ok := def.InputSchema.(map[string]any)
		if !ok {
			t.Fatalf("%s schema is %T, want map", name, def.InputSchema)
		}
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s schema has no properties map: %#v", name, schema)
		}
		for _, field := range want.required {
			if _, ok := properties[field]; !ok {
				t.Errorf("%s schema missing required field %q", name, field)
			}
		}
		for _, field := range want.optional {
			if _, ok := properties[field]; !ok {
				t.Errorf("%s schema missing optional field %q", name, field)
			}
		}
		required, _ := schema["required"].([]string)
		if len(required) != len(want.required) {
			t.Errorf("%s required = %#v, want %v", name, required, want.required)
		}
		if want.editTokenDoc && !strings.Contains(def.Description, "edit_token") {
			t.Errorf("%s description does not explain edit_token", name)
		}
		if want.editTokenDoc && !strings.Contains(def.Description, "站长") {
			t.Errorf("%s description does not tell the model the token comes from the owner", name)
		}
	}
	// The read-only tools must not demand the write scope for their scope().
	listTool := fixture.tool(t, "list_articles")
	if listTool.(interface{ Scope() string }).Scope() != scopeContentRead {
		t.Errorf("list_articles scope = %q, want %q", listTool.(interface{ Scope() string }).Scope(), scopeContentRead)
	}
	updateTool := fixture.tool(t, "update_article")
	if updateTool.(interface{ Scope() string }).Scope() != scopeContentWrite {
		t.Errorf("update_article scope = %q, want %q", updateTool.(interface{ Scope() string }).Scope(), scopeContentWrite)
	}
}

func TestToolsRejectMissingIdentity(t *testing.T) {
	fixture := newToolFixture(t)
	for _, tool := range fixture.tools {
		result := tool.Call(context.Background(), nil)
		if !result.IsError {
			t.Errorf("%s succeeded without an identity", tool.Name())
			continue
		}
		if !strings.Contains(result.Content[0].Text, "身份") {
			t.Errorf("%s error %q does not mention identity", tool.Name(), result.Content[0].Text)
		}
	}
}

func TestWriteToolsRejectScopeMissing(t *testing.T) {
	fixture := newToolFixture(t)
	reader := identity(7, scopeContentRead)
	for _, name := range []string{"create_article", "update_article", "upload_image"} {
		tool := fixture.tool(t, name)
		result, _, text := callTool(t, tool, reader, map[string]any{"title": "x", "content": "y", "id": 1, "file_name": "a.png", "data": tinyPNG})
		if !result.IsError {
			t.Errorf("%s succeeded without content:write", name)
			continue
		}
		if !strings.Contains(text, "权限") {
			t.Errorf("%s error %q does not mention missing permission", name, text)
		}
	}
}

func TestListArticlesKeyword(t *testing.T) {
	fixture := newToolFixture(t)
	fixture.createArticle(t, "Go 并发模型", 0)
	fixture.createArticle(t, "Vue 响应式", 0)
	tool := fixture.tool(t, "list_articles")
	_, data, _ := callTool(t, tool, identity(7, scopeContentRead), map[string]any{"keyword": "Go"})
	articles := data["articles"].([]any)
	if len(articles) != 1 {
		t.Fatalf("keyword hit = %d articles, want 1", len(articles))
	}
	first := articles[0].(map[string]any)
	if first["title"] != "Go 并发模型" {
		t.Fatalf("hit title = %v", first["title"])
	}
	if data["total"].(float64) != 1 {
		t.Fatalf("total = %v, want 1", data["total"])
	}
}

func TestListArticlesPaginationAndDefaults(t *testing.T) {
	fixture := newToolFixture(t)
	for i := 0; i < 3; i++ {
		fixture.createArticle(t, "文章"+string(rune('A'+i)), 0)
	}
	tool := fixture.tool(t, "list_articles")

	_, data, _ := callTool(t, tool, identity(7, scopeContentRead), map[string]any{"page": 2, "page_size": 2})
	if data["total"].(float64) != 3 {
		t.Fatalf("total = %v, want 3", data["total"])
	}
	if len(data["articles"].([]any)) != 1 {
		t.Fatalf("page 2 has %d articles, want 1", len(data["articles"].([]any)))
	}
	if data["page"].(float64) != 2 || data["pageSize"].(float64) != 2 {
		t.Fatalf("echoed page/pageSize = %v/%v", data["page"], data["pageSize"])
	}

	// No args: defaults to page 1 / page_size 20 and returns everything.
	_, defaults, _ := callTool(t, tool, identity(7, scopeContentRead), nil)
	if len(defaults["articles"].([]any)) != 3 {
		t.Fatalf("defaults returned %d articles, want 3", len(defaults["articles"].([]any)))
	}
	if defaults["pageSize"].(float64) != 20 {
		t.Fatalf("default pageSize = %v, want 20", defaults["pageSize"])
	}
}

func TestListArticlesRejectsPageSizeOverLimit(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "list_articles")
	result, _, text := callTool(t, tool, identity(7, scopeContentRead), map[string]any{"page_size": 51})
	if !result.IsError {
		t.Fatal("page_size 51 must be rejected")
	}
	if !strings.Contains(text, "50") {
		t.Fatalf("error %q does not mention the 50 cap", text)
	}
}

func TestListArticlesEditableFlag(t *testing.T) {
	fixture := newToolFixture(t)
	fixture.createArticle(t, "我的文章", 7)
	fixture.createArticle(t, "站长的文章", 0)
	fixture.createArticle(t, "别的 Agent 的", 9)
	tool := fixture.tool(t, "list_articles")
	_, data, _ := callTool(t, tool, identity(7, scopeContentRead), nil)
	byTitle := map[string]map[string]any{}
	for _, item := range data["articles"].([]any) {
		a := item.(map[string]any)
		byTitle[a["title"].(string)] = a
	}
	if byTitle["我的文章"]["editable"] != true {
		t.Fatalf("own article editable = %v, want true", byTitle["我的文章"]["editable"])
	}
	if byTitle["站长的文章"]["editable"] != false {
		t.Fatalf("owner article editable = %v, want false", byTitle["站长的文章"]["editable"])
	}
	if byTitle["别的 Agent 的"]["editable"] != false {
		t.Fatalf("other agent article editable = %v, want false", byTitle["别的 Agent 的"]["editable"])
	}
	if byTitle["我的文章"]["authorName"] != "写手" || byTitle["我的文章"]["authorType"] != "agent" {
		t.Fatalf("author fields = %v/%v", byTitle["我的文章"]["authorName"], byTitle["我的文章"]["authorType"])
	}
	if _, ok := byTitle["我的文章"]["createdAt"].(string); !ok {
		t.Fatalf("createdAt is not a string: %#v", byTitle["我的文章"]["createdAt"])
	}

	// A zero-value identity uses the same bare key comparison update_article
	// applies, so a zero-key article counts as its own and the hint never
	// disagrees with the write path.
	_, zeroData, _ := callTool(t, tool, identity(0, scopeContentRead), nil)
	zeroByTitle := map[string]map[string]any{}
	for _, item := range zeroData["articles"].([]any) {
		a := item.(map[string]any)
		zeroByTitle[a["title"].(string)] = a
	}
	if zeroByTitle["站长的文章"]["editable"] != true {
		t.Fatalf("zero-key article editable for a zero-key identity = %v, want true", zeroByTitle["站长的文章"]["editable"])
	}
	if zeroByTitle["我的文章"]["editable"] != false {
		t.Fatalf("key-7 article editable for a zero-key identity = %v, want false", zeroByTitle["我的文章"]["editable"])
	}
}

func TestGetArticleReturnsFullContent(t *testing.T) {
	fixture := newToolFixture(t)
	id := fixture.createArticle(t, "可读的文章", 7)
	tool := fixture.tool(t, "get_article")
	_, data, _ := callTool(t, tool, identity(7, scopeContentRead), map[string]any{"id": id})
	if data["title"] != "可读的文章" || data["content"] != "正文 可读的文章" {
		t.Fatalf("get = %#v", data)
	}
	if data["editable"] != true {
		t.Fatalf("editable = %v, want true", data["editable"])
	}
}

func TestGetArticleLockedHidesContent(t *testing.T) {
	fixture := newToolFixture(t)
	id := fixture.createArticle(t, "加密的文章", 0)
	if err := fixture.db.Exec("UPDATE articles SET is_locked = ? WHERE id = ?", true, id).Error; err != nil {
		t.Fatalf("lock article: %v", err)
	}
	tool := fixture.tool(t, "get_article")
	_, data, _ := callTool(t, tool, identity(7, scopeContentRead), map[string]any{"id": id})
	if data["content"] != "" {
		t.Fatalf("locked content = %q, want empty", data["content"])
	}
	if data["isLocked"] != true {
		t.Fatalf("isLocked = %v, want true", data["isLocked"])
	}
	if note, ok := data["contentNote"].(string); !ok || !strings.Contains(note, "已加密") {
		t.Fatalf("locked note = %#v, want an is-locked explanation", data["contentNote"])
	}
}

func TestGetArticleErrors(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "get_article")
	for _, args := range []map[string]any{{"id": 0}, {"id": -3}, {"id": 9999}} {
		result, _, text := callTool(t, tool, identity(7, scopeContentRead), args)
		if !result.IsError {
			t.Fatalf("get %v must fail", args)
		}
		if text == "" {
			t.Fatalf("get %v returned an empty error", args)
		}
	}
}

func TestCreateArticlePersistsAuthorAndReturnsID(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "create_article")
	_, data, _ := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"title": "Agent 第一篇文章", "content": "# 标题\n正文内容",
	})
	id := int(data["id"].(float64))
	if id <= 0 || data["title"] != "Agent 第一篇文章" {
		t.Fatalf("create result = %#v", data)
	}
	stored, err := fixture.articles.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("load created article: %v", err)
	}
	if stored.AuthorType != "agent" {
		t.Fatalf("authorType = %q, want agent", stored.AuthorType)
	}
	if stored.AuthorName != "测试 Agent" {
		t.Fatalf("authorName = %q, want the identity's name", stored.AuthorName)
	}
	if stored.AuthorKeyID != 7 {
		t.Fatalf("authorKeyID = %d, want 7", stored.AuthorKeyID)
	}
	if len(fixture.events.created) != 1 || fixture.events.created[0].id != id {
		t.Fatalf("created events = %#v", fixture.events.created)
	}
}

func TestCreateArticleNilEventsFallsBackToNoop(t *testing.T) {
	fixture := newToolFixtureWithEvents(t, nil)
	tool := fixture.tool(t, "create_article")
	_, data, _ := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"title": "无事件回退", "content": "正文",
	})
	id := int(data["id"].(float64))
	if id <= 0 || data["title"] != "无事件回退" {
		t.Fatalf("create with nil events = %#v", data)
	}
	stored, err := fixture.articles.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("load created article: %v", err)
	}
	if stored.AuthorKeyID != 7 {
		t.Fatalf("authorKeyID = %d, want 7", stored.AuthorKeyID)
	}
}

func TestCreateArticleSubmitsTasksWhenSummaryMissing(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "create_article")
	_, data, _ := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"title": "无摘要", "content": "正文",
	})
	id := int(data["id"].(float64))
	if len(fixture.tasks.tagIDs) != 1 || fixture.tasks.tagIDs[0] != id {
		t.Fatalf("tag generation submissions = %#v", fixture.tasks.tagIDs)
	}
	if len(fixture.tasks.summaryIDs) != 1 || fixture.tasks.summaryIDs[0] != id {
		t.Fatalf("summary generation submissions = %#v", fixture.tasks.summaryIDs)
	}
}

func TestCreateArticleSkipsTasksWhenSummaryGiven(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "create_article")
	callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"title": "有摘要", "content": "正文", "summary": "现成的摘要",
	})
	if len(fixture.tasks.tagIDs) != 0 || len(fixture.tasks.summaryIDs) != 0 {
		t.Fatalf("tasks submitted despite summary: tags=%v summaries=%v", fixture.tasks.tagIDs, fixture.tasks.summaryIDs)
	}
}

func TestCreateArticleCategoryMissingListsAvailable(t *testing.T) {
	fixture := newToolFixture(t)
	if err := fixture.db.Create(&article.Category{Name: "前端", Slug: "front-end"}).Error; err != nil {
		t.Fatalf("seed category: %v", err)
	}
	tool := fixture.tool(t, "create_article")
	result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"title": "分类错误", "content": "正文", "category": "不存在的分类",
	})
	if !result.IsError {
		t.Fatal("bogus category must fail")
	}
	if !strings.Contains(text, "不存在的分类") || !strings.Contains(text, "前端") {
		t.Fatalf("error %q does not name the category and the available list", text)
	}
}

func TestCreateArticleDuplicateTitleHintsUpdate(t *testing.T) {
	fixture := newToolFixture(t)
	fixture.createArticle(t, "重复标题", 0)
	tool := fixture.tool(t, "create_article")
	result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"title": "重复标题", "content": "另一篇正文",
	})
	if !result.IsError {
		t.Fatal("duplicate title must fail")
	}
	if !strings.Contains(text, "update_article") {
		t.Fatalf("duplicate error %q does not point at update_article", text)
	}
}

func TestCreateArticleRejectsEmptyTitleOrContent(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "create_article")
	for _, args := range []map[string]any{
		{"title": "", "content": "正文"},
		{"title": "  ", "content": "正文"},
		{"title": "有标题", "content": "  "},
	} {
		result, _, text := callTool(t, tool, identity(7, scopeContentWrite), args)
		if !result.IsError {
			t.Fatalf("create %v must fail", args)
		}
		if !strings.Contains(text, "title") && !strings.Contains(text, "content") {
			t.Fatalf("error %q does not name the offending field", text)
		}
	}
}

func TestUpdateOwnArticleWithoutToken(t *testing.T) {
	fixture := newToolFixture(t)
	id := fixture.createArticle(t, "自己的文章", 7)
	tool := fixture.tool(t, "update_article")
	_, data, _ := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"id": id, "title": "改名后的标题",
	})
	if data["id"].(float64) != float64(id) || data["updated"] != true {
		t.Fatalf("update result = %#v", data)
	}
	stored, err := fixture.articles.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Title != "改名后的标题" {
		t.Fatalf("title = %q", stored.Title)
	}
	if len(fixture.events.updated) != 1 || fixture.events.updated[0].viaGrant {
		t.Fatalf("updated events = %#v, want one non-grant event", fixture.events.updated)
	}
}

func TestUpdateForeignArticleWithoutTokenDenied(t *testing.T) {
	fixture := newToolFixture(t)
	id := fixture.createArticle(t, "站长的文章", 0)
	tool := fixture.tool(t, "update_article")
	result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"id": id, "title": "改站长的",
	})
	if !result.IsError {
		t.Fatal("updating the owner's article without a token must fail")
	}
	if !strings.Contains(text, "临时授权") || !strings.Contains(text, "edit_token") {
		t.Fatalf("denial %q does not ask for a temporary grant / edit_token", text)
	}
	if len(fixture.events.denied) != 1 {
		t.Fatalf("denied events = %#v, want one", fixture.events.denied)
	}
	if fixture.events.denied[0].id != id || !strings.Contains(fixture.events.denied[0].reason, "临时授权") {
		t.Fatalf("denied event = %#v", fixture.events.denied)
	}
	stored, err := fixture.articles.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Title != "站长的文章" {
		t.Fatalf("article was changed despite denial: %q", stored.Title)
	}
}

func TestUpdateForeignArticleWithValidToken(t *testing.T) {
	fixture := newToolFixture(t)
	id := fixture.createArticle(t, "站长的文章", 0)
	grant, err := fixture.grants.Grant(id, "允许改这一篇")
	if err != nil {
		t.Fatalf("issue grant: %v", err)
	}
	tool := fixture.tool(t, "update_article")
	_, data, _ := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"id": id, "edit_token": grant.TokenPlain, "content": "授权后的正文",
	})
	if data["updated"] != true {
		t.Fatalf("update result = %#v", data)
	}
	stored, err := fixture.articles.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Content != "授权后的正文" {
		t.Fatalf("content = %q", stored.Content)
	}
	reloaded := loadGrant(t, fixture.db, grant.ID)
	if reloaded.UsedCount != 1 {
		t.Fatalf("UsedCount = %d, want 1", reloaded.UsedCount)
	}
	if len(fixture.events.updated) != 1 || !fixture.events.updated[0].viaGrant {
		t.Fatalf("updated events = %#v, want one viaGrant", fixture.events.updated)
	}
}

func TestUpdateForeignArticleTokenFailuresAreDistinguishable(t *testing.T) {
	fixture := newToolFixture(t)
	id := fixture.createArticle(t, "站长的文章", 0)
	otherID := fixture.createArticle(t, "另一篇", 0)

	grant, err := fixture.grants.Grant(id, "有效授权")
	if err != nil {
		t.Fatalf("issue grant: %v", err)
	}
	revoked, err := fixture.grants.Grant(id, "待吊销")
	if err != nil {
		t.Fatalf("issue revoke-target: %v", err)
	}
	if err := fixture.grants.Revoke(revoked.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	expired, err := fixture.grants.Grant(id, "待过期")
	if err != nil {
		t.Fatalf("issue expire-target: %v", err)
	}
	tool := fixture.tool(t, "update_article")

	cases := []struct {
		name      string
		articleID int
		token     string
		wantHint  string
	}{
		{"invalid token", id, "ag_grant_totally_fake", "授权无效或不存在"},
		{"revoked", id, revoked.TokenPlain, "已被吊销"},
		{"bound to another article", otherID, grant.TokenPlain, "只允许修改指定的文章"},
	}
	for _, tc := range cases {
		result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
			"id": tc.articleID, "edit_token": tc.token,
		})
		if !result.IsError {
			t.Fatalf("%s must fail", tc.name)
		}
		if !strings.Contains(text, tc.wantHint) {
			t.Errorf("%s: error %q does not contain %q", tc.name, text, tc.wantHint)
		}
	}
	// Expired is asserted last: it advances the shared clock past the grant's
	// one-hour window, which would poison every earlier assertion.
	fixture.grants.now = func() time.Time { return fixedNow().Add(2 * time.Hour) }
	result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"id": id, "edit_token": expired.TokenPlain,
	})
	if !result.IsError {
		t.Fatal("expired token must fail")
	}
	if !strings.Contains(text, "已过期") {
		t.Errorf("expired: error %q does not contain %q", text, "已过期")
	}
	if len(fixture.events.denied) != 4 {
		t.Fatalf("denied events = %d, want 4 (four distinguishable failures)", len(fixture.events.denied))
	}
}

func TestUpdateOnlyPassedFields(t *testing.T) {
	fixture := newToolFixture(t)
	id := fixture.createArticle(t, "原标题", 7)
	tool := fixture.tool(t, "update_article")
	callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"id": id, "title": "只改标题",
	})
	stored, err := fixture.articles.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Title != "只改标题" {
		t.Fatalf("title = %q", stored.Title)
	}
	if stored.Content != "正文 原标题" {
		t.Fatalf("content changed unexpectedly: %q", stored.Content)
	}
}

func TestUpdateArticleNotFound(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "update_article")
	result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{"id": 9999, "title": "x"})
	if !result.IsError || text == "" {
		t.Fatalf("update of a missing article: result=%#v text=%q", result, text)
	}
}

func TestUploadImageSuccess(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "upload_image")
	_, data, _ := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"file_name": "配图.png", "data": tinyPNG,
	})
	if data["url"] != fixture.images.url {
		t.Fatalf("url = %v, want %v", data["url"], fixture.images.url)
	}
	if fixture.images.savedName != "配图.png" {
		t.Fatalf("saved name = %q", fixture.images.savedName)
	}
	decoded, err := base64.StdEncoding.DecodeString(tinyPNG)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if string(fixture.images.savedData) != string(decoded) {
		t.Fatal("saved data differs from the decoded payload")
	}
}

func TestUploadImageRejectsBadBase64(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "upload_image")
	result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"file_name": "a.png", "data": "这不是 base64!!",
	})
	if !result.IsError || !strings.Contains(text, "base64") {
		t.Fatalf("bad base64: result=%#v text=%q", result, text)
	}
}

func TestUploadImageRejectsOversize(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "upload_image")
	oversize := make([]byte, 5<<20+1)
	result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"file_name": "big.png", "data": base64.StdEncoding.EncodeToString(oversize),
	})
	if !result.IsError || !strings.Contains(text, "5MB") {
		t.Fatalf("oversize: result=%#v text=%q", result, text)
	}
	if fixture.images.calls != 0 {
		t.Fatal("oversize payload reached the image store")
	}
}

func TestUploadImageRejectsRawBase64OverCeiling(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "upload_image")
	// A payload far above the 5MB ceiling: its raw base64 text exceeds
	// maxUploadBase64Len, so the tool rejects it before even decoding.
	oversize := make([]byte, 10<<20)
	result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"file_name": "huge.png", "data": base64.StdEncoding.EncodeToString(oversize),
	})
	if !result.IsError || !strings.Contains(text, "5MB") {
		t.Fatalf("huge payload: result=%#v text=%q", result, text)
	}
	if fixture.images.calls != 0 {
		t.Fatal("huge payload reached the image store")
	}
}

func TestUploadImageRejectsNonImage(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "upload_image")
	result, _, text := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"file_name": "note.txt", "data": base64.StdEncoding.EncodeToString([]byte("hello world")),
	})
	if !result.IsError || !strings.Contains(text, "image") {
		t.Fatalf("text payload: result=%#v text=%q", result, text)
	}
	if fixture.images.calls != 0 {
		t.Fatal("non-image payload reached the image store")
	}
}

func TestUploadImageSanitizesPath(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "upload_image")
	for _, name := range []string{"../evil.png", "a/b/c.png", ".hidden"} {
		callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
			"file_name": name, "data": tinyPNG,
		})
		if strings.Contains(fixture.images.savedName, "/") || strings.Contains(fixture.images.savedName, "..") {
			t.Fatalf("saved name %q still contains path separators", fixture.images.savedName)
		}
		if fixture.images.savedName == "" {
			t.Fatalf("saved name empty for input %q", name)
		}
	}
}

func TestUploadImageRejectsEmptyFileName(t *testing.T) {
	fixture := newToolFixture(t)
	tool := fixture.tool(t, "upload_image")
	result, _, _ := callTool(t, tool, identity(7, scopeContentWrite), map[string]any{
		"file_name": "  /  ", "data": tinyPNG,
	})
	if !result.IsError {
		t.Fatal("empty file name must fail")
	}
	if fixture.images.calls != 0 {
		t.Fatal("empty file name reached the image store")
	}
}
