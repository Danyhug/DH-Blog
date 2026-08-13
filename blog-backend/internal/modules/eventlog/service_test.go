package eventlog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(MigrationModels()...); err != nil {
		t.Fatalf("migrate eventlog: %v", err)
	}
	return db
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	service := newService(newRepository(openTestDB(t)))
	t.Cleanup(service.Shutdown)
	return service
}

// waitForCursor gives the asynchronous writer a bounded chance to catch up.
func waitForCursor(t *testing.T, service *Service, want int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if service.Cursor() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("cursor did not reach %d, still %d", want, service.Cursor())
}

func TestPublishPersistsAndIsListable(t *testing.T) {
	service := newTestService(t)

	service.Publish(Event{
		Source: SourceTask, Kind: "AI_Gen_Summary", Status: StatusFailed,
		TargetID: 7, Title: "文章 #7 的 AI 摘要生成 最终失败", Detail: "AI 返回的摘要为空", Attempt: 10,
	})
	waitForCursor(t, service, 1)

	events, total, err := service.repo.list(context.Background(), listFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(events) != 1 {
		t.Fatalf("total = %d, len = %d, want 1/1", total, len(events))
	}
	got := events[0]
	if got.TargetID != 7 || got.Status != StatusFailed || got.Attempt != 10 {
		t.Errorf("stored event = %+v, want target 7 / failed / attempt 10", got)
	}
	if got.CreatedAt.IsZero() {
		t.Error("stored event has no timestamp")
	}
}

func TestListFiltersBySourceAndStatus(t *testing.T) {
	service := newTestService(t)

	service.Publish(Event{Source: SourceTask, Kind: "AI_Gen_Tags", Status: StatusSuccess})
	service.Publish(Event{Source: SourceWebDAV, Kind: kindDiskSync, Status: StatusFailed})
	waitForCursor(t, service, 2)

	events, total, err := service.repo.list(context.Background(), listFilter{
		Source: SourceWebDAV, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(events) != 1 || events[0].Kind != kindDiskSync {
		t.Fatalf("filtered list = %+v (total %d), want only the disk sync", events, total)
	}

	failed, _, err := service.repo.list(context.Background(), listFilter{
		Status: StatusFailed, Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	if len(failed) != 1 || failed[0].Source != SourceWebDAV {
		t.Fatalf("status filter returned %+v", failed)
	}
}

func TestSinceReplaysOnlyNewerEvents(t *testing.T) {
	service := newTestService(t)

	service.Publish(Event{Source: SourceTask, Kind: "AI_Gen_Tags", Status: StatusQueued})
	waitForCursor(t, service, 1)
	cursor := service.Cursor()

	service.Publish(Event{Source: SourceTask, Kind: "AI_Gen_Tags", Status: StatusSuccess})
	waitForCursor(t, service, cursor+1)

	events, err := service.repo.since(context.Background(), cursor, replayLimit)
	if err != nil {
		t.Fatalf("since: %v", err)
	}
	if len(events) != 1 || events[0].Status != StatusSuccess {
		t.Fatalf("replay = %+v, want just the success event", events)
	}
}

func TestPruneKeepsMostRecent(t *testing.T) {
	service := newTestService(t)

	for i := 0; i < 5; i++ {
		service.Publish(Event{Source: SourceTask, Kind: "AI_Gen_Tags", Status: StatusSuccess})
	}
	waitForCursor(t, service, 5)

	removed, err := service.repo.prune(context.Background(), 2)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if removed != 3 {
		t.Fatalf("removed = %d, want 3", removed)
	}
	_, total, err := service.repo.list(context.Background(), listFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list after prune: %v", err)
	}
	if total != 2 {
		t.Fatalf("remaining = %d, want 2", total)
	}
}

func TestPublishAfterShutdownDoesNotPanic(t *testing.T) {
	service := newService(newRepository(openTestDB(t)))
	service.Shutdown()
	// A task worker can outlive the feed during shutdown; publishing then must
	// be a no-op rather than a send on a closed channel.
	service.Publish(Event{Source: SourceTask, Kind: "AI_Gen_Tags", Status: StatusSuccess})
	service.Shutdown() // idempotent
}

// dialTestSocket stands the module's routes up on a real HTTP server and opens
// a client socket against them, exercising the actual upgrade path.
func dialTestSocket(t *testing.T, service *Service) (*websocket.Conn, *httptest.Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := newHandler(service)
	engine.GET("/api/admin/events/ws", handler.stream)

	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)

	conn, resp, err := websocket.DefaultDialer.Dial(
		strings.Replace(server.URL, "http://", "ws://", 1)+"/api/admin/events/ws", nil)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("dial websocket: %v (status %d)", err, status)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn, server
}

func readFrame(t *testing.T, conn *websocket.Conn) (string, json.RawMessage) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	var frame struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &frame); err != nil {
		t.Fatalf("decode frame %s: %v", data, err)
	}
	return frame.Type, frame.Payload
}

func TestSocketReceivesHelloThenBroadcastEvents(t *testing.T) {
	service := newTestService(t)
	conn, _ := dialTestSocket(t, service)

	frameType, payload := readFrame(t, conn)
	if frameType != FrameHello {
		t.Fatalf("first frame = %q, want %q", frameType, FrameHello)
	}
	var hello helloPayload
	if err := json.Unmarshal(payload, &hello); err != nil {
		t.Fatalf("decode hello: %v", err)
	}
	if hello.Clients < 1 {
		t.Errorf("hello reported %d clients, want at least 1", hello.Clients)
	}

	service.Publish(Event{
		Source: SourceWebDAV, Kind: kindDiskSync, Status: StatusSuccess,
		Title: "网盘文件索引重建完成",
	})

	frameType, payload = readFrame(t, conn)
	if frameType != FrameEvent {
		t.Fatalf("second frame = %q, want %q", frameType, FrameEvent)
	}
	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if event.ID == 0 {
		t.Error("broadcast event has no id, clients cannot deduplicate it")
	}
	if event.Title != "网盘文件索引重建完成" {
		t.Errorf("event title = %q", event.Title)
	}
}

func TestSocketAnswersPingWithPong(t *testing.T) {
	service := newTestService(t)
	conn, _ := dialTestSocket(t, service)
	readFrame(t, conn) // hello

	if err := conn.WriteJSON(map[string]string{"type": FramePing}); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	if frameType, _ := readFrame(t, conn); frameType != FramePong {
		t.Fatalf("reply to ping = %q, want %q", frameType, FramePong)
	}
}

func TestSocketRejectsUnknownCommand(t *testing.T) {
	service := newTestService(t)
	conn, _ := dialTestSocket(t, service)
	readFrame(t, conn) // hello

	if err := conn.WriteJSON(map[string]string{"type": "nope"}); err != nil {
		t.Fatalf("write command: %v", err)
	}
	if frameType, _ := readFrame(t, conn); frameType != FrameError {
		t.Fatalf("reply to unknown command = %q, want %q", frameType, FrameError)
	}
}

func TestRegisteredCommandRepliesInKind(t *testing.T) {
	service := newTestService(t)
	// This is the extension point the socket exists for: a new client→server
	// feature is one registration, not a new endpoint.
	service.HandleCommand("echo", func(_ context.Context, payload json.RawMessage) (any, error) {
		return map[string]string{"got": string(payload)}, nil
	})

	conn, _ := dialTestSocket(t, service)
	readFrame(t, conn) // hello

	if err := conn.WriteJSON(map[string]any{"type": "echo", "payload": "hi"}); err != nil {
		t.Fatalf("write command: %v", err)
	}
	frameType, payload := readFrame(t, conn)
	if frameType != "echo" {
		t.Fatalf("reply type = %q, want echo", frameType)
	}
	if !strings.Contains(string(payload), "hi") {
		t.Errorf("reply payload = %s, want it to carry the request", payload)
	}
}

func TestShutdownClosesAttachedSockets(t *testing.T) {
	service := newService(newRepository(openTestDB(t)))
	conn, _ := dialTestSocket(t, service)
	readFrame(t, conn) // hello

	service.Shutdown()

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("socket stayed open after shutdown")
	}
	if service.Clients() != 0 {
		t.Errorf("hub still reports %d clients after shutdown", service.Clients())
	}
}

func TestTaskObserverRecordsFinalFailure(t *testing.T) {
	service := newTestService(t)
	observer := (&Module{service: service}).TaskObserver()

	observer.TaskFailed("AI_Gen_Summary", 42, 10, context.DeadlineExceeded)
	waitForCursor(t, service, 1)

	events, _, err := service.repo.list(context.Background(), listFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := events[0]
	if got.Status != StatusFailed || got.TargetID != 42 || got.Attempt != 10 {
		t.Fatalf("event = %+v, want failed/42/10", got)
	}
	if !strings.Contains(got.Title, "AI 摘要生成") || !strings.Contains(got.Title, "#42") {
		t.Errorf("title = %q, want it to name the job and the article", got.Title)
	}
	if got.Detail == "" {
		t.Error("final failure recorded no error detail, which is the whole point")
	}
}

func TestHTTPListAndReplayEndpoints(t *testing.T) {
	service := newTestService(t)
	service.Publish(Event{Source: SourceTask, Kind: "AI_Gen_Tags", Status: StatusSuccess, Title: "第一条"})
	waitForCursor(t, service, 1)
	first := service.Cursor()
	service.Publish(Event{Source: SourceTask, Kind: "AI_Gen_Tags", Status: StatusFailed, Title: "第二条"})
	waitForCursor(t, service, first+1)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := newHandler(service)
	engine.GET("/events", handler.list)
	engine.GET("/events/since", handler.replay)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/events?status=failed", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "第二条") ||
		strings.Contains(recorder.Body.String(), "第一条") {
		t.Errorf("filtered list body = %s", recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/events/since?sinceId=1", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("replay status = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "第二条") {
		t.Errorf("replay body = %s", recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/events/since?sinceId=abc", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("bad sinceId status = %d, want 400", recorder.Code)
	}
}
