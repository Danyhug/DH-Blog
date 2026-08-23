package files

import (
	"context"
	"strconv"
	"testing"
)

// seedSearchTree 铺一棵 根/文档/2024/年终报告.docx 的目录树，返回可复用的 service。
func seedSearchTree(t *testing.T) (*fileService, context.Context) {
	t.Helper()
	db := openTestDB(t)
	repository := newRepository(db)
	service := &fileService{repo: repository, filePath: t.TempDir()}
	ctx := context.Background()

	docs := &File{UserID: 1, ParentID: "", Name: "文档", IsFolder: true, StoragePath: "文档"}
	if err := repository.Create(ctx, docs); err != nil {
		t.Fatalf("create 文档: %v", err)
	}
	year := &File{UserID: 1, ParentID: strconv.Itoa(docs.ID), Name: "2024", IsFolder: true, StoragePath: "文档/2024"}
	if err := repository.Create(ctx, year); err != nil {
		t.Fatalf("create 2024: %v", err)
	}

	records := []*File{
		{UserID: 1, ParentID: strconv.Itoa(year.ID), Name: "年终报告.docx", StoragePath: "文档/2024/年终报告.docx", Size: 1},
		{UserID: 1, ParentID: "", Name: "根目录报告.txt", StoragePath: "根目录报告.txt", Size: 1},
		{UserID: 1, ParentID: "", Name: "无关.txt", StoragePath: "无关.txt", Size: 1},
		{UserID: 2, ParentID: "", Name: "别人的报告.txt", StoragePath: "别人的报告.txt", Size: 1},
	}
	for _, record := range records {
		if err := repository.Create(ctx, record); err != nil {
			t.Fatalf("create %s: %v", record.Name, err)
		}
	}
	return service, ctx
}

func TestSearchFilesReturnsCrossDirectoryHitsWithParentPath(t *testing.T) {
	service, ctx := seedSearchTree(t)

	result, err := service.SearchFiles(ctx, 1, " 报告 ")
	if err != nil {
		t.Fatalf("search files: %v", err)
	}
	if result.Truncated {
		t.Fatalf("truncated=true, want false")
	}
	if len(result.Files) != 2 {
		t.Fatalf("got %d hits, want 2", len(result.Files))
	}

	byName := make(map[string]*SearchHit, len(result.Files))
	for _, hit := range result.Files {
		byName[hit.Name] = hit
	}

	nested, ok := byName["年终报告.docx"]
	if !ok {
		t.Fatalf("嵌套目录下的文件未被搜到: %v", byName)
	}
	if nested.ParentPath != "文档/2024" {
		t.Fatalf("parent path=%q, want 文档/2024", nested.ParentPath)
	}
	if len(nested.ParentSegments) != 2 ||
		nested.ParentSegments[0].Name != "文档" || nested.ParentSegments[1].Name != "2024" {
		t.Fatalf("unexpected parent segments: %+v", nested.ParentSegments)
	}

	root, ok := byName["根目录报告.txt"]
	if !ok {
		t.Fatalf("根目录下的文件未被搜到: %v", byName)
	}
	if root.ParentPath != "" || len(root.ParentSegments) != 0 {
		t.Fatalf("root hit should carry an empty path, got %q / %+v", root.ParentPath, root.ParentSegments)
	}

	if _, leaked := byName["别人的报告.txt"]; leaked {
		t.Fatal("搜索结果泄漏了其他用户的文件")
	}
}

func TestSearchFilesTreatsWildcardsAsLiterals(t *testing.T) {
	service, ctx := seedSearchTree(t)

	// 未转义时 "%" 会匹配全部记录，这里必须一条都搜不到。
	result, err := service.SearchFiles(ctx, 1, "%")
	if err != nil {
		t.Fatalf("search files: %v", err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("got %d hits for %%, want 0", len(result.Files))
	}
}

func TestSearchFilesRejectsEmptyKeywordWithoutQuerying(t *testing.T) {
	service, ctx := seedSearchTree(t)

	result, err := service.SearchFiles(ctx, 1, "   ")
	if err != nil {
		t.Fatalf("search files: %v", err)
	}
	if len(result.Files) != 0 {
		t.Fatalf("got %d hits for blank keyword, want 0", len(result.Files))
	}
}

func TestSearchFilesFlagsTruncationBeyondLimit(t *testing.T) {
	db := openTestDB(t)
	repository := newRepository(db)
	service := &fileService{repo: repository, filePath: t.TempDir()}
	ctx := context.Background()

	for i := 0; i < searchResultLimit+5; i++ {
		record := &File{UserID: 1, ParentID: "", Name: "报告" + strconv.Itoa(i) + ".txt", StoragePath: "报告" + strconv.Itoa(i) + ".txt", Size: 1}
		if err := repository.Create(ctx, record); err != nil {
			t.Fatalf("create %s: %v", record.Name, err)
		}
	}

	result, err := service.SearchFiles(ctx, 1, "报告")
	if err != nil {
		t.Fatalf("search files: %v", err)
	}
	if !result.Truncated {
		t.Fatal("truncated=false, want true")
	}
	if len(result.Files) != searchResultLimit {
		t.Fatalf("got %d hits, want %d", len(result.Files), searchResultLimit)
	}
}

func TestResolveParentSegmentsStopsOnCycle(t *testing.T) {
	// 索引成环时不能死循环，能拿到多少段就返回多少段。
	folders := map[string]*File{
		"1": {Name: "a", ParentID: "2"},
		"2": {Name: "b", ParentID: "1"},
	}
	segments := resolveParentSegments(folders, "1")
	if len(segments) != 2 {
		t.Fatalf("got %d segments, want 2", len(segments))
	}
}
