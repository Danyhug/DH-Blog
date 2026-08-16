package eventlog

import (
	"context"
	"testing"
)

// consumeEvents waits for the writer to persist the expected number of events
// and returns them in publish order (since lists ascending).
func consumeEvents(t *testing.T, service *Service, want int) []*Event {
	t.Helper()
	waitForCursor(t, service, int64(want))
	events, err := service.repo.since(context.Background(), 0, want)
	if err != nil {
		t.Fatalf("since: %v", err)
	}
	if len(events) != want {
		t.Fatalf("read %d events, want %d", len(events), want)
	}
	return events
}

// TestContentReporterFiresTheFiveAuditEvents pins the whole audit contract at
// once: every write action that must show up in the feed publishes exactly one
// event with the designed kind/status/title, and the two update flavours only
// differ by the viaGrant flag.
func TestContentReporterFiresTheFiveAuditEvents(t *testing.T) {
	service := newTestService(t)
	reporter := &ContentReporter{service: service}

	reporter.ArticleCreated("Claude", "Go 并发模型", 11)
	reporter.ArticleUpdated("Claude", "部署脚本", 12, false)
	reporter.ArticleUpdated("Claude", "站长的文章", 13, true)
	reporter.GrantIssued(42, "让 Claude 改错别字")
	reporter.ArticleUpdateDenied("Claude", "加密的文章", 14, "这篇文章不是本 Agent 创建的，修改需要临时授权 Token")

	events := consumeEvents(t, service, 5)

	assertEvent(t, events[0], EventExpect{
		Source: SourceArticle, Kind: kindAgentWrite, Status: StatusSuccess,
		TargetID: 11, Title: "Claude 发布了《Go 并发模型》", Detail: "",
		msg: "created",
	})
	assertEvent(t, events[1], EventExpect{
		Source: SourceArticle, Kind: kindAgentWrite, Status: StatusSuccess,
		TargetID: 12, Title: "Claude 修改了《部署脚本》", Detail: "",
		msg: "own update",
	})
	assertEvent(t, events[2], EventExpect{
		Source: SourceArticle, Kind: kindAgentWrite, Status: StatusSuccess,
		TargetID: 13, Title: "Claude 持临时授权修改了《站长的文章》", Detail: "",
		msg: "grant update",
	})
	assertEvent(t, events[3], EventExpect{
		Source: SourceArticle, Kind: kindEditGrant, Status: StatusRunning,
		TargetID: 42, Title: "已签发 AI 修改授权，1 小时后失效", Detail: "让 Claude 改错别字",
		msg: "grant issued",
	})
	assertEvent(t, events[4], EventExpect{
		Source: SourceArticle, Kind: kindAgentWrite, Status: StatusFailed,
		TargetID: 14, Title: "Claude 尝试修改《加密的文章》被拒绝",
		Detail: "这篇文章不是本 Agent 创建的，修改需要临时授权 Token",
		msg:    "denied",
	})
	// The grant-issued event must not be mistaken for a background job that
	// succeeded or failed: it is administrative work in flight.
	if events[3].Failed() {
		t.Error("grant-issued event reports failed, want running")
	}
}

// TestContentReporterDeniedWithEmptyReason does not leak an error box for a
// denial reason that was never provided.
func TestContentReporterDeniedWithEmptyReason(t *testing.T) {
	service := newTestService(t)
	reporter := &ContentReporter{service: service}

	reporter.ArticleUpdateDenied("Claude", "旧文章", 7, "  ")
	events := consumeEvents(t, service, 1)

	if events[0].Detail != "" {
		t.Fatalf("Detail = %q, want empty for a blank reason", events[0].Detail)
	}
	if events[0].Title != "Claude 尝试修改《旧文章》被拒绝" {
		t.Fatalf("Title = %q", events[0].Title)
	}
}

// TestGrantIssuedTrimsWhitespaceOnlyNotes keeps the Detail column clean when
// the admin signs a grant without a note.
func TestGrantIssuedTrimsWhitespaceOnlyNotes(t *testing.T) {
	service := newTestService(t)
	reporter := &ContentReporter{service: service}

	reporter.GrantIssued(0, "  ")
	events := consumeEvents(t, service, 1)

	if events[0].TargetID != 0 || events[0].Status != StatusRunning {
		t.Fatalf("event = %+v, want target 0 / running", events[0])
	}
	if events[0].Detail != "" {
		t.Fatalf("Detail = %q, want empty for a blank note", events[0].Detail)
	}
	if events[0].Title != "已签发 AI 修改授权，1 小时后失效" {
		t.Fatalf("Title = %q", events[0].Title)
	}
}

// EventExpect is one row of the audit contract the adapter tests assert.
type EventExpect struct {
	Source, Kind, Status string
	TargetID             int
	Title, Detail        string
	msg                  string
}

func assertEvent(t *testing.T, got *Event, want EventExpect) {
	t.Helper()
	if got.Source != want.Source || got.Kind != want.Kind || got.Status != want.Status {
		t.Errorf("%s event = %s/%s/%s, want %s/%s/%s",
			want.msg, got.Source, got.Kind, got.Status, want.Source, want.Kind, want.Status)
	}
	if got.TargetID != want.TargetID {
		t.Errorf("%s TargetID = %d, want %d", want.msg, got.TargetID, want.TargetID)
	}
	if got.Title != want.Title {
		t.Errorf("%s Title = %q, want %q", want.msg, got.Title, want.Title)
	}
	if got.Detail != want.Detail {
		t.Errorf("%s Detail = %q, want %q", want.msg, got.Detail, want.Detail)
	}
}
