package eventlog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func waitForLogCursor(t *testing.T, service *Service, want int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if service.LogCursor() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("log cursor did not reach %d, still %d", want, service.LogCursor())
}

func TestLogRingKeepsMostRecentInOrder(t *testing.T) {
	ring := newLogRing(3)
	for _, message := range []string{"a", "b", "c", "d", "e"} {
		ring.append(LogLine{Message: message})
	}

	lines := ring.snapshot(10)
	if len(lines) != 3 {
		t.Fatalf("snapshot len = %d, want 3 (the ring size)", len(lines))
	}
	// Oldest first, and the two earliest lines have been overwritten.
	if lines[0].Message != "c" || lines[1].Message != "d" || lines[2].Message != "e" {
		t.Fatalf("snapshot = %+v, want c/d/e oldest-first", lines)
	}
	if lines[2].Seq != 5 {
		t.Errorf("newest seq = %d, want 5", lines[2].Seq)
	}
}

func TestLogRingPartiallyFilled(t *testing.T) {
	ring := newLogRing(5)
	ring.append(LogLine{Message: "one"})
	ring.append(LogLine{Message: "two"})

	lines := ring.snapshot(10)
	if len(lines) != 2 || lines[0].Message != "one" || lines[1].Message != "two" {
		t.Fatalf("snapshot = %+v, want one/two", lines)
	}
	if got := ring.snapshot(1); len(got) != 1 || got[0].Message != "two" {
		t.Fatalf("limited snapshot = %+v, want just the newest", got)
	}
}

func TestLogLinesReachTheRingAndTheSocket(t *testing.T) {
	service := newTestService(t)
	conn, _ := dialTestSocket(t, service)
	readFrame(t, conn) // hello

	service.publishLogLine("info", "网盘文件索引重建完成")

	frameType, payload := readFrame(t, conn)
	if frameType != FrameLog {
		t.Fatalf("frame type = %q, want %q", frameType, FrameLog)
	}
	var line LogLine
	if err := json.Unmarshal(payload, &line); err != nil {
		t.Fatalf("decode log line: %v", err)
	}
	if line.Message != "网盘文件索引重建完成" || line.Level != "info" {
		t.Fatalf("streamed line = %+v", line)
	}
	if line.Seq == 0 {
		t.Error("streamed line has no seq, the console cannot deduplicate it")
	}

	waitForLogCursor(t, service, 1)
	if snapshot := service.LogSnapshot(10); len(snapshot) != 1 {
		t.Fatalf("scrollback = %+v, want the one line", snapshot)
	}
}

// The hook runs on the goroutine that logged, so anything it touches must not
// be touched by the fan-out. This drives logrus while a socket is attached,
// which is the exact arrangement that would deadlock if fanout logged.
func TestLogrusHookIsReentrantSafe(t *testing.T) {
	service := newTestService(t)
	hook := &logHook{queue: service.logQueue}

	logger := logrus.New()
	logger.SetOutput(new(strings.Builder))
	logger.AddHook(hook)

	conn, _ := dialTestSocket(t, service)
	readFrame(t, conn) // hello

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			logger.WithField("i", i).Info("压力测试日志行")
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("logging deadlocked while a console was attached")
	}

	waitForLogCursor(t, service, 1)
	frameType, payload := readFrame(t, conn)
	if frameType != FrameLog {
		t.Fatalf("frame type = %q, want %q", frameType, FrameLog)
	}
	var line LogLine
	if err := json.Unmarshal(payload, &line); err != nil {
		t.Fatalf("decode log line: %v", err)
	}
	if line.Fields["i"] == "" {
		t.Errorf("structured field lost, line = %+v", line)
	}
}

// A full queue must drop lines rather than block whoever is logging.
func TestLogHookDropsInsteadOfBlocking(t *testing.T) {
	hook := &logHook{queue: make(chan LogLine, 1)}
	entry := &logrus.Entry{Time: time.Now(), Level: logrus.InfoLevel, Message: "x"}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			_ = hook.Fire(entry)
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Fire blocked on a full queue")
	}
}

func TestLogSnapshotEndpoint(t *testing.T) {
	service := newTestService(t)
	service.publishLogLine("warning", "上游用量同步失败")
	waitForLogCursor(t, service, 1)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/events/logs", newHandler(service).logs)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/events/logs?limit=50", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "上游用量同步失败") || !strings.Contains(body, "warning") {
		t.Errorf("scrollback body = %s", body)
	}
}

func TestLogStreamStopsOnShutdownWithoutClosingQueue(t *testing.T) {
	service := newService(newRepository(openTestDB(t)))
	service.Shutdown()
	// Other packages keep logging all the way through shutdown; the hook must
	// survive that without a send on a closed channel.
	service.publishLogLine("info", "关闭之后仍然有日志")
	service.Shutdown()
}
