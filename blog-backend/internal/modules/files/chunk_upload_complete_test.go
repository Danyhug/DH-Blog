package files

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// newChunkTestContext 构造带 userID 的 gin 测试上下文。
// chunk handler 会读 c.Request.Context() 与 c.Get("userID")，两者都要就位。
func newChunkTestContext(t *testing.T, userID uint64, request *http.Request) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	ctx.Set("userID", float64(userID))
	return ctx, recorder
}

func decodeAjax(t *testing.T, recorder *httptest.ResponseRecorder) (int, string, map[string]any) {
	t.Helper()
	var resp struct {
		Code int            `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return resp.Code, resp.Msg, resp.Data
}

func initChunkSession(t *testing.T, handler *chunkUploadHandler, userID uint64, fileName string, fileSize, chunkSize int64, existingUploadID string) string {
	t.Helper()
	requestBody := map[string]any{
		"fileName":  fileName,
		"fileSize":  fileSize,
		"chunkSize": chunkSize,
	}
	if existingUploadID != "" {
		requestBody["uploadId"] = existingUploadID
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/files/upload/chunk/init", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	ctx, recorder := newChunkTestContext(t, userID, request)
	handler.InitChunkUpload(ctx)

	code, msg, data := decodeAjax(t, recorder)
	if code != 1 {
		t.Fatalf("init failed: code=%d msg=%s", code, msg)
	}
	uploadID, _ := data["uploadId"].(string)
	if uploadID == "" {
		t.Fatalf("init returned empty uploadId: %s", recorder.Body.String())
	}
	return uploadID
}

func uploadChunkPart(t *testing.T, handler *chunkUploadHandler, userID uint64, uploadID string, index int, content []byte) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("uploadId", uploadID); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("chunkIndex", strconv.Itoa(index)); err != nil {
		t.Fatal(err)
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="chunk"; filename="chunk_%d"`, index))
	header.Set("Content-Type", "application/octet-stream")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/files/upload/chunk", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx, recorder := newChunkTestContext(t, userID, request)
	handler.UploadChunk(ctx)

	code, msg, _ := decodeAjax(t, recorder)
	if code != 1 {
		t.Fatalf("upload chunk %d failed: code=%d msg=%s", index, code, msg)
	}
}

func completeChunkSession(t *testing.T, handler *chunkUploadHandler, userID uint64, uploadID string) (int, string, map[string]any) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"uploadId": uploadID})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/files/upload/chunk/complete", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	ctx, recorder := newChunkTestContext(t, userID, request)
	handler.CompleteChunkUpload(ctx)
	return decodeAjax(t, recorder)
}

// TestCompleteChunkUploadNewFileAndDuplicateRejection 钉住 hasNameConflict 的语义：
// 无同名文件是正常路径，必须能完成上传；有同名文件才报冲突，且不覆盖磁盘内容。
func TestCompleteChunkUploadNewFileAndDuplicateRejection(t *testing.T) {
	storagePath := t.TempDir()
	repository := newRepository(openTestDB(t))
	service := newService(repository, storagePath, 5120)
	handler := newChunkUploadHandler(service)

	uploadID := initChunkSession(t, handler, 1, "movie.mp4", 6, 4, "")
	uploadChunkPart(t, handler, 1, uploadID, 0, []byte("abcd"))
	uploadChunkPart(t, handler, 1, uploadID, 1, []byte("ef"))

	code, msg, data := completeChunkSession(t, handler, 1, uploadID)
	if code != 1 {
		t.Fatalf("complete failed: code=%d msg=%s", code, msg)
	}
	if data["name"] != "movie.mp4" {
		t.Fatalf("unexpected complete response: %+v", data)
	}

	content, err := os.ReadFile(filepath.Join(storagePath, "movie.mp4"))
	if err != nil || string(content) != "abcdef" {
		t.Fatalf("merged content=%q err=%v", content, err)
	}

	files, err := repository.ListAll(context.Background())
	if err != nil || len(files) != 1 {
		t.Fatalf("index records=%d err=%v", len(files), err)
	}
	if files[0].MimeType != "video/mp4" || files[0].Size != 6 {
		t.Fatalf("unexpected record: mime=%q size=%d", files[0].MimeType, files[0].Size)
	}

	// 同参数重新 init 会幂等复用同一会话，complete 必须报同名冲突而不是覆盖文件。
	sameUploadID := initChunkSession(t, handler, 1, "movie.mp4", 6, 4, uploadID)
	if sameUploadID != uploadID {
		t.Fatalf("expected idempotent session reuse, got %q vs %q", sameUploadID, uploadID)
	}
	code, msg, _ = completeChunkSession(t, handler, 1, sameUploadID)
	if code == 1 {
		t.Fatalf("duplicate complete unexpectedly succeeded: %s", msg)
	}
	if msg != "同名文件已存在" {
		t.Fatalf("unexpected duplicate error: %s", msg)
	}
	content, err = os.ReadFile(filepath.Join(storagePath, "movie.mp4"))
	if err != nil || string(content) != "abcdef" {
		t.Fatalf("duplicate upload overwrote disk content: content=%q err=%v", content, err)
	}
}

// TestHasNameConflictTreatsMissingAsNoConflict 直接钉住 helper 语义：
// GORM First 查不到时返回 ErrRecordNotFound，必须归一成"无冲突且无错误"。
func TestHasNameConflictTreatsMissingAsNoConflict(t *testing.T) {
	service := newService(newRepository(openTestDB(t)), t.TempDir(), 5120)
	ctx := context.Background()

	conflict, err := service.hasNameConflict(ctx, 1, "", "novel.txt")
	if err != nil || conflict {
		t.Fatalf("missing file must report no conflict: conflict=%v err=%v", conflict, err)
	}

	if err := service.repo.Create(ctx, &File{UserID: 1, Name: "novel.txt", StoragePath: "novel.txt"}); err != nil {
		t.Fatal(err)
	}
	conflict, err = service.hasNameConflict(ctx, 1, "", "novel.txt")
	if err != nil || !conflict {
		t.Fatalf("existing file must report conflict: conflict=%v err=%v", conflict, err)
	}
}

// TestSyncDiskIncrementalRefreshesMimeAndRebuildsTypeMismatch 验证增量对账：
// 存量 octet-stream 记录要刷成正确 MIME 且保留 ID；
// 目录↔文件类型错位的记录要物理重建。
func TestSyncDiskIncrementalRefreshesMimeAndRebuildsTypeMismatch(t *testing.T) {
	storagePath := t.TempDir()
	if err := os.WriteFile(filepath.Join(storagePath, "movie.mp4"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storagePath, "swapme"), []byte("now a file"), 0o644); err != nil {
		t.Fatal(err)
	}

	repository := newRepository(openTestDB(t))
	service := newService(repository, storagePath, 5120)
	ctx := context.Background()

	staleMime := &File{
		UserID:      1,
		Name:        "movie.mp4",
		IsFolder:    false,
		Size:        999,
		StoragePath: "movie.mp4",
		MimeType:    "application/octet-stream",
		FileHash:    "keep-me",
	}
	if err := repository.Create(ctx, staleMime); err != nil {
		t.Fatal(err)
	}
	staleType := &File{
		UserID:      1,
		Name:        "swapme",
		IsFolder:    true,
		StoragePath: "swapme",
	}
	if err := repository.Create(ctx, staleType); err != nil {
		t.Fatal(err)
	}

	if err := service.doSyncFilesFromDisk(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	files, err := repository.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byPath := make(map[string]*File, len(files))
	for _, file := range files {
		byPath[file.StoragePath] = file
	}

	refreshed := byPath["movie.mp4"]
	if refreshed == nil {
		t.Fatal("movie.mp4 missing after sync")
	}
	if refreshed.ID != staleMime.ID {
		t.Fatalf("mime refresh must keep ID: got %d want %d", refreshed.ID, staleMime.ID)
	}
	if refreshed.MimeType != "video/mp4" || refreshed.Size != 3 || refreshed.FileHash != "keep-me" {
		t.Fatalf("unexpected refresh: mime=%q size=%d hash=%q", refreshed.MimeType, refreshed.Size, refreshed.FileHash)
	}

	rebuilt := byPath["swapme"]
	if rebuilt == nil || rebuilt.IsFolder {
		t.Fatalf("type-mismatched record was not rebuilt as file: %+v", rebuilt)
	}
	if rebuilt.ID == staleType.ID {
		t.Fatalf("type-mismatched record must get a new ID, got %d", rebuilt.ID)
	}
}
