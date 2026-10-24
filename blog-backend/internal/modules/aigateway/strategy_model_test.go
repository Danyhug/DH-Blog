package aigateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// jevRequestBody is the System One request as the fake server receives it.
type jevRequestBody struct {
	Model     string         `json:"model"`
	State     map[string]any `json:"state"`
	Questions map[string]struct {
		Type         string            `json:"type"`
		Instructions string            `json:"instructions"`
		Criteria     map[string]string `json:"criteria"`
	} `json:"questions"`
}

// jevAnswer answers every Choice question with choice at the given confidence.
func jevAnswer(choice string, confidence float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body jevRequestBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		answers := map[string]any{}
		for id := range body.Questions {
			answers[id] = map[string]any{
				"type": "choice", "choice": choice, "confidence": confidence,
				"probabilities": map[string]float64{choice: confidence},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "jev-1.14.0", "answers": answers,
			"usage": map[string]int{"input_tokens": 321, "output_tokens": 12},
		})
	}
}

// chatAnswer is an OpenAI-compatible server that always answers letter. With
// logprobs it reports letter at probability p and the rest spread over "B"
// (or "A" when letter is "B").
func chatAnswer(letter string, p float64, withLogprobs bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		choice := map[string]any{"message": map[string]any{"role": "assistant", "content": letter}}
		if withLogprobs && body["logprobs"] == true {
			other := "B"
			if letter == "B" {
				other = "A"
			}
			choice["logprobs"] = map[string]any{"content": []any{map[string]any{
				"token": letter, "logprob": math.Log(p),
				"top_logprobs": []any{
					map[string]any{"token": letter, "logprob": math.Log(p)},
					map[string]any{"token": other, "logprob": math.Log(1 - p)},
				},
			}}}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "deepseek-chat", "choices": []any{choice},
			"usage": map[string]int{"prompt_tokens": 456},
		})
	}
}

// recorder wraps a handler and keeps every request body and auth header it saw.
type recorder struct {
	mu     sync.Mutex
	bodies [][]byte
	auth   []string
	next   http.HandlerFunc
}

func (j *recorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	j.mu.Lock()
	j.bodies = append(j.bodies, raw)
	j.auth = append(j.auth, r.Header.Get("Authorization"))
	j.mu.Unlock()
	r.Body = io.NopCloser(bytes.NewReader(raw))
	j.next(w, r)
}

func (j *recorder) count() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return len(j.bodies)
}

func (j *recorder) jevRequest(t *testing.T, index int) jevRequestBody {
	t.Helper()
	var body jevRequestBody
	if err := json.Unmarshal(j.bodies[index], &body); err != nil {
		t.Fatalf("解析 Jev 请求失败: %v", err)
	}
	return body
}

func ptr[T any](value T) *T { return &value }

func configureTestJev(t *testing.T, module *Module, baseURL string) {
	t.Helper()
	update := ModelRouterUpdate{Backend: ptr("jev"), Endpoint: ptr(baseURL), APIKey: ptr("ts-test-key-123456")}
	if err := module.service.UpdateModelRouter(context.Background(), update); err != nil {
		t.Fatalf("配置 Jev 失败: %v", err)
	}
}

func configureTestChat(t *testing.T, module *Module, endpoint string) {
	t.Helper()
	update := ModelRouterUpdate{
		Backend: ptr("openai"), Endpoint: ptr(endpoint), APIKey: ptr("sk-chat-key-123456"), Model: ptr("deepseek-chat"),
	}
	if err := module.service.UpdateModelRouter(context.Background(), update); err != nil {
		t.Fatalf("配置聊天模型失败: %v", err)
	}
}

// newModelStrategyModule enables brave and tavily, with brave nearly out of
// quota so the balanced ranking alone would always pick tavily, then points
// the classifier at classifierHandler via configure.
func newModelStrategyModule(t *testing.T, classifierHandler http.Handler,
	configure func(*testing.T, *Module, string)) (*Module, string) {

	t.Helper()
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:  braveOK("b1"),
		Tavily: tavilyOK("", "t1"),
	})
	ctx := context.Background()
	if err := module.service.repo.updateProvider(ctx, "brave", map[string]any{"monthly_quota": 2000}); err != nil {
		t.Fatalf("更新 brave 失败: %v", err)
	}
	if err := module.service.repo.addUsage(ctx, providerSubject("brave"), currentPeriod(module.service.now()), 1990, 1990, 0); err != nil {
		t.Fatalf("写入用量失败: %v", err)
	}
	if err := module.service.Reload(ctx); err != nil {
		t.Fatalf("Reload 失败: %v", err)
	}

	server := httptest.NewServer(classifierHandler)
	t.Cleanup(server.Close)
	configure(t, module, server.URL)
	if err := module.service.SetStrategy(ctx, StrategyModel); err != nil {
		t.Fatalf("SetStrategy 返回错误: %v", err)
	}
	return module, issueTestKey(t, module, nil)
}

func searchProvider(t *testing.T, module *Module, token, body string) string {
	t.Helper()
	recorder := doGateway(newTestEngine(module), http.MethodPost, "/api/gateway/v1/search", token, body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	return decodeSearch(t, recorder).Provider
}

func TestModelStrategyAsksJevAboutEligibleProviders(t *testing.T) {
	jev := &recorder{next: jevAnswer("brave", 0.9)}
	module, token := newModelStrategyModule(t, jev, configureTestJev)

	if got := searchProvider(t, module, token, `{"query":"site:go.dev release notes","language":"en"}`); got != "brave" {
		t.Fatalf("provider = %q, 期望 Jev 选中的 brave", got)
	}
	if jev.count() != 1 {
		t.Fatalf("Jev 调用次数 = %d, 期望 1", jev.count())
	}

	request := jev.jevRequest(t, 0)
	if jev.auth[0] != "Bearer ts-test-key-123456" {
		t.Errorf("Authorization = %q", jev.auth[0])
	}
	if request.Model != "jev-latest" {
		t.Errorf("model = %q, 期望默认的 jev-latest", request.Model)
	}
	if request.State["query"] != "site:go.dev release notes" || request.State["language"] != "en" {
		t.Errorf("state = %v", request.State)
	}
	if _, set := request.State["wants_direct_answer"]; set {
		t.Error("调用方没要答案，state 里不该出现 wants_direct_answer")
	}
	if len(request.Questions) != 1 {
		t.Fatalf("问题数 = %d, 期望 1", len(request.Questions))
	}
	for _, question := range request.Questions {
		if question.Type != "choice" || question.Instructions != defaultRouterPrompt {
			t.Errorf("问题 = %+v", question)
		}
		if len(question.Criteria) != 2 || question.Criteria["brave"] == "" || question.Criteria["tavily"] == "" {
			t.Errorf("选项 = %v, 期望恰好是两家启用的供应商且都有描述", question.Criteria)
		}
	}
}

func TestModelStrategyUsesCustomPromptAndProfiles(t *testing.T) {
	jev := &recorder{next: jevAnswer("brave", 0.9)}
	module, token := newModelStrategyModule(t, jev, configureTestJev)

	update := ModelRouterUpdate{
		Prompt:   ptr("Pick the provider for a Chinese developer blog."),
		Profiles: map[string]string{"brave": "Use for site: queries.", "tavily": defaultProviderProfiles["tavily"]},
	}
	if err := module.service.UpdateModelRouter(context.Background(), update); err != nil {
		t.Fatalf("更新提示词失败: %v", err)
	}
	searchProvider(t, module, token, `{"query":"x"}`)

	for _, question := range jev.jevRequest(t, 0).Questions {
		if question.Instructions != "Pick the provider for a Chinese developer blog." {
			t.Errorf("instructions = %q", question.Instructions)
		}
		if question.Criteria["brave"] != "Use for site: queries." || question.Criteria["tavily"] != defaultProviderProfiles["tavily"] {
			t.Errorf("criteria = %v", question.Criteria)
		}
	}
	// 与默认值相同的描述不落库，以后改进默认描述时它会跟着更新
	if stored, _ := module.service.repo.setting(context.Background(), SettingKeyRouterProfiles); strings.Contains(stored, "tavily") {
		t.Errorf("profiles = %s, 默认描述不该被存下来", stored)
	}
}

func TestModelStrategyOnlyOffersCapableProviders(t *testing.T) {
	jev := &recorder{next: jevAnswer("tavily", 0.9)}
	module, token := newModelStrategyModule(t, jev, configureTestJev)

	// 要答案时只有 tavily 能接，没得选就不该去问模型
	if got := searchProvider(t, module, token, `{"query":"what is go","include_answer":true}`); got != "tavily" {
		t.Fatalf("provider = %q, 期望 tavily", got)
	}
	if jev.count() != 0 {
		t.Fatalf("只有一个候选时调用次数 = %d, 期望 0", jev.count())
	}
}

func TestModelStrategyIgnoresLowConfidence(t *testing.T) {
	module, token := newModelStrategyModule(t, jevAnswer("brave", 0.2), configureTestJev)

	if got := searchProvider(t, module, token, `{"query":"ambiguous"}`); got != "tavily" {
		t.Fatalf("provider = %q, 置信度不够时应按负载均衡选 tavily", got)
	}
	status := module.service.modelRouterView().Status
	if status.LastProvider != "brave" || status.LastAccepted || status.LastDecidedAt == nil {
		t.Errorf("status = %+v, 应记下这次被弃用的判断", status)
	}
}

func TestModelStrategySurvivesClassifierFailure(t *testing.T) {
	module, token := newModelStrategyModule(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":{"error_type":"authentication_error","message":"invalid api key"}}`))
	}), configureTestJev)

	if got := searchProvider(t, module, token, `{"query":"anything"}`); got != "tavily" {
		t.Fatalf("provider = %q, 模型失败时搜索必须照常按负载均衡完成", got)
	}
	status := module.service.modelRouterView().Status
	if !strings.Contains(status.LastError, "invalid api key") || status.LastErrorAt == nil {
		t.Errorf("status = %+v, 应记下失败原因", status)
	}
}

func TestModelStrategySkipsClassifierForExplicitProvider(t *testing.T) {
	jev := &recorder{next: jevAnswer("tavily", 0.9)}
	module, token := newModelStrategyModule(t, jev, configureTestJev)

	if got := searchProvider(t, module, token, `{"query":"x","provider":"brave"}`); got != "brave" {
		t.Fatalf("provider = %q, 期望尊重显式指定", got)
	}
	if jev.count() != 0 {
		t.Fatalf("显式指定 provider 时调用次数 = %d, 期望 0", jev.count())
	}
}

func TestModelStrategyWithChatModel(t *testing.T) {
	// 选项按名字排序编号：A = brave, B = tavily
	chat := &recorder{next: chatAnswer("A", 0.8, true)}
	module, token := newModelStrategyModule(t, chat, configureTestChat)

	if got := searchProvider(t, module, token, `{"query":"site:go.dev release notes"}`); got != "brave" {
		t.Fatalf("provider = %q, 期望聊天模型选中的 brave", got)
	}
	if chat.auth[0] != "Bearer sk-chat-key-123456" {
		t.Errorf("Authorization = %q", chat.auth[0])
	}

	var body struct {
		Model       string `json:"model"`
		Logprobs    bool   `json:"logprobs"`
		TopLogprobs int    `json:"top_logprobs"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(chat.bodies[0], &body); err != nil {
		t.Fatalf("解析请求失败: %v", err)
	}
	if body.Model != "deepseek-chat" || !body.Logprobs || body.TopLogprobs != 2 || len(body.Messages) != 2 {
		t.Fatalf("请求 = %+v", body)
	}
	system := body.Messages[0].Content
	if !strings.HasPrefix(system, defaultRouterPrompt) || !strings.Contains(system, "A. brave: ") ||
		!strings.Contains(system, "B. tavily: ") {
		t.Errorf("system prompt = %q", system)
	}
	if !strings.Contains(body.Messages[1].Content, `"query":"site:go.dev release notes"`) {
		t.Errorf("user message = %q", body.Messages[1].Content)
	}

	status := module.service.modelRouterView().Status
	if !status.LastCalibrated || math.Abs(status.LastConfidence-0.8) > 1e-9 || status.LastModel != "deepseek-chat" {
		t.Errorf("status = %+v", status)
	}
}

func TestChatModelLowLogprobConfidenceAbstains(t *testing.T) {
	module, token := newModelStrategyModule(t, chatAnswer("A", 0.3, true), configureTestChat)

	if got := searchProvider(t, module, token, `{"query":"x"}`); got != "tavily" {
		t.Fatalf("provider = %q, logprobs 置信度不够时应按负载均衡", got)
	}
}

func TestChatModelWithoutLogprobsIsTakenAsIs(t *testing.T) {
	// 不返回 logprobs 的服务没有置信度可言：照单全收，并如实标为未校准
	module, token := newModelStrategyModule(t, chatAnswer("A", 0, false), configureTestChat)

	if got := searchProvider(t, module, token, `{"query":"x"}`); got != "brave" {
		t.Fatalf("provider = %q, 期望采用模型的选择", got)
	}
	status := module.service.modelRouterView().Status
	if status.LastCalibrated || !status.LastAccepted {
		t.Errorf("status = %+v, 应标为未校准且已采用", status)
	}
}

func TestUpdateModelRouterDropsKeyWhenTargetChanges(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	ctx := context.Background()
	configureTestJev(t, module, "https://api.typesafe.ai")

	// 只改阈值：Key 保留
	if err := module.service.UpdateModelRouter(ctx, ModelRouterUpdate{APIKey: ptr(""), MinConfidence: ptr(0.4)}); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	if settings, _, _ := module.service.router.snapshot(); settings.APIKey != "ts-test-key-123456" || settings.MinConfidence != 0.4 {
		t.Fatalf("settings = %+v", settings)
	}

	// 换成别的服务但没填新 Key：旧 Key 不能被发到新地址
	update := ModelRouterUpdate{Backend: ptr("openai"), Endpoint: ptr("http://localhost:11434/v1"), Model: ptr("qwen2.5")}
	if err := module.service.UpdateModelRouter(ctx, update); err != nil {
		t.Fatalf("更新失败: %v", err)
	}
	settings, _, _ := module.service.router.snapshot()
	if settings.APIKey != "" {
		t.Fatalf("切换服务后 APIKey = %q, 应被清空", settings.APIKey)
	}
	// 本地 Ollama 不需要 Key，地址和模型齐了就算配置好
	if !module.service.router.configured() {
		t.Error("OpenAI 兼容后端有地址和模型即可使用")
	}
}

func TestAdminConfiguresModelRouterAndSwitchesInOneRequest(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	var handler http.Handler = newTestEngine(module)

	body := `{"routingStrategy":"model","modelRouter":{"backend":"jev","apiKey":"ts-secret-abcdef","model":"jev-1.14.0","minConfidence":0.7,"timeoutMs":2500}}`
	recorder := adminRequest(&handler, http.MethodPut, "/api/admin/gateway/settings", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := module.service.Strategy(); got != StrategyModel {
		t.Fatalf("调度方式 = %q, 期望 model", got)
	}

	read := adminRequest(&handler, http.MethodGet, "/api/admin/gateway/settings", "")
	if strings.Contains(read.Body.String(), "ts-secret-abcdef") {
		t.Fatal("设置接口不应返回 Key 明文")
	}
	view := decodeAdmin[settingsView](t, read).ModelRouter
	if !view.Configured || view.APIKeyMasked != "ts-s****cdef" || view.Model != "jev-1.14.0" ||
		view.MinConfidence != 0.7 || view.TimeoutMS != 2500 {
		t.Errorf("modelRouter = %+v", view)
	}
	if len(view.Profiles) != len(defaultProviderProfiles) || view.Profiles["exa"] != defaultProviderProfiles["exa"] {
		t.Errorf("profiles = %v, 应带上默认描述", view.Profiles)
	}

	// 重启后配置仍在
	if err := module.service.Reload(context.Background()); err != nil {
		t.Fatalf("Reload 失败: %v", err)
	}
	if !module.service.router.configured() {
		t.Error("Reload 后判断模型配置丢失")
	}
}

func TestAdminRejectsInvalidModelRouterSettings(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{})
	var handler http.Handler = newTestEngine(module)

	for _, body := range []string{
		`{"modelRouter":{"minConfidence":1.5}}`,
		`{"modelRouter":{"backend":"claude"}}`,
		`{"modelRouter":{"endpoint":"ftp://x"}}`,
		`{"modelRouter":{"timeoutMs":60000}}`,
		`{"modelRouter":{"profiles":{"google":"x"}}}`,
	} {
		recorder := adminRequest(&handler, http.MethodPut, "/api/admin/gateway/settings", body)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: 状态码 = %d, 期望 400", body, recorder.Code)
		}
	}
}

func TestAdminTestModelRouter(t *testing.T) {
	module := newGatewayTestModule(t, gatewayTestConfig{
		Brave:  braveOK("b1"),
		Tavily: tavilyOK("", "t1"),
	})
	var handler http.Handler = newTestEngine(module)

	recorder := adminRequest(&handler, http.MethodPost, "/api/admin/gateway/settings/model-router/test", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("未配置时状态码 = %d, 期望 400", recorder.Code)
	}

	server := httptest.NewServer(jevAnswer("tavily", 0.8))
	t.Cleanup(server.Close)
	configureTestJev(t, module, server.URL)

	result := decodeAdmin[RouterProbeResult](t, adminRequest(&handler, http.MethodPost,
		"/api/admin/gateway/settings/model-router/test", `{"query":"golang generics tutorial"}`))
	if result.Choice != "tavily" || result.Model != "jev-1.14.0" || !result.Accepted || !result.Calibrated ||
		result.InputTokens != 321 || result.Backend != "jev" || result.Query != "golang generics tutorial" {
		t.Errorf("result = %+v", result)
	}
}
