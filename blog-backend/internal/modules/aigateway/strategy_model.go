package aigateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"dh-blog/internal/platform/ai"
	"dh-blog/internal/platform/jev"
	"dh-blog/internal/platform/search"

	"github.com/sirupsen/logrus"
)

// The model strategy asks a classifier one closed-set question per uncached
// auto-routed search: "which of these providers suits this query". Two kinds of
// classifier are supported behind one interface:
//
//   - Jev (TypeSafe's System One API): built for exactly this — a calibrated
//     distribution over the options in a few hundred milliseconds.
//   - Any OpenAI-compatible chat model: answers with an option letter, and the
//     letter token's logprobs stand in for the distribution. Slower, and when
//     the server returns no logprobs there is no confidence at all.
//
// Whatever the backend, it only ever chooses among providers the filters have
// already let through, and every failure abstains to the balanced order.

// RouterBackend names a classifier implementation.
type RouterBackend string

const (
	RouterBackendJev    RouterBackend = "jev"
	RouterBackendOpenAI RouterBackend = "openai"
)

func (b RouterBackend) valid() bool { return b == RouterBackendJev || b == RouterBackendOpenAI }

const (
	// defaultMinConfidence is the confidence below which a pick is ignored.
	// Choosing a search provider is low stakes, but a flat answer carries no
	// more information than the balanced ranking already has.
	defaultMinConfidence = 0.5
	// Per-backend defaults for how long a search waits for the classifier.
	// Past it the search goes ahead on the balanced order. A chat model needs
	// far longer than Jev, and its budget is still a cost on every search.
	defaultJevTimeout  = 3 * time.Second
	defaultChatTimeout = 8 * time.Second
	maxRouterTimeout   = 30 * time.Second
	// maxPromptRunes bounds the operator's prompt; it is sent on every search.
	maxPromptRunes = 4000
	// routerProbeQuery is what the admin test asks about when no query is given.
	routerProbeQuery = "latest release notes of the Go programming language"
)

// ErrModelRouterNotConfigured means the model strategy was chosen, or tested,
// without a usable classifier configuration.
var ErrModelRouterNotConfigured = errors.New("请先在「判断模型」里填好接口与密钥，再启用模型判断")

// defaultRouterPrompt is what the classifier is asked when the operator has not
// written a prompt. English, because that is where both kinds of model are
// most accurate; the queries themselves may be in any language. The option list
// and answer format are added by the client, never written here.
const defaultRouterPrompt = "Which web search provider should handle this search request? " +
	"Pick the one whose index and result format best fit what the query is looking for."

// defaultProviderProfiles describe each upstream to the classifier. Option
// names and these descriptions are all it sees, so each one says when to pick
// the provider — and, where two overlap, when not to — rather than listing
// features they all share. Benchmarks, prices and customer lists stay out: they
// cost tokens on every search and say nothing about which query goes where.
var defaultProviderProfiles = map[string]string{
	search.ProviderBrave: "Brave Search: its own independent web index, not resold from another engine — the " +
		"closest thing to a real general web search. Best when the caller wants links rather than an answer and " +
		"knows what words to search for: news and other fresh or time-sensitive content, navigational lookups " +
		"for a known site or name, and plain lists of real URLs. Also the one for search operators such as " +
		"site:, filetype: or quoted phrases. Cheapest and fastest; returns short snippets only.",
	search.ProviderTavily: "Tavily: a one-stop search API built for AI agents. Returns search results together " +
		"with the relevant content already extracted, and can write a direct answer. Best when the request is a " +
		"question asking for a fact or an explanation (what is, how does, how many, why) and the caller wants " +
		"the answer itself, assembled from several sources, rather than a list of links.",
	search.ProviderExa: "Exa: semantic search over an embedding-based index. Finds pages whose meaning is close " +
		"to the request even when no keyword matches. Best when the request describes what it is looking for " +
		"instead of naming it: things similar to an example (\"projects like X\", \"类似 X 的项目\"), companies or " +
		"people working in a direction, research papers on an idea. Naming an example does not make it a " +
		"keyword lookup. Not for news.",
	search.ProviderFirecrawl: "Firecrawl: a scraper first and a search engine second. Runs a basic keyword search " +
		"and turns every result page into clean full text, handling JavaScript rendering and anti-bot pages. " +
		"Pick it only when the full text of the result pages is what is needed; it is slower, and ranking is " +
		"not its strength.",
}

// ModelRouterSettings is the persisted classifier configuration. Prompt and
// Profiles hold only what the operator changed: an untouched default is stored
// as empty, so improving a default in code reaches every install.
type ModelRouterSettings struct {
	Backend       RouterBackend
	Endpoint      string
	APIKey        string
	Model         string
	Prompt        string
	Profiles      map[string]string
	MinConfidence float64
	// Timeout is zero for the backend's default.
	Timeout time.Duration
}

func (s ModelRouterSettings) prompt() string {
	if prompt := strings.TrimSpace(s.Prompt); prompt != "" {
		return prompt
	}
	return defaultRouterPrompt
}

func (s ModelRouterSettings) profile(name string) string {
	if profile := strings.TrimSpace(s.Profiles[name]); profile != "" {
		return profile
	}
	if profile, ok := defaultProviderProfiles[name]; ok {
		return profile
	}
	return search.MetaFor(name).DisplayName
}

func (s ModelRouterSettings) timeout() time.Duration {
	if s.Timeout > 0 {
		return s.Timeout
	}
	return defaultTimeoutFor(s.Backend)
}

func defaultTimeoutFor(backend RouterBackend) time.Duration {
	if backend == RouterBackendOpenAI {
		return defaultChatTimeout
	}
	return defaultJevTimeout
}

// classification is a classifier's answer in backend-neutral form.
type classification struct {
	Choice     string
	Confidence float64
	// Calibrated is false when the backend gave no probabilities. The
	// threshold cannot be applied then, so the pick is taken as is.
	Calibrated    bool
	Probabilities map[string]float64
	Model         string
	InputTokens   int
}

// accepted reports whether a pick should steer routing.
func (c classification) accepted(minConfidence float64) bool {
	return !c.Calibrated || c.Confidence >= minConfidence
}

// classifier is the one thing the gateway needs from a model backend.
type classifier interface {
	configured() bool
	classify(ctx context.Context, prompt string, state map[string]any, options map[string]string) (classification, error)
}

type jevClassifier struct{ client *jev.Client }

func (j jevClassifier) configured() bool { return j.client.Configured() }

func (j jevClassifier) classify(ctx context.Context, prompt string, state map[string]any, options map[string]string) (classification, error) {
	decision, err := j.client.Choose(ctx, state, jev.Choice{Instructions: prompt, Criteria: options})
	if err != nil {
		return classification{}, err
	}
	return classification{
		Choice:        decision.Choice,
		Confidence:    decision.Confidence,
		Calibrated:    true,
		Probabilities: decision.Probabilities,
		Model:         decision.Model,
		InputTokens:   decision.InputTokens,
	}, nil
}

type chatClassifier struct{ client *ai.ChoiceClient }

func (c chatClassifier) configured() bool { return c.client.Configured() }

func (c chatClassifier) classify(ctx context.Context, prompt string, state map[string]any, options map[string]string) (classification, error) {
	result, err := c.client.Choose(ctx, ai.ChoiceRequest{Instructions: prompt, State: state, Options: options})
	if err != nil {
		return classification{}, err
	}
	out := classification{
		Choice:        result.Choice,
		Probabilities: result.Probabilities,
		Model:         result.Model,
		InputTokens:   result.InputTokens,
	}
	// Confidence is the normalised probability of the pick. It is not the
	// same quantity as Jev's spread-based confidence, so a threshold tuned for
	// one backend should be revisited after switching.
	if result.Probabilities != nil {
		out.Calibrated = true
		out.Confidence = result.Probabilities[result.Choice]
	}
	return out, nil
}

func buildClassifier(settings ModelRouterSettings, httpClient *http.Client) classifier {
	if settings.Backend == RouterBackendOpenAI {
		return chatClassifier{client: ai.NewChoiceClient(settings.Endpoint, settings.APIKey, settings.Model, httpClient)}
	}
	return jevClassifier{client: jev.New(settings.APIKey, settings.Endpoint, settings.Model, httpClient)}
}

// ModelRouterStatus is the classifier's most recent outcome, kept in memory so
// the admin page can show whether the strategy is actually doing anything.
type ModelRouterStatus struct {
	LastProvider   string     `json:"lastProvider"`
	LastConfidence float64    `json:"lastConfidence"`
	LastCalibrated bool       `json:"lastCalibrated"`
	LastModel      string     `json:"lastModel"`
	LastAccepted   bool       `json:"lastAccepted"`
	LastDecidedAt  *time.Time `json:"lastDecidedAt"`
	LastError      string     `json:"lastError"`
	LastErrorAt    *time.Time `json:"lastErrorAt"`
}

// modelRouter owns the classifier and its last outcome. It has its own lock: a
// decision is an outbound call and must not hold the service lock while it
// runs. A reconfiguration clears the status, since it described a backend that
// is no longer the one being asked.
type modelRouter struct {
	mu         sync.Mutex
	settings   ModelRouterSettings
	classifier classifier
	status     ModelRouterStatus
}

func (r *modelRouter) configure(settings ModelRouterSettings, httpClient *http.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.classifier != nil && !sameBackend(r.settings, settings) {
		r.status = ModelRouterStatus{}
	}
	r.settings = settings
	r.classifier = buildClassifier(settings, httpClient)
}

func sameBackend(a, b ModelRouterSettings) bool {
	return a.Backend == b.Backend && a.Endpoint == b.Endpoint && a.APIKey == b.APIKey && a.Model == b.Model
}

func (r *modelRouter) configured() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.classifier != nil && r.classifier.configured()
}

func (r *modelRouter) snapshot() (ModelRouterSettings, classifier, ModelRouterStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.settings, r.classifier, r.status
}

// decide returns the provider the classifier picks for req among pool, or ""
// to leave the order to the balanced ranking. Every failure abstains: the
// classifier is an optimisation, and a search must never fail because of it.
func (r *modelRouter) decide(ctx context.Context, req SearchRequest, pool []candidate) string {
	// One candidate leaves nothing to decide, and none leaves nothing to route.
	if len(pool) < 2 {
		return ""
	}
	settings, model, _ := r.snapshot()
	if model == nil || !model.configured() {
		r.recordError(ErrModelRouterNotConfigured)
		return ""
	}

	ctx, cancel := context.WithTimeout(ctx, settings.timeout())
	defer cancel()
	result, err := model.classify(ctx, settings.prompt(), routerState(req), routerOptions(settings, pool))
	if err != nil {
		r.recordError(err)
		return ""
	}
	accepted := result.accepted(settings.MinConfidence)
	r.recordDecision(result, accepted)
	if !accepted {
		return ""
	}
	return result.Choice
}

func (r *modelRouter) recordDecision(result classification, accepted bool) {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = ModelRouterStatus{
		LastProvider:   result.Choice,
		LastConfidence: result.Confidence,
		LastCalibrated: result.Calibrated,
		LastModel:      result.Model,
		LastAccepted:   accepted,
		LastDecidedAt:  &now,
	}
}

// recordError keeps the failure for the admin page and logs it once per
// distinct message: a bad key fails every search the same way, and one line
// per search would bury everything else in the log.
func (r *modelRouter) recordError(err error) {
	now := time.Now()
	message := err.Error()
	r.mu.Lock()
	repeated := r.status.LastError == message
	r.status.LastError = message
	r.status.LastErrorAt = &now
	r.mu.Unlock()
	if !repeated {
		logrus.Warnf("模型判断失败，本次按负载均衡调度: %v", err)
	}
}

// routerState is the search request as the classifier reads it: structured
// fields rather than prose, and only the ones the caller set, so an unset
// option does not read as a requirement.
func routerState(req SearchRequest) map[string]any {
	state := map[string]any{"query": strings.TrimSpace(req.Query)}
	if req.Topic != "" {
		state["topic"] = req.Topic
	}
	if req.Freshness != "" {
		state["freshness"] = req.Freshness
	}
	if req.Language != "" {
		state["language"] = req.Language
	}
	if req.Country != "" {
		state["country"] = req.Country
	}
	if len(req.IncludeDomains) > 0 {
		state["only_domains"] = req.IncludeDomains
	}
	if len(req.ExcludeDomains) > 0 {
		state["excluded_domains"] = req.ExcludeDomains
	}
	if req.IncludeAnswer {
		state["wants_direct_answer"] = true
	}
	if req.IncludeRawContent {
		state["wants_full_page_text"] = true
	}
	return state
}

func routerOptions(settings ModelRouterSettings, pool []candidate) map[string]string {
	options := make(map[string]string, len(pool))
	for _, item := range pool {
		options[item.Name] = settings.profile(item.Name)
	}
	return options
}

// loadModelRouterSettings reads the classifier configuration. A malformed
// value falls back to its default rather than failing the reload: a typo in
// one setting must not take the whole gateway down at startup.
func (s *Service) loadModelRouterSettings(ctx context.Context) (ModelRouterSettings, error) {
	keys := []string{
		SettingKeyRouterBackend, SettingKeyRouterEndpoint, SettingKeyRouterAPIKey, SettingKeyRouterModel,
		SettingKeyRouterPrompt, SettingKeyRouterProfiles, SettingKeyRouterMinConfidence, SettingKeyRouterTimeoutMS,
	}
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		value, err := s.repo.setting(ctx, key)
		if err != nil {
			return ModelRouterSettings{}, err
		}
		values[key] = strings.TrimSpace(value)
	}

	settings := ModelRouterSettings{
		Backend:       RouterBackend(values[SettingKeyRouterBackend]),
		Endpoint:      values[SettingKeyRouterEndpoint],
		APIKey:        values[SettingKeyRouterAPIKey],
		Model:         values[SettingKeyRouterModel],
		Prompt:        values[SettingKeyRouterPrompt],
		Profiles:      map[string]string{},
		MinConfidence: defaultMinConfidence,
	}
	if !settings.Backend.valid() {
		settings.Backend = RouterBackendJev
	}
	if raw := values[SettingKeyRouterProfiles]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &settings.Profiles); err != nil {
			logrus.Warnf("解析供应商描述失败，使用默认描述: %v", err)
			settings.Profiles = map[string]string{}
		}
	}
	if threshold, err := strconv.ParseFloat(values[SettingKeyRouterMinConfidence], 64); err == nil && validConfidence(threshold) {
		settings.MinConfidence = threshold
	}
	if millis, err := strconv.Atoi(values[SettingKeyRouterTimeoutMS]); err == nil && millis > 0 &&
		time.Duration(millis)*time.Millisecond <= maxRouterTimeout {
		settings.Timeout = time.Duration(millis) * time.Millisecond
	}
	return settings, nil
}

func validConfidence(value float64) bool { return value >= 0 && value <= 1 }

// ModelRouterUpdate changes the classifier configuration. A nil field is left
// alone. An empty APIKey means "keep the stored key" — except when Backend or
// Endpoint changes, where the stored key is dropped unless a new one is given,
// so a key issued by one service is never sent to another.
type ModelRouterUpdate struct {
	Backend       *string           `json:"backend"`
	Endpoint      *string           `json:"endpoint"`
	APIKey        *string           `json:"apiKey"`
	ClearAPIKey   bool              `json:"clearApiKey"`
	Model         *string           `json:"model"`
	Prompt        *string           `json:"prompt"`
	Profiles      map[string]string `json:"profiles"`
	MinConfidence *float64          `json:"minConfidence"`
	TimeoutMS     *int              `json:"timeoutMs"`
}

// UpdateModelRouter validates and persists a configuration change, then
// applies it immediately.
func (s *Service) UpdateModelRouter(ctx context.Context, update ModelRouterUpdate) error {
	current, err := s.loadModelRouterSettings(ctx)
	if err != nil {
		return err
	}
	values, err := modelRouterValues(current, update)
	if err != nil {
		return err
	}
	for key, value := range values {
		if err := s.repo.saveSetting(ctx, key, value); err != nil {
			return err
		}
	}

	settings, err := s.loadModelRouterSettings(ctx)
	if err != nil {
		return err
	}
	s.router.configure(settings, s.httpClient)
	return nil
}

// modelRouterValues turns an update into the settings rows to write.
func modelRouterValues(current ModelRouterSettings, update ModelRouterUpdate) (map[string]string, error) {
	values := map[string]string{}

	backend := current.Backend
	if update.Backend != nil {
		backend = RouterBackend(strings.TrimSpace(*update.Backend))
		if !backend.valid() {
			return nil, fmt.Errorf("未知的判断模型类型: %s", *update.Backend)
		}
		values[SettingKeyRouterBackend] = string(backend)
	}
	endpoint := current.Endpoint
	if update.Endpoint != nil {
		endpoint = strings.TrimSpace(*update.Endpoint)
		if endpoint != "" && !strings.HasPrefix(endpoint, "https://") && !strings.HasPrefix(endpoint, "http://") {
			return nil, fmt.Errorf("接口地址必须以 http:// 或 https:// 开头")
		}
		values[SettingKeyRouterEndpoint] = endpoint
	}

	newKey := ""
	if update.APIKey != nil {
		newKey = strings.TrimSpace(*update.APIKey)
	}
	switch {
	case newKey != "":
		values[SettingKeyRouterAPIKey] = newKey
	case update.ClearAPIKey, backend != current.Backend, endpoint != current.Endpoint:
		values[SettingKeyRouterAPIKey] = ""
	}

	if update.Model != nil {
		values[SettingKeyRouterModel] = strings.TrimSpace(*update.Model)
	}
	if update.Prompt != nil {
		prompt := strings.TrimSpace(*update.Prompt)
		if len([]rune(prompt)) > maxPromptRunes {
			return nil, fmt.Errorf("提示词不能超过 %d 个字符", maxPromptRunes)
		}
		// The default is stored as empty so a later improvement to it applies.
		if prompt == defaultRouterPrompt {
			prompt = ""
		}
		values[SettingKeyRouterPrompt] = prompt
	}
	if update.Profiles != nil {
		profiles := map[string]string{}
		for name, profile := range update.Profiles {
			name = strings.ToLower(strings.TrimSpace(name))
			if _, known := defaultProviderProfiles[name]; !known {
				return nil, fmt.Errorf("未知的搜索供应商: %s", name)
			}
			profile = strings.TrimSpace(profile)
			if len([]rune(profile)) > maxPromptRunes {
				return nil, fmt.Errorf("%s 的描述不能超过 %d 个字符", name, maxPromptRunes)
			}
			if profile != "" && profile != defaultProviderProfiles[name] {
				profiles[name] = profile
			}
		}
		encoded, err := json.Marshal(profiles)
		if err != nil {
			return nil, err
		}
		values[SettingKeyRouterProfiles] = string(encoded)
	}
	if update.MinConfidence != nil {
		if !validConfidence(*update.MinConfidence) {
			return nil, fmt.Errorf("置信度阈值必须在 0 到 1 之间")
		}
		values[SettingKeyRouterMinConfidence] = strconv.FormatFloat(*update.MinConfidence, 'f', -1, 64)
	}
	if update.TimeoutMS != nil {
		millis := *update.TimeoutMS
		if millis < 0 || time.Duration(millis)*time.Millisecond > maxRouterTimeout {
			return nil, fmt.Errorf("超时必须在 0 到 %d 毫秒之间（0 表示使用默认值）", maxRouterTimeout.Milliseconds())
		}
		values[SettingKeyRouterTimeoutMS] = strconv.Itoa(millis)
	}
	return values, nil
}

// routerBackendOption describes a backend for the admin page, so its defaults
// and hints come from the code that applies them.
type routerBackendOption struct {
	Value            string `json:"value"`
	Label            string `json:"label"`
	Description      string `json:"description"`
	EndpointHint     string `json:"endpointHint"`
	DefaultEndpoint  string `json:"defaultEndpoint"`
	DefaultModel     string `json:"defaultModel"`
	DefaultTimeoutMS int64  `json:"defaultTimeoutMs"`
	KeyRequired      bool   `json:"keyRequired"`
}

func routerBackendOptions() []routerBackendOption {
	return []routerBackendOption{
		{
			Value: string(RouterBackendJev), Label: "Jev（TypeSafe）",
			Description: "专为结构化判断训练的 System One 模型，直接返回校准过的概率，" +
				"通常几百毫秒。TypeSafe 不对中国大陆 IP 提供服务，服务器需经境外网络访问。",
			EndpointHint:     "留空使用官方地址",
			DefaultEndpoint:  jev.DefaultBaseURL,
			DefaultModel:     jev.DefaultModel,
			DefaultTimeoutMS: defaultJevTimeout.Milliseconds(),
			KeyRequired:      true,
		},
		{
			Value: string(RouterBackendOpenAI), Label: "OpenAI 兼容",
			Description: "任意 chat completions 接口（DeepSeek、通义、Ollama 等）。模型只回一个选项字母，" +
				"置信度取自 logprobs；接口不支持 logprobs 时阈值不生效。比 Jev 慢，建议用非推理模型。",
			EndpointHint:     "完整的 chat completions 地址，如 https://api.deepseek.com/v1/chat/completions",
			DefaultTimeoutMS: defaultChatTimeout.Milliseconds(),
			KeyRequired:      false,
		},
	}
}

// modelRouterView is the classifier configuration as the admin page sees it.
// The key is never returned, only whether one is set and its masked form.
// Prompt and Profiles are the effective values, defaults filled in, so the
// form opens on what is actually being sent.
type modelRouterView struct {
	Backend         string                `json:"backend"`
	Backends        []routerBackendOption `json:"backends"`
	Configured      bool                  `json:"configured"`
	APIKeyMasked    string                `json:"apiKeyMasked"`
	Endpoint        string                `json:"endpoint"`
	Model           string                `json:"model"`
	Prompt          string                `json:"prompt"`
	DefaultPrompt   string                `json:"defaultPrompt"`
	Profiles        map[string]string     `json:"profiles"`
	DefaultProfiles map[string]string     `json:"defaultProfiles"`
	MinConfidence   float64               `json:"minConfidence"`
	TimeoutMS       int64                 `json:"timeoutMs"`
	Status          ModelRouterStatus     `json:"status"`
}

func (s *Service) modelRouterView() modelRouterView {
	settings, model, status := s.router.snapshot()
	profiles := make(map[string]string, len(defaultProviderProfiles))
	for name := range defaultProviderProfiles {
		profiles[name] = settings.profile(name)
	}
	return modelRouterView{
		Backend:         string(settings.Backend),
		Backends:        routerBackendOptions(),
		Configured:      model != nil && model.configured(),
		APIKeyMasked:    MaskSecret(settings.APIKey),
		Endpoint:        settings.Endpoint,
		Model:           settings.Model,
		Prompt:          settings.prompt(),
		DefaultPrompt:   defaultRouterPrompt,
		Profiles:        profiles,
		DefaultProfiles: defaultProviderProfiles,
		MinConfidence:   settings.MinConfidence,
		TimeoutMS:       settings.timeout().Milliseconds(),
		Status:          status,
	}
}

// RouterProbeResult is one test decision, with the full distribution so an
// operator can see how sure the model was and who came second.
type RouterProbeResult struct {
	Query         string             `json:"query"`
	Backend       string             `json:"backend"`
	Model         string             `json:"model"`
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Calibrated    bool               `json:"calibrated"`
	Probabilities map[string]float64 `json:"probabilities"`
	Accepted      bool               `json:"accepted"`
	InputTokens   int                `json:"inputTokens"`
	LatencyMS     int                `json:"latencyMs"`
}

// TestModelRouter asks the classifier about one query using the saved
// configuration and the providers auto routing could currently pick. Unlike
// the search path it asks even with a single candidate, because the point is
// to exercise the backend, and it uses the backend's ceiling rather than the
// configured timeout, so a slow model shows how slow it is instead of timing out.
func (s *Service) TestModelRouter(ctx context.Context, query string) (RouterProbeResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		query = routerProbeQuery
	}
	settings, model, _ := s.router.snapshot()
	if model == nil || !model.configured() {
		return RouterProbeResult{}, ErrModelRouterNotConfigured
	}
	candidates, err := s.candidates(ctx, s.now())
	if err != nil {
		return RouterProbeResult{}, err
	}
	pool := eligible(routeInput{Candidates: candidates}, "")
	if len(pool) == 0 {
		return RouterProbeResult{}, ErrNoProviderAvailable
	}

	ctx, cancel := context.WithTimeout(ctx, maxRouterTimeout)
	defer cancel()
	started := time.Now()
	result, err := model.classify(ctx, settings.prompt(), routerState(SearchRequest{Query: query}), routerOptions(settings, pool))
	if err != nil {
		s.router.recordError(err)
		return RouterProbeResult{}, err
	}
	accepted := result.accepted(settings.MinConfidence)
	s.router.recordDecision(result, accepted)
	return RouterProbeResult{
		Query:         query,
		Backend:       string(settings.Backend),
		Model:         result.Model,
		Choice:        result.Choice,
		Confidence:    result.Confidence,
		Calibrated:    result.Calibrated,
		Probabilities: result.Probabilities,
		Accepted:      accepted,
		InputTokens:   result.InputTokens,
		LatencyMS:     int(time.Since(started) / time.Millisecond),
	}, nil
}
