package aigateway

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"dh-blog/internal/dhcache"
	"dh-blog/internal/platform/search"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// maxAttempts bounds one request to the first choice plus a single fallback.
const maxAttempts = 2

// queryLogLimit truncates the stored query text.
const queryLogLimit = 200

// logBuffer is how many log entries may be queued before the writer drops
// them. Losing an audit line is preferable to stalling a caller.
const logBuffer = 256

// Options carry the tunables read from config.yaml.
type Options struct {
	CacheTTL         time.Duration
	UpstreamTimeout  time.Duration
	QueueWait        time.Duration
	LogRetentionDays int
}

// Dependencies are the application-owned services this module needs.
type Dependencies struct {
	DB         *gorm.DB
	Cache      dhcache.Cache
	HTTPClient *http.Client
	Options    Options
	// Events is optional. Without it the hourly usage sync can pull a
	// credential out of rotation with nobody the wiser.
	Events EventReporter
	// ExtraTools is optional: the agent module's content-writing tools,
	// mounted on the MCP endpoint and filtered by the caller's scopes.
	ExtraTools ToolSource
}

// EventReporter records rotation changes made by the background usage sync for
// the admin event feed. A nil reporter simply means nobody is watching.
type EventReporter interface {
	UsageSyncFinished(failed int, parked, revived []string)
}

// providerKeyRuntime pairs one upstream credential with the adapter built from
// it. Adapters are cheap structs, so one per credential costs nothing and keeps
// the key out of the request-path call signature.
type providerKeyRuntime struct {
	config   ProviderKey
	provider search.Provider
}

// providerRuntime holds a provider's credentials plus its pacing and health
// state. Pacing and health are per provider, not per credential: the rate limit
// protects the upstream account family, and a broken upstream is broken no
// matter which key is presented.
type providerRuntime struct {
	config Provider
	// capability and passthrough come from a keyless probe adapter — both are
	// static per provider, so asking them never requires a live credential.
	capability  search.Capability
	passthrough bool
	// reportsUsage records whether this provider can be asked what it has
	// spent. It is per provider rather than per key, and the admin page shows
	// it so a missing number reads as "this upstream does not tell us" instead
	// of "the sync is broken".
	reportsUsage bool

	mu     sync.Mutex
	keys   []*providerKeyRuntime
	cursor int

	limiter *search.Limiter
	breaker *search.Breaker
}

// pick returns the next usable credential in round-robin order, or nil when the
// provider has none left.
func (r *providerRuntime) pick(now time.Time) *providerKeyRuntime {
	r.mu.Lock()
	defer r.mu.Unlock()
	for range r.keys {
		candidate := r.keys[r.cursor%len(r.keys)]
		r.cursor = (r.cursor + 1) % len(r.keys)
		if candidate.config.Usable(now) {
			return candidate
		}
	}
	return nil
}

// firstUsableKey returns a credential without advancing the rotation cursor,
// for callers that are inspecting rather than serving traffic.
func (r *providerRuntime) firstUsableKey(now time.Time) *providerKeyRuntime {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, key := range r.keys {
		if key.config.Usable(now) {
			return key
		}
	}
	return nil
}

// usableKeys counts the credentials currently in rotation.
func (r *providerRuntime) usableKeys(now time.Time) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, key := range r.keys {
		if key.config.Usable(now) {
			count++
		}
	}
	return count
}

// usableKeyConfigs copies the configuration of the credentials in rotation, for
// the callers that need to add up what those credentials are allowed to spend.
func (r *providerRuntime) usableKeyConfigs(now time.Time) []ProviderKey {
	r.mu.Lock()
	defer r.mu.Unlock()
	configs := make([]ProviderKey, 0, len(r.keys))
	for _, key := range r.keys {
		if key.config.Usable(now) {
			configs = append(configs, key.config)
		}
	}
	return configs
}

// keyIDs lists every credential, in rotation or not, so a caller can ask the
// usage table about all of them in one query.
func (r *providerRuntime) keyIDs() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]int, 0, len(r.keys))
	for _, key := range r.keys {
		ids = append(ids, key.config.ID)
	}
	return ids
}

// park takes a credential out of rotation in memory. It reports false when the
// key was already parked, so a concurrent second failure does not log twice.
func (r *providerRuntime) park(id int, status, reason string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, key := range r.keys {
		if key.config.ID != id {
			continue
		}
		if key.config.Status == status {
			return false
		}
		key.config.Status = status
		key.config.LastError = truncateQuery(reason)
		parked := now
		key.config.DisabledAt = &parked
		return true
	}
	return false
}

// revive marks a self-recovered credential active again in memory.
func (r *providerRuntime) revive(id int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, key := range r.keys {
		if key.config.ID == id {
			key.config.Status = ProviderKeyActive
			key.config.LastError = ""
			key.config.DisabledAt = nil
			return
		}
	}
}

// credentialSnapshot is a copy of one credential's routing-relevant state, taken
// under the runtime lock so background work can walk the list without holding it
// across an upstream call.
type credentialSnapshot struct {
	id       int
	label    string
	enabled  bool
	status   string
	provider search.Provider
}

// name identifies a credential in a log line or an admin summary. Labels are
// optional, so an unlabelled key falls back to its row id rather than showing up
// as a bare provider name that could be any of several.
func (c credentialSnapshot) name(provider string) string {
	label := strings.TrimSpace(c.label)
	if label == "" {
		label = fmt.Sprintf("#%d", c.id)
	}
	return provider + "/" + label
}

// credentials copies the credential list for an inspector.
func (r *providerRuntime) credentials() []credentialSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	snapshots := make([]credentialSnapshot, 0, len(r.keys))
	for _, key := range r.keys {
		snapshots = append(snapshots, credentialSnapshot{
			id:       key.config.ID,
			label:    key.config.Label,
			enabled:  key.config.Enabled,
			status:   key.config.Status,
			provider: key.provider,
		})
	}
	return snapshots
}

// upstreamCostMicroUSD totals what the upstreams say has been spent under this
// provider's credentials, and reports whether any of them said anything usable.
//
// A provider whose upstream reports money is judged on the upstream's number
// rather than the gateway's own tally, because the gateway only ever sees the
// calls it made: keys shared with something else, or a request issued straight
// against the provider, are invisible to the local counter and very visible on
// the bill.
//
// Every credential counts here, including the parked ones: this is money already
// spent this month, and parking a key does not refund it. That is the opposite
// of upstreamHeadroom below, which asks what is left to spend and therefore only
// counts the credentials still in rotation.
//
// The windows do not line up — Exa reports a rolling 30 days while the local
// counter resets with the calendar month — so a spend cap enforced against this
// number is a cap on the last 30 days. That is the more conservative reading of
// a prepaid balance, but it is a different question from the one the local
// counter answers, which is why both are shown on the admin page.
func (r *providerRuntime) upstreamCostMicroUSD(now time.Time) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	accountMax, keySum, found := 0, 0, false
	for _, key := range r.keys {
		config := key.config
		if config.UpstreamUnit != search.UsageUnitMicroUSD || config.UpstreamSyncedAt == nil {
			continue
		}
		if now.Sub(*config.UpstreamSyncedAt) > upstreamUsageTTL {
			continue
		}
		found = true
		// Account-scoped reports all describe the same bill, so several
		// credentials on one account each report the whole of it. Adding those
		// up charges the same dollars once per key — which is how a two-key
		// provider came to look like it had spent twice what it had.
		if config.UpstreamScope == search.UsageScopeAccount {
			if config.UpstreamUsed > accountMax {
				accountMax = config.UpstreamUsed
			}
			continue
		}
		keySum += config.UpstreamUsed
	}
	if !found {
		return 0, false
	}
	// Mixed scopes cannot be added either, since the account figure already
	// covers the keys underneath it; the larger of the two is the safer read.
	if accountMax > keySum {
		return accountMax, true
	}
	return keySum, true
}

// upstreamHeadroom is the share of the upstreams' own allowance still unspent,
// expressed as a ratio so credits, requests and dollars land on one scale.
//
// This is the only way the gateway learns that an account is nearly empty when
// the plan is sold in a unit the local counter does not speak. Without it, a
// provider whose local ceiling was left open scores a permanent 1.0 on the quota
// term no matter what its upstream says, and eventually beats every provider
// that does declare a ceiling — including when the upstream has been reporting
// zero credits left for days.
//
// Only credentials in rotation count: a parked one contributes no capacity, and
// its spent allowance is not something future requests can draw on.
//
// Account-scoped credentials draw from one shared pool and the report does not
// say which account a key belongs to, so they are folded by taking the tightest
// rather than summed — adding them would invent capacity that does not exist.
// Key-scoped credentials are independent, so their remainders do add up, and
// that sum is exactly the "two accounts of 1000 make 2000" the provider-level
// ceiling could never express.
func (r *providerRuntime) upstreamHeadroom(now time.Time) (float64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tightest, hasAccount := 1.0, false
	keyRemaining, keyLimit := 0, 0
	for _, key := range r.keys {
		config := key.config
		if !config.Usable(now) || config.UpstreamLimit <= 0 || config.UpstreamSyncedAt == nil {
			continue
		}
		if now.Sub(*config.UpstreamSyncedAt) > upstreamUsageTTL {
			continue
		}
		remaining := config.UpstreamLimit - config.UpstreamUsed
		if remaining < 0 {
			remaining = 0
		}
		if config.UpstreamScope == search.UsageScopeAccount {
			hasAccount = true
			if ratio := float64(remaining) / float64(config.UpstreamLimit); ratio < tightest {
				tightest = ratio
			}
			continue
		}
		keyRemaining += remaining
		keyLimit += config.UpstreamLimit
	}

	if keyLimit > 0 {
		byKey := float64(keyRemaining) / float64(keyLimit)
		if hasAccount && tightest < byKey {
			return tightest, true
		}
		return byKey, true
	}
	if hasAccount {
		return tightest, true
	}
	return 0, false
}

// recordUsage mirrors a synced report into the runtime, so a routing decision
// taken before the next Reload sees the same numbers the admin page does.
func (r *providerRuntime) recordUsage(id int, report search.UsageReport, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, key := range r.keys {
		if key.config.ID != id {
			continue
		}
		synced := now
		key.config.UpstreamUsed = report.Used
		key.config.UpstreamLimit = report.Limit
		key.config.UpstreamUnit = report.Unit
		key.config.UpstreamScope = report.Scope
		key.config.UpstreamWindow = report.Window
		key.config.UpstreamSyncedAt = &synced
		key.config.UpstreamError = ""
		return
	}
}

// recordUsageError notes that a refresh failed without touching the last known
// numbers: a stale figure is still more informative than a blank one.
func (r *providerRuntime) recordUsageError(id int, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, key := range r.keys {
		if key.config.ID == id {
			key.config.UpstreamError = message
			return
		}
	}
}

// pruneInterval is how often expired request logs are swept. Retention is
// measured in days, so a daily pass is granular enough.
const pruneInterval = 24 * time.Hour

// Service orchestrates one gateway search: authenticate, meter, route, call,
// fall back, and account for what was spent.
type Service struct {
	repo       *repository
	cache      dhcache.Cache
	httpClient *http.Client
	options    Options

	mu       sync.RWMutex
	runtimes map[string]*providerRuntime
	strategy RoutingStrategy

	rates *minuteCounters
	now   func() time.Time

	// events reports background rotation changes; nil when nothing listens.
	events EventReporter

	logs     chan RequestLog
	pruner   *time.Ticker
	stop     chan struct{}
	workerWG sync.WaitGroup
	stopOnce sync.Once
}

func newService(deps Dependencies) (*Service, error) {
	service := &Service{
		repo:       newRepository(deps.DB),
		cache:      deps.Cache,
		httpClient: deps.HTTPClient,
		options:    deps.Options,
		runtimes:   make(map[string]*providerRuntime),
		rates:      newMinuteCounters(),
		now:        time.Now,
		events:     deps.Events,
		logs:       make(chan RequestLog, logBuffer),
		stop:       make(chan struct{}),
	}
	if service.httpClient == nil {
		service.httpClient = &http.Client{Timeout: service.options.UpstreamTimeout}
	}
	if err := service.repo.ensureDefaults(context.Background()); err != nil {
		return nil, err
	}
	if err := service.Reload(context.Background()); err != nil {
		return nil, err
	}
	service.workerWG.Add(1)
	go service.writeLogs()
	service.workerWG.Add(1)
	go service.usageSyncLoop()
	if service.options.LogRetentionDays > 0 {
		service.pruner = time.NewTicker(pruneInterval)
		service.workerWG.Add(1)
		go service.pruneLoop()
	}
	return service, nil
}

// Reload rebuilds the provider runtimes from the database. Health and pacing
// state survive a configuration change when they are still meaningful: a
// breaker tracks the upstream, not our settings, and a limiter only has to be
// rebuilt when its rate actually changed.
func (s *Service) Reload(ctx context.Context) error {
	providers, err := s.repo.listProviders(ctx)
	if err != nil {
		return err
	}
	credentials, err := s.repo.listProviderKeys(ctx)
	if err != nil {
		return err
	}
	strategy, err := s.loadStrategy(ctx)
	if err != nil {
		return err
	}

	byProvider := make(map[string][]ProviderKey, len(providers))
	for _, credential := range credentials {
		byProvider[credential.Provider] = append(byProvider[credential.Provider], credential)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.strategy = strategy

	rebuilt := make(map[string]*providerRuntime, len(providers))
	for _, config := range providers {
		probe, err := s.buildAdapter(config, "", "")
		if err != nil {
			logrus.Warnf("搜索供应商 %s 初始化失败: %v", config.Name, err)
			continue
		}
		_, forwards := probe.(search.Forwarder)
		_, reportsUsage := probe.(search.UsageReporter)
		runtime := &providerRuntime{
			config:       config,
			capability:   probe.Capabilities(),
			passthrough:  forwards,
			reportsUsage: reportsUsage,
		}
		for _, credential := range byProvider[config.Name] {
			adapter, err := s.buildAdapter(config, credential.APIKey, credential.UsageKeyID)
			if err != nil {
				logrus.Warnf("供应商 %s 的密钥 %d 初始化失败: %v", config.Name, credential.ID, err)
				continue
			}
			runtime.keys = append(runtime.keys, &providerKeyRuntime{config: credential, provider: adapter})
		}
		if previous, ok := s.runtimes[config.Name]; ok {
			runtime.breaker = previous.breaker
			if previous.limiter.Rate() == config.RPS {
				runtime.limiter = previous.limiter
			}
		}
		if runtime.breaker == nil {
			runtime.breaker = search.NewBreaker()
		}
		if runtime.limiter == nil {
			runtime.limiter = search.NewLimiter(config.RPS)
		}
		rebuilt[config.Name] = runtime
	}
	s.runtimes = rebuilt
	return nil
}

// buildAdapter builds one provider adapter. usageKeyID belongs to the credential
// rather than the provider, so it is passed in separately instead of being read
// out of Extra: it names a single key at the upstream, and one id shared by a
// whole rotation makes every credential report the same key's spend.
func (s *Service) buildAdapter(config Provider, apiKey, usageKeyID string) (search.Provider, error) {
	switch config.Name {
	case search.ProviderBrave:
		return search.NewBrave(apiKey, config.BaseURL, s.httpClient), nil
	case search.ProviderTavily:
		var options search.TavilyOptions
		if err := decodeExtra(config.Extra, &options); err != nil {
			logrus.Warnf("解析 Tavily 附加配置失败，使用默认值: %v", err)
		}
		return search.NewTavily(apiKey, config.BaseURL, options, s.httpClient), nil
	case search.ProviderExa:
		var options search.ExaOptions
		if err := decodeExtra(config.Extra, &options); err != nil {
			logrus.Warnf("解析 Exa 附加配置失败，使用默认值: %v", err)
		}
		// The credential is the only source for this id; a leftover copy in Extra
		// would be the shared-id bug coming back through a side door.
		options.UsageKeyID = usageKeyID
		return search.NewExa(apiKey, config.BaseURL, options, s.httpClient), nil
	case search.ProviderFirecrawl:
		var options search.FirecrawlOptions
		if err := decodeExtra(config.Extra, &options); err != nil {
			logrus.Warnf("解析 Firecrawl 附加配置失败，使用默认值: %v", err)
		}
		return search.NewFirecrawl(apiKey, config.BaseURL, options, s.httpClient), nil
	default:
		return nil, fmt.Errorf("未知的搜索供应商: %s", config.Name)
	}
}

// decodeExtra reads a provider's private JSON settings, treating an empty value
// as "use the defaults".
func decodeExtra(extra string, target any) error {
	trimmed := strings.TrimSpace(extra)
	if trimmed == "" {
		return nil
	}
	return json.Unmarshal([]byte(trimmed), target)
}

// Shutdown stops the background workers. It is safe to call more than once.
func (s *Service) Shutdown() {
	s.stopOnce.Do(func() {
		if s.pruner != nil {
			s.pruner.Stop()
		}
		close(s.stop)
		close(s.logs)
		s.workerWG.Wait()
	})
}

// pruneLoop drops request logs past the configured retention.
func (s *Service) pruneLoop() {
	defer s.workerWG.Done()
	for {
		select {
		case <-s.pruner.C:
			removed, err := s.pruneLogs(context.Background())
			if err != nil {
				logrus.Warnf("清理网关请求日志失败: %v", err)
				continue
			}
			if removed > 0 {
				logrus.Infof("已清理 %d 条过期的网关请求日志", removed)
			}
		case <-s.stop:
			return
		}
	}
}

func (s *Service) writeLogs() {
	defer s.workerWG.Done()
	for entry := range s.logs {
		record := entry
		if err := s.repo.writeLog(context.Background(), &record); err != nil {
			logrus.Warnf("写入网关请求日志失败: %v", err)
		}
	}
}

func (s *Service) enqueueLog(entry RequestLog) {
	select {
	case s.logs <- entry:
	default:
		logrus.Warn("网关日志队列已满，丢弃一条请求日志")
	}
}

// SearchRequest is the gateway's own request shape, already validated.
type SearchRequest struct {
	Query             string
	Provider          string
	MaxResults        int
	Topic             string
	Freshness         string
	Country           string
	Language          string
	IncludeDomains    []string
	ExcludeDomains    []string
	IncludeAnswer     bool
	IncludeRawContent bool
	AllowFallback     bool
	NoCache           bool
}

// SearchMeta describes how a result was produced.
type SearchMeta struct {
	RequestID string `json:"request_id"`
	Cached    bool   `json:"cached"`
	LatencyMS int    `json:"latency_ms"`
	Credits   int    `json:"credits"`
	// CostMicroUSD is the request's price in millionths of a US dollar. It is
	// zero for providers that bill by unit rather than by amount.
	CostMicroUSD     int    `json:"cost_micro_usd,omitempty"`
	FallbackFrom     string `json:"fallback_from,omitempty"`
	ResultsTruncated bool   `json:"results_truncated"`
}

// SearchResult is what the gateway returns to an agent.
type SearchResult struct {
	Query    string          `json:"query"`
	Provider string          `json:"provider"`
	Answer   string          `json:"answer,omitempty"`
	Results  []search.Result `json:"results"`
	Meta     SearchMeta      `json:"meta"`
}

// cachedSearch is the payload stored in the result cache.
type cachedSearch struct {
	Provider     string
	Query        string
	Answer       string
	Results      []search.Result
	Credits      int
	CostMicroUSD int
}

// Search runs one gateway search on behalf of an authenticated key.
func (s *Service) Search(ctx context.Context, key *APIKey, req SearchRequest, clientIP string) (SearchResult, error) {
	return s.SearchFrom(ctx, key, req, clientIP, "search")
}

// SearchFrom is Search with an explicit流水标签。统一接口与 MCP 走的是同一条
// 路径，只有日志里的 endpoint 能区分调用来自哪个入口。
func (s *Service) SearchFrom(ctx context.Context, key *APIKey, req SearchRequest, clientIP, endpoint string) (SearchResult, error) {
	started := s.now()
	requestID := newRequestID()

	entry := RequestLog{
		CreatedAt: started,
		Endpoint:  endpoint,
		Query:     truncateQuery(req.Query),
		ClientIP:  clientIP,
	}
	if key != nil {
		entry.APIKeyID = key.ID
	}

	result, err := s.search(ctx, key, req, requestID, &entry)
	entry.LatencyMS = int(s.now().Sub(started) / time.Millisecond)

	if err != nil {
		gatewayErr := asGatewayError(err)
		entry.Status = gatewayErr.LogStatus()
		entry.HTTPStatus = gatewayErr.Status
		entry.Error = gatewayErr.Message
		if entry.Provider == "" {
			entry.Provider = gatewayErr.Provider
		}
		s.enqueueLog(entry)
		return SearchResult{}, gatewayErr
	}

	result.Meta.RequestID = requestID
	result.Meta.LatencyMS = entry.LatencyMS

	entry.Status = StatusOK
	entry.HTTPStatus = http.StatusOK
	entry.Provider = result.Provider
	entry.ResultCount = len(result.Results)
	entry.Cached = result.Meta.Cached
	entry.Credits = result.Meta.Credits
	entry.CostMicroUSD = result.Meta.CostMicroUSD
	entry.FallbackFrom = result.Meta.FallbackFrom
	s.enqueueLog(entry)
	return result, nil
}

func (s *Service) search(ctx context.Context, key *APIKey, req SearchRequest, requestID string, entry *RequestLog) (SearchResult, error) {
	now := s.now()

	if key != nil {
		if !s.rates.allow(key.ID, key.RateLimitPerMin, now) {
			return SearchResult{}, newGatewayError(http.StatusTooManyRequests, "rate_limit_exceeded", ErrRateLimited.Error(), "")
		}
		if err := s.checkKeyQuota(ctx, key, now); err != nil {
			return SearchResult{}, err
		}
	}

	cacheKey := s.cacheKey(req)
	if !req.NoCache && s.options.CacheTTL > 0 {
		if hit, ok := s.cache.Get(cacheKey); ok {
			if payload, valid := hit.(cachedSearch); valid {
				return SearchResult{
					Query:    payload.Query,
					Provider: payload.Provider,
					Answer:   payload.Answer,
					Results:  payload.Results,
					Meta:     SearchMeta{RequestID: requestID, Cached: true, Credits: 0},
				}, nil
			}
		}
	}

	order, err := s.plan(ctx, key, req, now)
	if err != nil {
		return SearchResult{}, err
	}

	upstream := search.Request{
		Query:             req.Query,
		MaxResults:        req.MaxResults,
		Topic:             req.Topic,
		Freshness:         req.Freshness,
		Country:           req.Country,
		Language:          req.Language,
		IncludeDomains:    req.IncludeDomains,
		ExcludeDomains:    req.ExcludeDomains,
		IncludeAnswer:     req.IncludeAnswer,
		IncludeRawContent: req.IncludeRawContent,
	}

	var lastErr error
	attempts := 0
	for _, name := range order {
		if attempts >= maxAttempts {
			break
		}
		runtime := s.runtime(name)
		if runtime == nil {
			continue
		}
		// A provider whose breaker is open or whose queue is saturated is
		// skipped without consuming an attempt: nothing was actually tried.
		if !runtime.breaker.Allow() {
			continue
		}

		attempts++
		entry.Provider = name
		response, providerKeyID, reached, callErr := s.callProvider(ctx, runtime, upstream, now)
		if callErr == nil {
			runtime.breaker.Report(true)
			return s.finish(ctx, key, req, name, order[0], providerKeyID, response, requestID, cacheKey, now), nil
		}
		if !reached {
			// Nothing got as far as the upstream, so the breaker must not treat
			// this as evidence either way.
			runtime.breaker.Release()
			if errors.Is(callErr, search.ErrLimiterBusy) || errors.Is(callErr, ErrNoProviderAvailable) {
				lastErr = callErr
				continue
			}
			return SearchResult{}, newGatewayError(http.StatusGatewayTimeout, "provider_timeout", callErr.Error(), name)
		}

		runtime.breaker.Report(false)
		lastErr = callErr
		s.noteProviderFailure(ctx, runtime, callErr, now)

		var providerErr *search.Error
		if errors.As(callErr, &providerErr) && !providerErr.Retryable() {
			return SearchResult{}, gatewayErrorFromProvider(providerErr)
		}
		logrus.Warnf("搜索供应商 %s 调用失败，尝试回退: %v", name, callErr)
	}

	return SearchResult{}, s.exhausted(lastErr)
}

// callProvider runs one provider attempt, rotating through that provider's
// credentials. A key the upstream rejects for authentication or quota is parked
// and the next one takes over, because that is a credential problem rather than
// a provider outage — falling back to a different provider there would waste a
// perfectly good upstream.
//
// It returns the credential that served the request, so the caller can bill the
// call to the account that actually paid for it, and a bool reporting whether
// any call reached the upstream, so the caller knows whether the circuit breaker
// saw real evidence.
func (s *Service) callProvider(ctx context.Context, runtime *providerRuntime,
	upstream search.Request, now time.Time) (search.Response, int, bool, error) {

	reached := false
	var lastErr error
	// 轮换上限就是入口处可用的密钥数：每把最多试一次，不会因为轮换把单次请求拖长
	for remaining := runtime.usableKeys(now); remaining > 0; remaining-- {
		picked := runtime.pick(now)
		if picked == nil {
			break
		}
		// 跨月自愈的密钥要把状态写回去，否则后台一直显示它还在停用
		if picked.config.Recovered(now) {
			runtime.revive(picked.config.ID)
			if err := s.repo.reviveProviderKey(ctx, picked.config.ID); err != nil {
				logrus.Warnf("恢复供应商密钥 %d 状态失败: %v", picked.config.ID, err)
			}
		}

		if err := runtime.limiter.Wait(ctx, s.options.QueueWait); err != nil {
			return search.Response{}, 0, reached, err
		}

		response, err := picked.provider.Search(ctx, upstream)
		reached = true
		if err == nil {
			if err := s.repo.touchProviderKey(ctx, picked.config.ID, now); err != nil {
				logrus.Warnf("更新供应商密钥使用时间失败: %v", err)
			}
			return response, picked.config.ID, true, nil
		}

		lastErr = err
		if !s.parkKey(ctx, runtime, picked, err, now) {
			// 不是凭据的问题，换一把也一样，交给上层决定要不要换供应商
			return search.Response{}, 0, true, err
		}
	}

	if lastErr == nil {
		// 一把可用密钥都没有：这不是调用失败，是这家根本没得用
		lastErr = ErrNoProviderAvailable
	}
	return search.Response{}, 0, reached, lastErr
}

// parkKey takes a rejected credential out of rotation and reports whether
// trying the provider's next credential makes sense.
func (s *Service) parkKey(ctx context.Context, runtime *providerRuntime,
	picked *providerKeyRuntime, err error, now time.Time) bool {

	var providerErr *search.Error
	if !errors.As(err, &providerErr) {
		return false
	}
	var status string
	switch providerErr.Kind {
	case search.KindAuthFailed:
		status = ProviderKeyAuthFailed
	case search.KindQuotaExceeded:
		status = ProviderKeyQuotaExceeded
	default:
		return false
	}

	if runtime.park(picked.config.ID, status, providerErr.Message, now) {
		if err := s.repo.parkProviderKey(ctx, picked.config.ID, status, providerErr.Message, now); err != nil {
			logrus.Warnf("停用供应商密钥 %d 失败: %v", picked.config.ID, err)
		}
		logrus.Warnf("供应商 %s 的密钥 %s 已停止调度（%s）: %s",
			runtime.config.Name, MaskSecret(picked.config.APIKey), status, providerErr.Message)
	}
	return true
}

// finish records usage, caches the payload and assembles the response.
func (s *Service) finish(ctx context.Context, key *APIKey, req SearchRequest, used, preferred string,
	providerKeyID int, response search.Response, requestID, cacheKey string, now time.Time) SearchResult {

	period := currentPeriod(now)
	// Usage is written synchronously: quota decisions read it back, and one
	// small upsert is negligible next to the upstream round trip.
	if err := s.repo.addUsage(ctx, providerSubject(used), period, 1, response.Credits, response.CostMicroUSD); err != nil {
		logrus.Warnf("累计供应商用量失败: %v", err)
	}
	// 再按上游密钥记一份：密钥级额度只能靠这份计数衡量，供应商级的总数说不出是哪个账号花的
	if providerKeyID > 0 {
		if err := s.repo.addUsage(ctx, providerKeySubject(providerKeyID), period, 1, response.Credits, response.CostMicroUSD); err != nil {
			logrus.Warnf("累计供应商密钥用量失败: %v", err)
		}
	}
	if key != nil {
		if err := s.repo.addUsage(ctx, keySubject(key.ID), period, 1, response.Credits, response.CostMicroUSD); err != nil {
			logrus.Warnf("累计 API Key 用量失败: %v", err)
		}
		if err := s.repo.touchAPIKey(ctx, key.ID, now); err != nil {
			logrus.Warnf("更新 API Key 使用时间失败: %v", err)
		}
	}

	truncated := req.MaxResults > 0 && len(response.Results) > req.MaxResults
	results := response.Results
	if truncated {
		results = results[:req.MaxResults]
	}
	if results == nil {
		results = []search.Result{}
	}

	if s.options.CacheTTL > 0 {
		_ = s.cache.Set(cacheKey, cachedSearch{
			Provider: used, Query: response.Query, Answer: response.Answer,
			Results: results, Credits: response.Credits, CostMicroUSD: response.CostMicroUSD,
		}, s.options.CacheTTL)
	}

	meta := SearchMeta{
		RequestID: requestID, Credits: response.Credits,
		CostMicroUSD: response.CostMicroUSD, ResultsTruncated: truncated,
	}
	if preferred != "" && preferred != used {
		meta.FallbackFrom = preferred
	}
	return SearchResult{
		Query:    response.Query,
		Provider: used,
		Answer:   response.Answer,
		Results:  results,
		Meta:     meta,
	}
}

// noteProviderFailure records that an upstream declared its own quota gone, so
// the routing policy stops picking it for the rest of the month.
// noteProviderFailure reacts to an upstream saying it is out of allowance.
//
// With several credentials per provider this is deliberately conservative: one
// exhausted key says nothing about the others, so the provider-wide counter is
// only pushed to its ceiling once no credential is left in rotation.
func (s *Service) noteProviderFailure(ctx context.Context, runtime *providerRuntime, err error, now time.Time) {
	var providerErr *search.Error
	if !errors.As(err, &providerErr) || providerErr.Kind != search.KindQuotaExceeded {
		return
	}
	if runtime.usableKeys(now) > 0 {
		return
	}
	s.markProviderExhausted(ctx, runtime.config.Name, now)
}

// markProviderExhausted pushes a provider's monthly counters up to their
// configured ceilings, so quota filtering reflects what the upstream just told
// us rather than what we had counted ourselves.
//
// Both ceilings are pushed, because either one is enough to keep the provider
// out of routing and we do not know which of them the upstream was talking
// about — only that it says there is nothing left.
func (s *Service) markProviderExhausted(ctx context.Context, name string, now time.Time) {
	runtime := s.runtime(name)
	if runtime == nil {
		return
	}
	if runtime.config.MonthlyQuota <= 0 && runtime.config.MonthlyCostLimit <= 0 {
		return
	}
	period := currentPeriod(now)
	usage, err := s.repo.usageFor(ctx, period, []string{providerSubject(name)})
	if err != nil {
		return
	}
	current := usage[providerSubject(name)]

	countGap := 0
	if runtime.config.MonthlyQuota > 0 && current.Count < runtime.config.MonthlyQuota {
		countGap = runtime.config.MonthlyQuota - current.Count
	}
	costGap := 0
	if runtime.config.MonthlyCostLimit > 0 && current.CostMicroUSD < runtime.config.MonthlyCostLimit {
		costGap = runtime.config.MonthlyCostLimit - current.CostMicroUSD
	}
	if countGap == 0 && costGap == 0 {
		return
	}
	if err := s.repo.addUsage(ctx, providerSubject(name), period, countGap, 0, costGap); err != nil {
		logrus.Warnf("标记供应商 %s 配额耗尽失败: %v", name, err)
	}
}

// allowance is what one provider may spend this month, after folding the
// credentials in rotation into a single pair of ceilings.
type allowance struct {
	Quota     int
	Used      int
	CostLimit int
	CostUsed  int
}

// dimension accumulates one kind of ceiling across a provider's credentials.
type dimension struct {
	limit  int
	used   int
	capped bool // at least one credential declared a ceiling of its own
	open   bool // at least one credential declared none
}

func (d *dimension) add(limit, used int) {
	d.used += used
	if limit > 0 {
		d.limit += limit
		d.capped = true
		return
	}
	d.open = true
}

// resolve folds the per-credential figures into one ceiling and the consumption
// measured against it.
//
// Falling back to the provider-level pair when no credential declared anything
// is what keeps existing installs behaving exactly as they did: per-account
// allowances are something an operator opts into by filling them in. One
// uncapped credential makes the whole total uncapped, because summing only the
// ceilings we do know would understate the capacity and stop routing to a
// provider that still has room.
//
// The consumption side switches to the per-credential counters along with the
// ceiling, so the two always measure the same thing. Those counters only start
// filling once this version is running, so a provider given per-account
// allowances mid-month reads as less used than it was until the month rolls
// over. Mixing in the provider-wide total instead would be worse: it includes
// the credentials that have since been parked, and charges their spend against
// the allowance of the ones still in rotation.
func (d dimension) resolve(providerLimit, providerUsed int) (limit, used int) {
	if !d.capped {
		return providerLimit, providerUsed
	}
	if d.open {
		return 0, d.used
	}
	return d.limit, d.used
}

// allowanceOf turns a provider's credentials into the single allowance the
// routing policy scores.
//
// Capacity belongs to the accounts behind the credentials rather than to the
// provider: two accounts of 1000 calls each are 2000, and become 1000 again the
// moment one of them is parked. A single provider-level number can express
// neither — set to 1000 it wastes half the capacity, set to 2000 it keeps
// scheduling against an account that is no longer there.
func allowanceOf(runtime *providerRuntime, now time.Time, usage map[string]Usage) allowance {
	var quota, cost dimension
	for _, key := range runtime.usableKeyConfigs(now) {
		spent := usage[providerKeySubject(key.ID)]
		quota.add(key.MonthlyQuota, spent.Count)
		cost.add(key.MonthlyCostLimit, spent.CostMicroUSD)
	}
	provider := usage[providerSubject(runtime.config.Name)]

	var result allowance
	result.Quota, result.Used = quota.resolve(runtime.config.MonthlyQuota, provider.Count)
	result.CostLimit, result.CostUsed = cost.resolve(runtime.config.MonthlyCostLimit, provider.CostMicroUSD)
	return result
}

// usageSubjects lists every counter the allowance calculation reads, so they can
// be fetched in one query instead of one per credential.
func usageSubjects(runtimes []*providerRuntime) []string {
	subjects := make([]string, 0, len(runtimes)*2)
	for _, runtime := range runtimes {
		subjects = append(subjects, providerSubject(runtime.config.Name))
		for _, id := range runtime.keyIDs() {
			subjects = append(subjects, providerKeySubject(id))
		}
	}
	return subjects
}

// plan asks the routing policy for an ordered candidate list.
func (s *Service) plan(ctx context.Context, key *APIKey, req SearchRequest, now time.Time) ([]string, error) {
	runtimes := s.snapshot()
	if len(runtimes) == 0 {
		return nil, newGatewayError(http.StatusServiceUnavailable, "no_provider_available", ErrNoProviderAvailable.Error(), "")
	}

	usage, err := s.repo.usageFor(ctx, currentPeriod(now), usageSubjects(runtimes))
	if err != nil {
		return nil, err
	}

	candidates := make([]candidate, 0, len(runtimes))
	for _, runtime := range runtimes {
		if !runtime.config.Enabled {
			continue
		}
		// 一把可用密钥都没有的供应商直接不参与选路：留在候选里只会白白消耗一次尝试
		if runtime.usableKeys(now) == 0 {
			continue
		}
		spend := allowanceOf(runtime, now, usage)
		// 上游自己报的花费优先，拿不到或已过期就退回本地累加值
		costUsed := spend.CostUsed
		if upstream, ok := runtime.upstreamCostMicroUSD(now); ok {
			costUsed = upstream
		}
		item := candidate{
			Name:             runtime.config.Name,
			Capability:       runtime.capability,
			Priority:         runtime.config.Priority,
			Weight:           runtime.config.Weight,
			MonthlyQuota:     spend.Quota,
			Used:             spend.Used,
			MonthlyCostLimit: spend.CostLimit,
			CostUsed:         costUsed,
			Healthy:          runtime.breaker.State() != search.BreakerOpen,
		}
		item.UpstreamHeadroom, item.HasUpstreamHeadroom = runtime.upstreamHeadroom(now)
		candidates = append(candidates, item)
	}

	input := routeInput{
		Requested:        req.Provider,
		Strategy:         s.Strategy(),
		AllowFallback:    req.AllowFallback,
		NeedAnswer:       req.IncludeAnswer,
		NeedRawContent:   req.IncludeRawContent,
		PrefersOperators: queryUsesOperators(req.Query),
		Candidates:       candidates,
	}
	if key != nil {
		input.Allowed = key.AllowedList()
	}

	order, err := route(input)
	if err != nil {
		return nil, routeGatewayError(err)
	}
	return order, nil
}

func (s *Service) checkKeyQuota(ctx context.Context, key *APIKey, now time.Time) error {
	if key.MonthlyQuota <= 0 {
		return nil
	}
	usage, err := s.repo.usageFor(ctx, currentPeriod(now), []string{keySubject(key.ID)})
	if err != nil {
		return err
	}
	if usage[keySubject(key.ID)].Count >= key.MonthlyQuota {
		return newGatewayError(http.StatusTooManyRequests, "rate_limit_exceeded", ErrQuotaExceeded.Error(), "")
	}
	return nil
}

// exhausted converts the last upstream failure into the response the caller
// sees once every candidate has been tried.
func (s *Service) exhausted(lastErr error) error {
	if lastErr == nil {
		return newGatewayError(http.StatusServiceUnavailable, "no_provider_available", ErrNoProviderAvailable.Error(), "")
	}
	if errors.Is(lastErr, search.ErrLimiterBusy) {
		return newGatewayError(http.StatusServiceUnavailable, "no_provider_available", "所有搜索供应商都在限速排队中", "")
	}
	var providerErr *search.Error
	if errors.As(lastErr, &providerErr) {
		return gatewayErrorFromProvider(providerErr)
	}
	return newGatewayError(http.StatusBadGateway, "provider_error", lastErr.Error(), "")
}

// ProviderStatuses reports what the caller can currently reach, so an agent
// can pick its parameters instead of discovering a capability gap through an
// empty answer field.
func (s *Service) ProviderStatuses(ctx context.Context, key *APIKey) ([]providerStatus, error) {
	runtimes := s.snapshot()
	subjects := make([]string, 0, len(runtimes))
	for _, runtime := range runtimes {
		subjects = append(subjects, providerSubject(runtime.config.Name))
	}
	usage, err := s.repo.usageFor(ctx, currentPeriod(s.now()), subjects)
	if err != nil {
		return nil, err
	}

	statuses := make([]providerStatus, 0, len(runtimes))
	for _, runtime := range runtimes {
		if !runtime.config.Enabled {
			continue
		}
		if key != nil && !key.Allows(runtime.config.Name) {
			continue
		}
		if runtime.usableKeys(s.now()) == 0 {
			continue
		}
		capability := runtime.capability
		meta := search.MetaFor(runtime.config.Name)
		statuses = append(statuses, providerStatus{
			Name:             runtime.config.Name,
			DisplayName:      runtime.config.DisplayName,
			HomeURL:          meta.HomeURL,
			DocsURL:          meta.DocsURL,
			Healthy:          runtime.breaker.State() != search.BreakerOpen,
			MonthlyQuota:     runtime.config.MonthlyQuota,
			MonthlyUsed:      usage[providerSubject(runtime.config.Name)].Count,
			MonthlyCostLimit: runtime.config.MonthlyCostLimit,
			MonthlyCost:      usage[providerSubject(runtime.config.Name)].CostMicroUSD,
			SupportsAnswer:   capability.Answer,
			SupportsRaw:      capability.RawContent,
		})
	}
	return statuses, nil
}

// Strategy reports the routing strategy currently in force.
func (s *Service) Strategy() RoutingStrategy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.strategy == "" {
		return StrategyBalanced
	}
	return s.strategy
}

// loadStrategy reads the persisted strategy, treating an unset or unrecognized
// value as the balanced default rather than failing the whole reload.
func (s *Service) loadStrategy(ctx context.Context) (RoutingStrategy, error) {
	raw, err := s.repo.setting(ctx, SettingKeyRoutingStrategy)
	if err != nil {
		return "", err
	}
	strategy := RoutingStrategy(strings.TrimSpace(raw))
	if !strategy.Valid() {
		return StrategyBalanced, nil
	}
	return strategy, nil
}

// SetStrategy persists a new routing strategy and applies it immediately.
func (s *Service) SetStrategy(ctx context.Context, strategy RoutingStrategy) error {
	if !strategy.Valid() {
		return fmt.Errorf("未知的调度方式: %s", strategy)
	}
	if err := s.repo.saveSetting(ctx, SettingKeyRoutingStrategy, string(strategy)); err != nil {
		return err
	}
	s.mu.Lock()
	s.strategy = strategy
	s.mu.Unlock()
	if !strategy.Implemented() {
		logrus.Warnf("调度方式已设为 %s，但该方式尚未接入，实际仍按 %s 执行", strategy, StrategyBalanced)
	}
	return nil
}

func (s *Service) runtime(name string) *providerRuntime {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtimes[name]
}

// snapshot returns the runtimes in a stable order for deterministic planning.
func (s *Service) snapshot() []*providerRuntime {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runtimes := make([]*providerRuntime, 0, len(s.runtimes))
	for _, runtime := range s.runtimes {
		runtimes = append(runtimes, runtime)
	}
	sort.Slice(runtimes, func(i, j int) bool { return runtimes[i].config.Name < runtimes[j].config.Name })
	return runtimes
}

// cacheKey hashes every field that can change the result set.
func (s *Service) cacheKey(req SearchRequest) string {
	parts := []string{
		strings.ToLower(strings.Join(strings.Fields(req.Query), " ")),
		strings.ToLower(strings.TrimSpace(req.Provider)),
		fmt.Sprint(req.MaxResults),
		req.Topic, req.Freshness, strings.ToLower(req.Country), strings.ToLower(req.Language),
		strings.Join(req.IncludeDomains, "|"), strings.Join(req.ExcludeDomains, "|"),
		fmt.Sprint(req.IncludeAnswer), fmt.Sprint(req.IncludeRawContent),
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "gw:search:" + hex.EncodeToString(digest[:])
}

func truncateQuery(query string) string {
	runes := []rune(query)
	if len(runes) > queryLogLimit {
		return string(runes[:queryLogLimit])
	}
	return query
}

func newRequestID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "gw_unknown"
	}
	return "gw_" + hex.EncodeToString(buffer)
}
