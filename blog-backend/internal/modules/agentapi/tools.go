package agentapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"dh-blog/internal/model"
	"dh-blog/internal/modules/article"
	"dh-blog/internal/platform/mcp"

	"github.com/sirupsen/logrus"
)

// Tool names. No prefix: MCP clients namespace them under the server name
// automatically, so a "write_" prefix would just be noise the model has to
// type twice.
const (
	toolListArticles   = "list_articles"
	toolGetArticle     = "get_article"
	toolCreateArticle  = "create_article"
	toolUpdateArticle  = "update_article"
	toolUploadImage    = "upload_image"
	authorTypeAgent    = "agent"
	maxUploadImageSize = 5 << 20
	// listArticleMaxPageSize caps how many rows one listing can ask for. 50
	// keeps a page cheap while still covering a whole writing session.
	listArticleMaxPageSize = 50
)

// requireIdentity pulls the caller out of the context and verifies it holds
// the scope the tool needs. The scope check is the in-tool backstop: tools/list
// already hides the tool from keys without the scope, but a direct tools/call
// must not bypass the same rule.
func requireIdentity(ctx context.Context, scope string) (Identity, string) {
	identity, ok := IdentityFrom(ctx)
	if !ok {
		return nil, "缺少调用者身份"
	}
	if !identity.HasScope(scope) {
		return nil, "当前凭证缺少 " + scope + " 权限，无法执行该操作"
	}
	return identity, ""
}

// editableOf reports whether the given author key can rewrite the article
// without a temporary grant: only the key that created it.
func editableOf(authorKeyID, keyID int) bool {
	return authorKeyID != 0 && authorKeyID == keyID
}

// timeText renders an article's created time in the repo's canonical format.
// JSONTime embeds time.Time, so Format works on the value directly; the helper
// exists only to keep the format string in one place.
func timeText(t model.JSONTime) string { return t.Format("2006-01-02 15:04:05") }

// --- list_articles ---

type listArticlesArgs struct {
	Keyword  string `json:"keyword"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type listArticleView struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Category   string   `json:"category"`
	Tags       []string `json:"tags"`
	Summary    string   `json:"summary"`
	WordNum    int      `json:"wordNum"`
	CreatedAt  string   `json:"createdAt"`
	AuthorType string   `json:"authorType"`
	AuthorName string   `json:"authorName"`
	Editable   bool     `json:"editable"`
}

type listArticlesResult struct {
	Articles []listArticleView `json:"articles"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// listArticlesTool advertises what the agent can read so it can decide whether
// to edit (own article) or ask the owner for a grant (anyone else's).
type listArticlesTool struct {
	articles Articles
}

func (t *listArticlesTool) Name() string  { return toolListArticles }
func (t *listArticlesTool) Scope() string { return scopeContentRead }
func (t *listArticlesTool) Definition(context.Context) mcp.Definition {
	return mcp.Definition{
		Name:        toolListArticles,
		Title:       "列出文章",
		Description: "按关键词和分页列出博客文章，返回标题、分类、标签、摘要、字数、创建时间、作者，以及 editable 字段（本凭证能否免授权修改该篇）。editable 为 false 的文章需要站长签发临时授权才能修改。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"keyword": map[string]any{"type": "string", "description": "标题或正文的关键词，可选。"},
				"page":    map[string]any{"type": "integer", "description": "页码，从 1 开始，默认 1。"},
				"page_size": map[string]any{
					"type": "integer", "description": "每页条数，默认 20，最大 50。",
				},
			},
		},
	}
}

func (t *listArticlesTool) Call(ctx context.Context, raw json.RawMessage) mcp.Result {
	identity, errText := requireIdentity(ctx, t.Scope())
	if errText != "" {
		return mcp.ToolError(errText)
	}
	var args listArticlesArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return mcp.ToolError("arguments 解析失败: " + err.Error())
		}
	}
	page, pageSize := args.Page, args.PageSize
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if page < 1 {
		return mcp.ToolError("page 必须大于等于 1")
	}
	if pageSize < 1 {
		return mcp.ToolError("page_size 必须大于等于 1")
	}
	if pageSize > listArticleMaxPageSize {
		return mcp.ToolError(fmt.Sprintf("page_size 不能超过 %d", listArticleMaxPageSize))
	}

	briefs, total, err := t.articles.List(ctx, args.Keyword, page, pageSize)
	if err != nil {
		return mcp.ToolError("查询文章列表失败: " + err.Error())
	}
	views := make([]listArticleView, 0, len(briefs))
	for _, brief := range briefs {
		views = append(views, listArticleView{
			ID:         brief.ID,
			Title:      brief.Title,
			Category:   brief.CategoryName,
			Tags:       brief.Tags,
			Summary:    brief.Summary,
			WordNum:    brief.WordNum,
			CreatedAt:  timeText(brief.CreatedAt),
			AuthorType: brief.AuthorType,
			AuthorName: brief.AuthorName,
			Editable:   editableOf(brief.AuthorKeyID, identity.KeyID()),
		})
	}
	return textResult(listArticlesResult{Articles: views, Total: total, Page: page, PageSize: pageSize})
}

// --- get_article ---

type getArticleArgs struct {
	ID int `json:"id"`
}

type getArticleResult struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	ContentNote string   `json:"contentNote,omitempty"`
	Summary     string   `json:"summary"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	WordNum     int      `json:"wordNum"`
	Thumbnail   string   `json:"thumbnailUrl"`
	IsLocked    bool     `json:"isLocked"`
	CreatedAt   string   `json:"createdAt"`
	AuthorType  string   `json:"authorType"`
	AuthorName  string   `json:"authorName"`
	Editable    bool     `json:"editable"`
}

// getArticleTool hands an agent the full Markdown of one article, minus the
// body of locked ones.
type getArticleTool struct {
	articles Articles
}

func (t *getArticleTool) Name() string  { return toolGetArticle }
func (t *getArticleTool) Scope() string { return scopeContentRead }
func (t *getArticleTool) Definition(context.Context) mcp.Definition {
	return mcp.Definition{
		Name:        toolGetArticle,
		Title:       "读取文章",
		Description: "按 id 读取一篇文章的完整 Markdown 正文与元信息。加密文章不返回正文。editable 表示本凭证能否免授权修改该篇。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "integer", "description": "文章 id。"},
			},
			"required": []string{"id"},
		},
	}
}

func (t *getArticleTool) Call(ctx context.Context, raw json.RawMessage) mcp.Result {
	identity, errText := requireIdentity(ctx, t.Scope())
	if errText != "" {
		return mcp.ToolError(errText)
	}
	var args getArticleArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return mcp.ToolError("arguments 解析失败: " + err.Error())
	}
	if args.ID <= 0 {
		return mcp.ToolError("id 必须大于 0")
	}
	detail, err := t.articles.Get(ctx, args.ID)
	if err != nil {
		return mcp.ToolError("读取文章失败: " + err.Error())
	}
	result := getArticleResult{
		ID:         detail.ID,
		Title:      detail.Title,
		Content:    detail.Content,
		Summary:    detail.Summary,
		Category:   detail.CategoryName,
		Tags:       detail.Tags,
		WordNum:    detail.WordNum,
		Thumbnail:  detail.ThumbnailURL,
		IsLocked:   detail.IsLocked,
		CreatedAt:  timeText(detail.CreatedAt),
		AuthorType: detail.AuthorType,
		AuthorName: detail.AuthorName,
		Editable:   editableOf(detail.AuthorKeyID, identity.KeyID()),
	}
	if detail.IsLocked {
		// The body is deliberately absent; the note tells the model why and
		// that unlocking is an owner-side action, not a parameter it can pass.
		result.Content = ""
		result.ContentNote = "该文章已加密，正文不可见"
	}
	return textResult(result)
}

// --- create_article ---

type createArticleArgs struct {
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Summary      string   `json:"summary"`
	Category     string   `json:"category"`
	Tags         []string `json:"tags"`
	ThumbnailURL string   `json:"thumbnail_url"`
}

type createArticleResult struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// createArticleTool publishes a new article signed with the caller's identity.
type createArticleTool struct {
	articles Articles
	tasks    TaskSubmitter
	events   ContentReporter
}

func (t *createArticleTool) Name() string  { return toolCreateArticle }
func (t *createArticleTool) Scope() string { return scopeContentWrite }
func (t *createArticleTool) Definition(context.Context) mcp.Definition {
	return mcp.Definition{
		Name:        toolCreateArticle,
		Title:       "创建文章",
		Description: "创建一篇新文章并直接发布，正文用 Markdown。作者署名自动使用当前凭证。分类不存在时会报错并列出可用分类；标题已存在时会报错，请改用 update_article。未传 summary 时会后台自动生成摘要与标签。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":         map[string]any{"type": "string", "description": "文章标题，必填。"},
				"content":       map[string]any{"type": "string", "description": "Markdown 正文，必填。"},
				"summary":       map[string]any{"type": "string", "description": "摘要，可选；不传则后台自动生成。"},
				"category":      map[string]any{"type": "string", "description": "分类名，可选；不存在时报错并列出可用分类。"},
				"tags":          map[string]any{"type": "array", "description": "标签名称数组，可选。", "items": map[string]any{"type": "string"}},
				"thumbnail_url": map[string]any{"type": "string", "description": "缩略图 URL，可选；可用 upload_image 生成的链接。"},
			},
			"required": []string{"title", "content"},
		},
	}
}

func (t *createArticleTool) Call(ctx context.Context, raw json.RawMessage) mcp.Result {
	identity, errText := requireIdentity(ctx, t.Scope())
	if errText != "" {
		return mcp.ToolError(errText)
	}
	var args createArticleArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return mcp.ToolError("arguments 解析失败: " + err.Error())
	}
	if strings.TrimSpace(args.Title) == "" {
		return mcp.ToolError("title 不能为空")
	}
	if strings.TrimSpace(args.Content) == "" {
		return mcp.ToolError("content 不能为空")
	}

	id, err := t.articles.Create(ctx, article.CreateInput{
		Title:        args.Title,
		Content:      args.Content,
		Summary:      args.Summary,
		CategoryName: args.Category,
		Tags:         args.Tags,
		ThumbnailURL: args.ThumbnailURL,
		AuthorType:   authorTypeAgent,
		AuthorName:   identity.AuthorName(),
		AuthorKeyID:  identity.KeyID(),
	})
	if err != nil {
		// The article service already appends the usable category list and the
		// update_article hint to its errors; pass them through verbatim so the
		// model gets the full correction text.
		return mcp.ToolError(err.Error())
	}

	if strings.TrimSpace(args.Summary) == "" {
		t.tasks.SubmitTagGeneration(id, args.Content)
		t.tasks.SubmitSummaryGeneration(id, args.Content)
	}
	t.events.ArticleCreated(identity.AuthorName(), args.Title, id)
	return textResult(createArticleResult{ID: id, Title: args.Title})
}

// --- update_article ---

type updateArticleArgs struct {
	ID        int       `json:"id"`
	EditToken string    `json:"edit_token"`
	Title     *string   `json:"title"`
	Content   *string   `json:"content"`
	Summary   *string   `json:"summary"`
	Category  *string   `json:"category"`
	Tags      *[]string `json:"tags"`
}

type updateArticleResult struct {
	ID      int  `json:"id"`
	Updated bool `json:"updated"`
}

// updateArticleTool rewrites articles under the authorization rule from the
// design doc: own articles for free, everyone else's with a temporary grant.
type updateArticleTool struct {
	articles Articles
	grants   GrantService
	events   ContentReporter
}

func (t *updateArticleTool) Name() string  { return toolUpdateArticle }
func (t *updateArticleTool) Scope() string { return scopeContentWrite }
func (t *updateArticleTool) Definition(context.Context) mcp.Definition {
	return mcp.Definition{
		Name:        toolUpdateArticle,
		Title:       "更新文章",
		Description: "按 id 更新文章，只更新传入的字段。edit_token 只有改非本 Agent 创建的文章时才需要，由站长从后台签发，有效期 1 小时；改自己创建的文章不需要。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":         map[string]any{"type": "integer", "description": "文章 id，必填。"},
				"edit_token": map[string]any{"type": "string", "description": "修改非本 Agent 创建的文章所需的临时授权 Token，由站长从后台「文章管理 → 生成 AI 修改授权」签发，有效期 1 小时。"},
				"title":      map[string]any{"type": "string", "description": "新标题，可选。"},
				"content":    map[string]any{"type": "string", "description": "新 Markdown 正文，可选。"},
				"summary":    map[string]any{"type": "string", "description": "新摘要，可选。"},
				"category":   map[string]any{"type": "string", "description": "新分类名，可选；不存在时报错。"},
				"tags":       map[string]any{"type": "array", "description": "新的标签名称数组，可选。", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"id"},
		},
	}
}

// grantDenialText maps a grant validation failure onto a reason the model can
// act on (or relay to the owner). The messages stay distinguishable so the
// agent can tell "ask for a new one" from "this one is dead".
func grantDenialText(err error) string {
	switch {
	case errors.Is(err, ErrGrantNotFound):
		return "授权无效或不存在，请重新向站长索取"
	case errors.Is(err, ErrGrantExpired):
		return "授权已过期（有效期 1 小时），请重新签发"
	case errors.Is(err, ErrGrantRevoked):
		return "授权已被吊销"
	case errors.Is(err, ErrGrantWrongArticle):
		return "该授权只允许修改指定的文章"
	default:
		logrus.Errorf("校验临时授权失败: %v", err)
		return "校验授权失败，请稍后重试"
	}
}

func (t *updateArticleTool) Call(ctx context.Context, raw json.RawMessage) mcp.Result {
	identity, errText := requireIdentity(ctx, t.Scope())
	if errText != "" {
		return mcp.ToolError(errText)
	}
	var args updateArticleArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return mcp.ToolError("arguments 解析失败: " + err.Error())
	}
	if args.ID <= 0 {
		return mcp.ToolError("id 必须大于 0")
	}

	detail, err := t.articles.Get(ctx, args.ID)
	if err != nil {
		return mcp.ToolError("读取文章失败: " + err.Error())
	}

	// Authorization rule, in order: own article passes for free; everything
	// else needs a grant token, and each failure mode says which one.
	viaGrant := false
	agent := identity.AuthorName()
	if detail.AuthorKeyID != identity.KeyID() {
		if strings.TrimSpace(args.EditToken) == "" {
			denial := "这篇文章不是本 Agent 创建的，修改需要临时授权 Token。请让站长在后台「文章管理 → 生成 AI 修改授权」签发一个（有效期 1 小时），并把它作为 edit_token 参数传入"
			t.events.ArticleUpdateDenied(agent, detail.Title, args.ID, denial)
			return mcp.ToolError(denial)
		}
		if _, err := t.grants.Validate(args.ID, args.EditToken); err != nil {
			denial := grantDenialText(err)
			t.events.ArticleUpdateDenied(agent, detail.Title, args.ID, denial)
			return mcp.ToolError(denial)
		}
		viaGrant = true
	}

	input := article.UpdateInput{ID: args.ID}
	if args.Title != nil {
		input.Title = args.Title
	}
	if args.Content != nil {
		input.Content = args.Content
	}
	if args.Summary != nil {
		input.Summary = args.Summary
	}
	if args.Category != nil {
		input.CategoryName = args.Category
	}
	if args.Tags != nil {
		input.Tags = args.Tags
	}
	if err := t.articles.Update(ctx, input); err != nil {
		return mcp.ToolError("更新文章失败: " + err.Error())
	}
	t.events.ArticleUpdated(agent, detail.Title, args.ID, viaGrant)
	return textResult(updateArticleResult{ID: args.ID, Updated: true})
}

// --- upload_image ---

type uploadImageArgs struct {
	FileName string `json:"file_name"`
	Data     string `json:"data"`
}

type uploadImageResult struct {
	URL string `json:"url"`
}

// uploadImageTool accepts base64-encoded images only — never a source_url —
// so a prompt-injected fetch of an attacker URL can never make the server open
// it. Size and MIME limits keep the upload path honest.
type uploadImageTool struct {
	images Images
}

func (t *uploadImageTool) Name() string  { return toolUploadImage }
func (t *uploadImageTool) Scope() string { return scopeContentWrite }
func (t *uploadImageTool) Definition(context.Context) mcp.Definition {
	return mcp.Definition{
		Name:        toolUploadImage,
		Title:       "上传图片",
		Description: "上传一张图片，返回可直接写进 Markdown 的 URL。data 为 base64 编码的图片内容，解码后不超过 5MB，只接受图片类型。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file_name": map[string]any{"type": "string", "description": "文件名，必填，用于命名存储的文件。"},
				"data":      map[string]any{"type": "string", "description": "图片内容的 base64 编码，必填，解码后不超过 5MB。"},
			},
			"required": []string{"file_name", "data"},
		},
	}
}

func (t *uploadImageTool) Call(ctx context.Context, raw json.RawMessage) mcp.Result {
	if _, errText := requireIdentity(ctx, t.Scope()); errText != "" {
		return mcp.ToolError(errText)
	}
	var args uploadImageArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return mcp.ToolError("arguments 解析失败: " + err.Error())
	}
	data, err := base64.StdEncoding.DecodeString(args.Data)
	if err != nil {
		return mcp.ToolError("data 不是合法的 base64: " + err.Error())
	}
	if len(data) > maxUploadImageSize {
		return mcp.ToolError(fmt.Sprintf("图片超过大小限制（解码后 %dMB）", maxUploadImageSize>>20))
	}
	if !strings.HasPrefix(http.DetectContentType(data), "image/") {
		return mcp.ToolError("只接受图片类型（image/*），当前内容不是图片")
	}
	name := filepath.Base(strings.TrimSpace(args.FileName))
	// "." and ".." are directory references, not file names; rejecting them
	// here keeps the tool's own contract airtight even though the files layer
	// re-checks traversal on its side too.
	if name == "" || name == "." || name == ".." || name == "/" {
		return mcp.ToolError("file_name 无效")
	}
	url, err := t.images.SaveBlogImage(ctx, name, data)
	if err != nil {
		return mcp.ToolError("保存图片失败: " + err.Error())
	}
	return textResult(uploadImageResult{URL: url})
}

// textResult renders a successful structured answer for the model to parse.
func textResult(payload any) mcp.Result {
	encoded, err := json.Marshal(payload)
	if err != nil {
		logrus.Errorf("序列化工具结果失败: %v", err)
		return mcp.ToolError("生成结果失败")
	}
	return mcp.Text(string(encoded))
}
