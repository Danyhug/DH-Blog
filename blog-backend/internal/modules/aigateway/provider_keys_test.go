package aigateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// recordingBrave answers like Brave and records which credential each call
// presented, so a test can observe rotation directly instead of inferring it.
type recordingBrave struct {
	mu     sync.Mutex
	seen   []string
	reject map[string]int // 密钥 -> 要返回的 HTTP 状态码
}

func newRecordingBrave() *recordingBrave {
	return &recordingBrave{reject: map[string]int{}}
}

func (r *recordingBrave) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		token := req.Header.Get("X-Subscription-Token")
		r.mu.Lock()
		r.seen = append(r.seen, token)
		status := r.reject[token]
		r.mu.Unlock()

		if status != 0 {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"rejected"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"web": map[string]any{"results": []map[string]any{
			{"title": "ok", "url": "https://ok.dev", "description": "d"},
		}}})
	}
}

func (r *recordingBrave) tokens() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.seen...)
}

func TestProviderKeysRotateAcrossRequests(t *testing.T) {
	upstream := newRecordingBrave()
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: upstream.handler()})
	addTestProviderKey(t, module, "brave", "key-b")
	if err := module.service.Reload(context.Background()); err != nil {
		t.Fatalf("Reload 返回错误: %v", err)
	}

	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)
	for index := range 4 {
		body := `{"query":"轮换 ` + string(rune('a'+index)) + `"}`
		if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, body); recorder.Code != http.StatusOK {
			t.Fatalf("第 %d 次状态码 = %d, body=%s", index, recorder.Code, recorder.Body.String())
		}
	}

	seen := upstream.tokens()
	counts := map[string]int{}
	for _, value := range seen {
		counts[value]++
	}
	// 4 次请求两把密钥各承担一半，说明是轮换而不是永远用第一把
	if counts["test-key"] != 2 || counts["key-b"] != 2 {
		t.Fatalf("密钥使用分布 = %v, 期望两把各 2 次", counts)
	}
}

func TestProviderKeyParkedAfterAuthFailure(t *testing.T) {
	upstream := newRecordingBrave()
	upstream.reject["dead-key"] = http.StatusUnauthorized
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: upstream.handler()})
	deadID := addTestProviderKey(t, module, "brave", "dead-key")
	if err := module.service.Reload(context.Background()); err != nil {
		t.Fatalf("Reload 返回错误: %v", err)
	}

	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	// 轮换到坏密钥的那次请求不该失败：换一把自家的密钥就能继续
	for index := range 4 {
		body := `{"query":"停用 ` + string(rune('a'+index)) + `"}`
		recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("第 %d 次状态码 = %d, body=%s", index, recorder.Code, recorder.Body.String())
		}
	}

	stored, err := module.service.repo.providerKeyByID(context.Background(), deadID)
	if err != nil {
		t.Fatalf("providerKeyByID 返回错误: %v", err)
	}
	if stored.Status != ProviderKeyAuthFailed {
		t.Fatalf("坏密钥状态 = %q, 期望 %q", stored.Status, ProviderKeyAuthFailed)
	}
	if stored.LastError == "" || stored.DisabledAt == nil {
		t.Errorf("停用原因与时间应被记录: %+v", stored)
	}

	// 停用之后不该再被调度
	before := len(upstream.tokens())
	if recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token,
		`{"query":"停用之后"}`); recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", recorder.Code)
	}
	for _, presented := range upstream.tokens()[before:] {
		if presented == "dead-key" {
			t.Fatal("已停用的密钥仍在被调度")
		}
	}
}

func TestProviderDropsOutOfRoutingWhenNoKeyLeft(t *testing.T) {
	upstream := newRecordingBrave()
	upstream.reject["test-key"] = http.StatusUnauthorized
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: upstream.handler()})
	engine := newTestEngine(module)
	token := issueTestKey(t, module, nil)

	// 唯一一把密钥被拒后整家供应商就没得用了。第一次仍会真的打一次上游，
	// 拿到 502（凭据坏了是运维的问题，不是调用方能改请求解决的）
	first := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"第一次"}`)
	if first.Code != http.StatusBadGateway {
		t.Fatalf("状态码 = %d, 期望 502, body=%s", first.Code, first.Body.String())
	}

	before := len(upstream.tokens())
	recorder := doGateway(engine, http.MethodPost, "/api/gateway/v1/search", token, `{"query":"第二次"}`)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d, 期望 503", recorder.Code)
	}
	if got := decodeError(t, recorder).Type; got != "no_provider_available" {
		t.Errorf("错误类型 = %q", got)
	}
	if len(upstream.tokens()) != before {
		t.Error("没有可用密钥时不应再打上游")
	}
}

func TestProviderKeyQuotaParkingRecoversNextMonth(t *testing.T) {
	parked := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	key := ProviderKey{Enabled: true, Status: ProviderKeyQuotaExceeded, DisabledAt: &parked}

	if key.Usable(parked.Add(72 * time.Hour)) {
		t.Error("同一个月内配额不会重置，不该放回调度")
	}
	next := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !key.Usable(next) {
		t.Error("跨月后配额已重置，应自动放回调度")
	}
	if !key.Recovered(next) {
		t.Error("自愈的密钥要能被识别出来，否则后台一直显示停用")
	}

	// 凭据被拒是另一回事：跨月也不该自己回来
	rejected := ProviderKey{Enabled: true, Status: ProviderKeyAuthFailed, DisabledAt: &parked}
	if rejected.Usable(next) {
		t.Error("鉴权失败的密钥不应自动恢复")
	}
	disabled := ProviderKey{Enabled: false, Status: ProviderKeyActive}
	if disabled.Usable(next) {
		t.Error("手动停用的密钥不应参与调度")
	}
}

func TestAdminTestProviderUsesDraftCredential(t *testing.T) {
	upstream := newRecordingBrave()
	upstream.reject["test-key"] = http.StatusUnauthorized
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: upstream.handler()})
	engine := newTestEngine(module)

	// 草稿密钥要在保存之前就能测通，否则只能"先存坏的再等报错"
	recorder := doAdmin(engine, http.MethodPost, "/api/admin/gateway/providers/brave/test",
		`{"apiKey":"draft-key"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data ProbeResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v (body=%s)", err, recorder.Body.String())
	}
	if !response.Data.OK {
		t.Fatalf("草稿密钥应测通: %+v", response.Data)
	}

	tokens := upstream.tokens()
	if len(tokens) == 0 || tokens[len(tokens)-1] != "draft-key" {
		t.Fatalf("上游收到的密钥 = %v, 期望用草稿里的那把", tokens)
	}

	keys, err := module.service.repo.listProviderKeys(context.Background())
	if err != nil {
		t.Fatalf("listProviderKeys 返回错误: %v", err)
	}
	for _, stored := range keys {
		if stored.APIKey == "draft-key" {
			t.Fatal("连通性测试不应把草稿密钥写进库")
		}
	}
}

func TestAdminProviderKeyLifecycle(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{Brave: braveOK("a")})
	engine := newTestEngine(module)

	recorder := doAdmin(engine, http.MethodPost, "/api/admin/gateway/providers/brave/keys",
		`{"label":"备用号","apiKey":"second-key"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("新增密钥状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}

	views, err := module.service.providerViews(context.Background())
	if err != nil {
		t.Fatalf("providerViews 返回错误: %v", err)
	}
	var added providerKeyView
	for _, view := range views {
		if view.Name != "brave" {
			continue
		}
		if view.ActiveKeys != 2 {
			t.Fatalf("在调度的密钥数 = %d, 期望 2", view.ActiveKeys)
		}
		for _, key := range view.Keys {
			if key.Label == "备用号" {
				added = key
			}
		}
	}
	if added.ID == 0 {
		t.Fatal("新增的密钥没有出现在列表里")
	}
	if strings.Contains(added.Masked, "second-key") {
		t.Errorf("密钥未脱敏: %q", added.Masked)
	}

	path := "/api/admin/gateway/providers/brave/keys/" + strconv.Itoa(added.ID)
	if recorder := doAdmin(engine, http.MethodPut, path, `{"enabled":false}`); recorder.Code != http.StatusOK {
		t.Fatalf("停用密钥状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if runtime := module.service.runtime("brave"); runtime.usableKeys(time.Now()) != 1 {
		t.Fatal("停用后应立刻退出调度，不该等到重启")
	}

	if recorder := doAdmin(engine, http.MethodDelete, path, ""); recorder.Code != http.StatusOK {
		t.Fatalf("删除密钥状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	keys, err := module.service.repo.listProviderKeys(context.Background())
	if err != nil {
		t.Fatalf("listProviderKeys 返回错误: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("剩余密钥数 = %d, 期望 1", len(keys))
	}
}

func TestAdminRevealAPIKeyReturnsSamePlaintext(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)

	recorder := doAdmin(engine, http.MethodPost, "/api/admin/gateway/keys", `{"name":"agent"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("创建状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var created struct {
		Data struct {
			ID     int    `json:"id"`
			APIKey string `json:"apiKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("解析创建响应失败: %v", err)
	}

	// 重复复制正是这个端点的目的：明文丢了不该被迫重签发
	recorder = doAdmin(engine, http.MethodGet, "/api/admin/gateway/keys/"+strconv.Itoa(created.Data.ID)+"/reveal", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("查看状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var revealed struct {
		Data struct {
			APIKey string `json:"apiKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &revealed); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if revealed.Data.APIKey != created.Data.APIKey {
		t.Fatalf("再次取到的 Key = %q, 与创建时不一致", revealed.Data.APIKey)
	}

	// 列表接口仍然不带明文
	list := doAdmin(engine, http.MethodGet, "/api/admin/gateway/keys", "")
	if strings.Contains(list.Body.String(), created.Data.APIKey) {
		t.Fatal("列表接口不应返回明文 Key")
	}
}

func TestAdminRejectsUnimplementedStrategy(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	engine := newTestEngine(module)

	recorder := doAdmin(engine, http.MethodPut, "/api/admin/gateway/settings", `{"routingStrategy":"model"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d, 未接入的调度方式应被拒绝", recorder.Code)
	}
	if got := module.service.Strategy(); got != StrategyBalanced {
		t.Fatalf("调度方式 = %q, 不该被改动", got)
	}
}

func doAdmin(engine *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var reader io.Reader = http.NoBody
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder
}
