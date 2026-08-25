package files

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newBatchTestHandler(t *testing.T) (*handler, *fileService) {
	t.Helper()
	service := newService(newRepository(openTestDB(t)), t.TempDir(), 5120)
	return newHandler(service), service
}

func performBatchDownload(t *testing.T, h *handler, userID uint64, query string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/files/download-batch?"+query, nil)
	if userID != 0 {
		c.Set("userID", userID)
	}
	h.DownloadBatch(c)
	return recorder
}

// zipEntryContents 把响应体解成 条目名 -> 内容，目录项内容为空串。
func zipEntryContents(t *testing.T, body []byte) map[string]string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	entries := make(map[string]string, len(reader.File))
	for _, entry := range reader.File {
		f, err := entry.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", entry.Name, err)
		}
		content, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			t.Fatalf("read entry %s: %v", entry.Name, err)
		}
		entries[entry.Name] = string(content)
	}
	return entries
}

func TestDownloadBatchStreamsSelectedFiles(t *testing.T) {
	h, service := newBatchTestHandler(t)
	ctx := context.Background()

	ids := make([]string, 0, 3)
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		uploaded, err := service.UploadFile(ctx, 7, "", name, 999, strings.NewReader("content-"+name))
		if err != nil {
			t.Fatalf("upload %s: %v", name, err)
		}
		ids = append(ids, strconv.Itoa(uploaded.ID))
	}

	recorder := performBatchDownload(t, h, 7, "ids="+strings.Join(ids, ",")+"&name=选中文件")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/zip" {
		t.Fatalf("content-type=%q", got)
	}
	if disposition := recorder.Header().Get("Content-Disposition"); !strings.Contains(disposition, "filename*=") {
		t.Fatalf("disposition=%q, want RFC 5987 encoded name", disposition)
	}

	body := recorder.Body.Bytes()
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	if len(reader.File) != 3 {
		t.Fatalf("zip entries=%d, want 3", len(reader.File))
	}
	for _, entry := range reader.File {
		f, err := entry.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", entry.Name, err)
		}
		content, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil || string(content) != "content-"+entry.Name {
			t.Fatalf("entry %s content=%q err=%v", entry.Name, content, err)
		}
	}
}

func TestDownloadBatchPacksFolderTree(t *testing.T) {
	h, service := newBatchTestHandler(t)
	ctx := context.Background()

	folder, err := service.CreateFolder(ctx, 7, "", "照片")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	sub, err := service.CreateFolder(ctx, 7, strconv.Itoa(folder.ID), "子目录")
	if err != nil {
		t.Fatalf("create sub folder: %v", err)
	}
	if _, err := service.UploadFile(ctx, 7, strconv.Itoa(folder.ID), "img1.jpg", 999, strings.NewReader("one")); err != nil {
		t.Fatalf("upload img1: %v", err)
	}
	if _, err := service.UploadFile(ctx, 7, strconv.Itoa(sub.ID), "img2.jpg", 999, strings.NewReader("two")); err != nil {
		t.Fatalf("upload img2: %v", err)
	}
	report, err := service.UploadFile(ctx, 7, "", "报告.pdf", 999, strings.NewReader("pdf"))
	if err != nil {
		t.Fatalf("upload report: %v", err)
	}

	recorder := performBatchDownload(t, h, 7, "ids="+strconv.Itoa(report.ID)+","+strconv.Itoa(folder.ID))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	got := zipEntryContents(t, recorder.Body.Bytes())
	want := map[string]string{
		"报告.pdf":          "pdf",
		"照片/img1.jpg":     "one",
		"照片/子目录/img2.jpg": "two",
	}
	if len(got) != len(want) {
		t.Fatalf("zip entries=%v, want %v", got, want)
	}
	for name, content := range want {
		if got[name] != content {
			t.Errorf("entry %s=%q, want %q", name, got[name], content)
		}
	}
}

func TestDownloadBatchKeepsEmptyFolderAsDirectoryEntry(t *testing.T) {
	h, service := newBatchTestHandler(t)
	folder, err := service.CreateFolder(context.Background(), 7, "", "docs")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}

	recorder := performBatchDownload(t, h, 7, "ids="+strconv.Itoa(folder.ID))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	got := zipEntryContents(t, recorder.Body.Bytes())
	if len(got) != 1 {
		t.Fatalf("zip entries=%v, want only the directory entry", got)
	}
	if _, ok := got["docs/"]; !ok {
		t.Fatalf("zip entries=%v, want docs/", got)
	}
}

func TestDownloadBatchDeduplicatesTopLevelNames(t *testing.T) {
	h, service := newBatchTestHandler(t)
	ctx := context.Background()

	folder, err := service.CreateFolder(ctx, 7, "", "a")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if _, err := service.UploadFile(ctx, 7, strconv.Itoa(folder.ID), "inner.txt", 999, strings.NewReader("inner")); err != nil {
		t.Fatalf("upload inner: %v", err)
	}
	// 同名的文件与文件夹分处不同目录，压缩包里却要挤在同一层。
	other, err := service.CreateFolder(ctx, 7, "", "box")
	if err != nil {
		t.Fatalf("create box: %v", err)
	}
	plain, err := service.UploadFile(ctx, 7, strconv.Itoa(other.ID), "a", 999, strings.NewReader("plain"))
	if err != nil {
		t.Fatalf("upload a: %v", err)
	}

	recorder := performBatchDownload(t, h, 7, "ids="+strconv.Itoa(folder.ID)+","+strconv.Itoa(plain.ID))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	got := zipEntryContents(t, recorder.Body.Bytes())
	if got["a/inner.txt"] != "inner" || got["a(1)"] != "plain" {
		t.Fatalf("zip entries=%v, want a/inner.txt and a(1)", got)
	}
}

func TestDownloadBatchRejectsFilesOfAnotherUser(t *testing.T) {
	h, service := newBatchTestHandler(t)
	uploaded, err := service.UploadFile(context.Background(), 7, "", "a.txt", 999, strings.NewReader("hi"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	recorder := performBatchDownload(t, h, 8, "ids="+strconv.Itoa(uploaded.ID))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", recorder.Code)
	}
}

func TestDownloadBatchRequiresIDs(t *testing.T) {
	h, _ := newBatchTestHandler(t)
	if recorder := performBatchDownload(t, h, 7, "ids=%20,%20"); recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", recorder.Code)
	}
	if recorder := performBatchDownload(t, h, 0, "ids=1"); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", recorder.Code)
	}
}

func TestUniqueEntryNameDeduplicatesAndFlattens(t *testing.T) {
	used := make(map[string]struct{})
	cases := []struct{ in, want string }{
		{"a.txt", "a.txt"},
		{"a.txt", "a(1).txt"},
		{"a.txt", "a(2).txt"},
		{"dir/b.txt", "dir_b.txt"},
		{"", "unnamed"},
	}
	for _, tc := range cases {
		if got := uniqueEntryName(used, tc.in); got != tc.want {
			t.Errorf("uniqueEntryName(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}
