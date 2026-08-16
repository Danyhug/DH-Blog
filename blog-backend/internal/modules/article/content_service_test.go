package article

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func newTestContentService(t *testing.T) *contentService {
	t.Helper()
	db := openArticleTestDB(t)
	if err := ensureDefaults(db); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	cache := newTestCache()
	tags := NewTagRepository(db, cache)
	categories := NewCategoryRepository(db)
	articles := NewArticleRepository(db, categories, tags, cache)
	return newContentService(articles, db)
}

func TestContentServiceCreatePersistsAuthorAndResolvesCategory(t *testing.T) {
	svc := newTestContentService(t)
	category := Category{Name: "Backend", Slug: "backend"}
	if err := svc.db.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	id, err := svc.Create(context.Background(), CreateInput{
		Title:        "AI 写的文章",
		Content:      "第一段 第二段 第三段 第四段",
		Summary:      "摘要",
		CategoryName: "Backend",
		Tags:         []string{"go", "agent"},
		AuthorType:   "agent",
		AuthorName:   "Claude",
		AuthorKeyID:  7,
	})
	if err != nil {
		t.Fatalf("create article: %v", err)
	}
	if id <= 0 {
		t.Fatalf("created id = %d, want > 0", id)
	}

	stored, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get created article: %v", err)
	}
	if stored.Title != "AI 写的文章" {
		t.Fatalf("title = %q", stored.Title)
	}
	if stored.CategoryID != category.ID || stored.CategoryName != "Backend" {
		t.Fatalf("category = %d/%q, want %d/Backend", stored.CategoryID, stored.CategoryName, category.ID)
	}
	if len(stored.Tags) != 2 || stored.Tags[0] != "go" || stored.Tags[1] != "agent" {
		t.Fatalf("tags = %#v, want [go agent]", stored.Tags)
	}
	if stored.WordNum != 4 {
		t.Fatalf("word count = %d, want 4", stored.WordNum)
	}
	if stored.AuthorType != "agent" || stored.AuthorName != "Claude" || stored.AuthorKeyID != 7 {
		t.Fatalf("author = %q/%q/%d", stored.AuthorType, stored.AuthorName, stored.AuthorKeyID)
	}
}

func TestContentServiceCreateFindsCategoryBySlug(t *testing.T) {
	svc := newTestContentService(t)
	if err := svc.db.Create(&Category{Name: "前端", Slug: "front-end"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	id, err := svc.Create(context.Background(), CreateInput{
		Title: "按 slug 找分类", Content: "正文", CategoryName: "front-end",
	})
	if err != nil {
		t.Fatalf("create article: %v", err)
	}
	stored, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get article: %v", err)
	}
	if stored.CategoryName != "前端" {
		t.Fatalf("category name = %q, want 前端", stored.CategoryName)
	}
}

func TestContentServiceCreateValidatesTitleAndContent(t *testing.T) {
	svc := newTestContentService(t)
	if _, err := svc.Create(context.Background(), CreateInput{Title: "  ", Content: "正文"}); err == nil {
		t.Fatal("expected an error for an empty title")
	}
	if _, err := svc.Create(context.Background(), CreateInput{Title: "有标题", Content: "   \n\t"}); err == nil {
		t.Fatal("expected an error for an empty content")
	}
}

func TestContentServiceCreateRejectsUnknownCategoryWithAvailableList(t *testing.T) {
	svc := newTestContentService(t)
	for _, name := range []string{"Zed", "Backend", "Alpha"} {
		if err := svc.db.Create(&Category{Name: name, Slug: strings.ToLower(name)}).Error; err != nil {
			t.Fatalf("create category %s: %v", name, err)
		}
	}

	_, err := svc.Create(context.Background(), CreateInput{Title: "工具", Content: "正文", CategoryName: "Nope"})
	if err == nil {
		t.Fatal("expected an error for an unknown category")
	}
	msg := err.Error()
	for _, want := range []string{"分类不存在", "Nope", "Alpha", "Backend", "Zed"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not mention %q", msg, want)
		}
	}
	iAlpha, iBackend, iZed := strings.Index(msg, "Alpha"), strings.Index(msg, "Backend"), strings.Index(msg, "Zed")
	if iAlpha >= iBackend || iBackend >= iZed {
		t.Fatalf("available categories are not sorted by name in %q", msg)
	}
}

func TestContentServiceCreateRejectsDuplicateTitle(t *testing.T) {
	svc := newTestContentService(t)

	title := "重名文章"
	if _, err := svc.Create(context.Background(), CreateInput{Title: title, Content: "第一版"}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := svc.Create(context.Background(), CreateInput{Title: title, Content: "第二版"}); err == nil {
		t.Fatal("expected a duplicate-title error")
	} else if !strings.Contains(err.Error(), title) || strings.Contains(err.Error(), "update_article") == false {
		t.Fatalf("duplicate error %q should name the title and suggest update_article", err.Error())
	}

	var total int64
	if err := svc.db.Model(&Article{}).Count(&total).Error; err != nil {
		t.Fatalf("count articles: %v", err)
	}
	if total != 1 {
		t.Fatalf("article count = %d, want 1 (no silent duplicate)", total)
	}
}

func TestContentServiceCreateDuplicateScopeSkipsSoftDeleted(t *testing.T) {
	svc := newTestContentService(t)

	title := "删过还能再建"
	first, err := svc.Create(context.Background(), CreateInput{Title: title, Content: "正文"})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := svc.repository.Delete(context.Background(), first); err != nil {
		t.Fatalf("soft delete article: %v", err)
	}
	if _, err := svc.Create(context.Background(), CreateInput{Title: title, Content: "重建"}); err != nil {
		t.Fatalf("create after soft delete: %v", err)
	}
}

func TestContentServiceGetReturnsFullDetail(t *testing.T) {
	svc := newTestContentService(t)
	category := Category{Name: "Backend", Slug: "backend"}
	if err := svc.db.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	id, err := svc.Create(context.Background(), CreateInput{
		Title: "读取测试", Content: "# 标题\n正文", Summary: "预览",
		CategoryName: "Backend", Tags: []string{"go"}, AuthorType: "agent", AuthorName: "GPT", AuthorKeyID: 9,
	})
	if err != nil {
		t.Fatalf("create article: %v", err)
	}

	detail, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get article: %v", err)
	}
	if detail.Content != "# 标题\n正文" || detail.Summary != "预览" {
		t.Fatalf("content/summary = %q/%q", detail.Content, detail.Summary)
	}
	if detail.CategoryName != "Backend" || len(detail.Tags) != 1 || detail.Tags[0] != "go" {
		t.Fatalf("category/tags = %q/%#v", detail.CategoryName, detail.Tags)
	}
	if detail.AuthorType != "agent" || detail.AuthorName != "GPT" || detail.AuthorKeyID != 9 {
		t.Fatalf("author = %q/%q/%d", detail.AuthorType, detail.AuthorName, detail.AuthorKeyID)
	}
	if detail.CreatedAt.IsZero() {
		t.Fatal("createdAt is zero")
	}
}

func TestContentServiceGetRejectsUnknownID(t *testing.T) {
	svc := newTestContentService(t)
	if _, err := svc.Get(context.Background(), 4242); err == nil {
		t.Fatal("expected an error for an unknown id")
	} else if !errors.Is(err, ErrArticleNotFound) {
		t.Fatalf("error = %v, want ErrArticleNotFound", err)
	}
}

func TestContentServiceUpdateOnlyTouchesProvidedFields(t *testing.T) {
	svc := newTestContentService(t)

	id, err := svc.Create(context.Background(), CreateInput{
		Title: "原标题", Content: "原正文 原正文", Summary: "原摘要", Tags: []string{"old"},
	})
	if err != nil {
		t.Fatalf("create article: %v", err)
	}

	title := "改标题"
	if err := svc.Update(context.Background(), UpdateInput{ID: id, Title: &title}); err != nil {
		t.Fatalf("update title: %v", err)
	}

	stored, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get article: %v", err)
	}
	if stored.Title != "改标题" {
		t.Fatalf("title = %q, want 改标题", stored.Title)
	}
	if stored.Content != "原正文 原正文" || stored.Summary != "原摘要" {
		t.Fatalf("content/summary changed without being provided: %q/%q", stored.Content, stored.Summary)
	}
	if len(stored.Tags) != 1 || stored.Tags[0] != "old" {
		t.Fatalf("tags changed without being provided: %#v", stored.Tags)
	}
}

func TestContentServiceUpdateReplacesContentAndTags(t *testing.T) {
	svc := newTestContentService(t)

	id, err := svc.Create(context.Background(), CreateInput{
		Title: "标题", Content: "一 二 三", Summary: "摘要", Tags: []string{"old"},
		AuthorType: "agent", AuthorName: "Claude", AuthorKeyID: 3,
	})
	if err != nil {
		t.Fatalf("create article: %v", err)
	}

	content := "新的 正文 四 五 六"
	tags := []string{"new"}
	if err := svc.Update(context.Background(), UpdateInput{ID: id, Content: &content, Tags: &tags}); err != nil {
		t.Fatalf("update content/tags: %v", err)
	}

	stored, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get article: %v", err)
	}
	if stored.Content != content || stored.WordNum != 5 {
		t.Fatalf("content = %q (words %d), want %q with 5 words", stored.Content, stored.WordNum, content)
	}
	if len(stored.Tags) != 1 || stored.Tags[0] != "new" {
		t.Fatalf("tags = %#v, want [new]", stored.Tags)
	}
	if stored.AuthorType != "agent" || stored.AuthorName != "Claude" || stored.AuthorKeyID != 3 {
		t.Fatalf("author changed during update: %q/%q/%d", stored.AuthorType, stored.AuthorName, stored.AuthorKeyID)
	}
}

func TestContentServiceUpdateChangesCategoryByName(t *testing.T) {
	svc := newTestContentService(t)
	if err := svc.db.Create(&Category{Name: "From", Slug: "from"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	to := Category{Name: "To", Slug: "to"}
	if err := svc.db.Create(&to).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	var defaultCategory Category
	if err := svc.db.Where("slug = ?", "default").First(&defaultCategory).Error; err != nil {
		t.Fatalf("load default category: %v", err)
	}

	id, err := svc.Create(context.Background(), CreateInput{Title: "分类变更", Content: "正文", CategoryName: "From"})
	if err != nil {
		t.Fatalf("create article: %v", err)
	}

	categoryName := "To"
	if err := svc.Update(context.Background(), UpdateInput{ID: id, CategoryName: &categoryName}); err != nil {
		t.Fatalf("update category: %v", err)
	}

	stored, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get article: %v", err)
	}
	if stored.CategoryID != to.ID || stored.CategoryName != "To" {
		t.Fatalf("category = %d/%q, want %d/To", stored.CategoryID, stored.CategoryName, to.ID)
	}

	// 再次更新到不存在的分类必须报错而不落库
	bad := "不存在的分类"
	if err := svc.Update(context.Background(), UpdateInput{ID: id, CategoryName: &bad}); err == nil {
		t.Fatal("expected an error for an unknown category")
	}
	var exists Article
	if err := svc.db.First(&exists, id).Error; err != nil {
		t.Fatalf("article should still exist after a failed category update: %v", err)
	}
	if exists.CategoryID == defaultCategory.ID {
		t.Fatalf("article fell back to default category %d after a failed category update", defaultCategory.ID)
	}
}

func TestContentServiceUpdateRejectsUnknownID(t *testing.T) {
	svc := newTestContentService(t)
	title := "任何标题"
	err := svc.Update(context.Background(), UpdateInput{ID: 4242, Title: &title})
	if err == nil {
		t.Fatal("expected an error for an unknown id")
	} else if !errors.Is(err, ErrArticleNotFound) {
		t.Fatalf("error = %v, want ErrArticleNotFound", err)
	}
}

func TestContentServiceListPaginatesAndOrdersDesc(t *testing.T) {
	svc := newTestContentService(t)

	var ids []int
	for i := 0; i < 5; i++ {
		id, err := svc.Create(context.Background(), CreateInput{
			Title: "第" + strings.Repeat("一", i+1) + "篇", Content: "正文内容",
		})
		if err != nil {
			t.Fatalf("create article %d: %v", i, err)
		}
		ids = append(ids, id)
	}

	briefs, total, err := svc.List(context.Background(), "", 1, 2)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if total != 5 {
		t.Fatalf("total = %d, want 5", total)
	}
	if len(briefs) != 2 {
		t.Fatalf("page length = %d, want 2", len(briefs))
	}
	for i, brief := range briefs {
		if brief.ID != ids[len(ids)-1-i] {
			t.Fatalf("page item %d id = %d, want %d (id DESC)", i, brief.ID, ids[len(ids)-1-i])
		}
	}

	pageThree, _, err := svc.List(context.Background(), "", 3, 2)
	if err != nil {
		t.Fatalf("list page 3: %v", err)
	}
	if len(pageThree) != 1 {
		t.Fatalf("page 3 length = %d, want 1", len(pageThree))
	}

	emptyPage, totalAfterSoftDelete, err := svc.List(context.Background(), "", 4, 2)
	if err != nil {
		t.Fatalf("list beyond end: %v", err)
	}
	if len(emptyPage) != 0 || totalAfterSoftDelete != 5 {
		t.Fatalf("empty page = %#v, total = %d, want 0 rows but total 5", emptyPage, totalAfterSoftDelete)
	}

	// 非法分页参数不得 panic
	if _, _, err := svc.List(context.Background(), "", 0, 0); err != nil {
		t.Fatalf("list with invalid page params: %v", err)
	}
}

func TestContentServiceListBatchesTagsAndCategoryNames(t *testing.T) {
	svc := newTestContentService(t)

	categories := []Category{
		{Name: "Backend", Slug: "backend"},
		{Name: "Frontend", Slug: "frontend"},
	}
	if err := svc.db.Create(&categories).Error; err != nil {
		t.Fatalf("create categories: %v", err)
	}

	_, err := svc.Create(context.Background(), CreateInput{
		Title: "后端文", Content: "内容", CategoryName: "Backend", Tags: []string{"go", "gorm"},
		AuthorType: "agent", AuthorName: "Claude", AuthorKeyID: 5,
	})
	if err != nil {
		t.Fatalf("create backend article: %v", err)
	}
	_, err = svc.Create(context.Background(), CreateInput{
		Title: "前端文", Content: "内容", CategoryName: "Frontend", Tags: []string{"vue"},
	})
	if err != nil {
		t.Fatalf("create frontend article: %v", err)
	}

	briefs, total, err := svc.List(context.Background(), "", 1, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	for _, brief := range briefs {
		switch brief.Title {
		case "后端文":
			if brief.CategoryName != "Backend" || len(brief.Tags) != 2 {
				t.Fatalf("backend brief = %#v, want category Backend and tags [go gorm]", brief)
			}
			if brief.AuthorType != "agent" || brief.AuthorName != "Claude" || brief.AuthorKeyID != 5 {
				t.Fatalf("backend author = %q/%q/%d", brief.AuthorType, brief.AuthorName, brief.AuthorKeyID)
			}
		case "前端文":
			if brief.CategoryName != "Frontend" || len(brief.Tags) != 1 || brief.Tags[0] != "vue" {
				t.Fatalf("frontend brief = %#v, want category Frontend and tags [vue]", brief)
			}
		default:
			t.Fatalf("unexpected brief title %q", brief.Title)
		}
	}
}

func TestContentServiceListFiltersByKeyword(t *testing.T) {
	svc := newTestContentService(t)

	_, err := svc.Create(context.Background(), CreateInput{Title: "Go 并发模型", Content: "channel 使用"})
	if err != nil {
		t.Fatalf("create article: %v", err)
	}
	_, err = svc.Create(context.Background(), CreateInput{Title: "Vue 组件", Content: "props 传递"})
	if err != nil {
		t.Fatalf("create article: %v", err)
	}

	briefs, total, err := svc.List(context.Background(), "Vue", 1, 10)
	if err != nil {
		t.Fatalf("list by title keyword: %v", err)
	}
	if total != 1 || len(briefs) != 1 || briefs[0].Title != "Vue 组件" {
		t.Fatalf("title keyword result = %#v (total %d)", briefs, total)
	}

	briefs, total, err = svc.List(context.Background(), "channel", 1, 10)
	if err != nil {
		t.Fatalf("list by content keyword: %v", err)
	}
	if total != 1 || briefs[0].Title != "Go 并发模型" {
		t.Fatalf("content keyword result = %#v (total %d)", briefs, total)
	}

	briefs, total, err = svc.List(context.Background(), "不存在", 1, 10)
	if err != nil {
		t.Fatalf("list by missing keyword: %v", err)
	}
	if total != 0 || len(briefs) != 0 {
		t.Fatalf("missing keyword result = %#v (total %d), want empty", briefs, total)
	}
}
