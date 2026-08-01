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
}

// providerRuntime pairs a provider adapter with its pacing and health state.
type providerRuntime struct {
	config   Provider
	provider search.Provider
	limiter  *search.Limiter
	breaker  *search.Breaker
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
	strategy, err := s.loadStrategy(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.strategy = strategy

	rebuilt := make(map[string]*providerRuntime, len(providers))
	for _, config := range providers {
		adapter, err := s.buildAdapter(config)
		if err != nil {
			logrus.Warnf("搜索供应商 %s 初始化失败: %v", config.Name, err)
			continue
		}
		runtime := &providerRuntime{config: config, provider: adapter}
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

func (s *Service) buildAdapter(config Provider) (search.Provider, error) {
	switch config.Name {
	case search.ProviderBrave:
		return search.NewBrave(config.APIKey, config.BaseURL, s.httpClient), nil
	case search.ProviderTavily:
		var options search.TavilyOptions
		if err := decodeExtra(config.Extra, &options); err != nil {
			logrus.Warnf("解析 Tavily 附加配置失败，使用默认值: %v", err)
		}
		return search.NewTavily(config.APIKey, config.BaseURL, options, s.httpClient), nil
	case search.ProviderExa:
		var options search.ExaOptions
		if err := decodeExtra(config.Extra, &options); err != nil {
			logrus.Warnf("解析 Exa 附加配置失败，使用默认值: %v", err)
		}
		return search.NewExa(config.APIKey, config.BaseURL, options, s.httpClient), nil
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
	started := s.now()
	requestID := newRequestID()

	entry := RequestLog{
		CreatedAt: started,
		Endpoint:  "search",
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
		if err := runtime.limiter.Wait(ctx, s.options.QueueWait); err != nil {
			// Nothing reached the upstream, so the breaker must not treat this
			// as evidence either way.
			runtime.breaker.Release()
			if errors.Is(err, search.ErrLimiterBusy) {
				lastErr = err
				continue
			}
			return SearchResult{}, newGatewayError(http.StatusGatewayTimeout, "provider_timeout", err.Error(), name)
		}

		attempts++
		entry.Provider = name
		response, callErr := runtime.provider.Search(ctx, upstream)
		if callErr == nil {
			runtime.breaker.Report(true)
			return s.finish(ctx, key, req, name, order[0], response, requestID, cacheKey, now), nil
		}

		runtime.breaker.Report(false)
		lastErr = callErr
		s.noteProviderFailure(ctx, name, callErr, now)

		var providerErr *search.Error
		if errors.As(callErr, &providerErr) && !providerErr.Retryable() {
			return SearchResult{}, gatewayErrorFromProvider(providerErr)
		}
		logrus.Warnf("搜索供应商 %s 调用失败，尝试回退: %v", name, callErr)
	}

	return SearchResult{}, s.exhausted(lastErr)
}

// finish records usage, caches the payload and assembles the response.
func (s *Service) finish(ctx context.Context, key *APIKey, req SearchRequest, used, preferred string,
	response search.Response, requestID, cacheKey string, now time.Time) SearchResult {

	period := currentPeriod(now)
	// Usage is written synchronously: quota decisions read it back, and one
	// small upsert is negligible next to the upstream round trip.
	if err := s.repo.addUsage(ctx, providerSubject(used), period, 1, response.Credits, response.CostMicroUSD); err != nil {
		logrus.Warnf("累计供应商用量失败: %v", err)
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
func (s *Service) noteProviderFailure(ctx context.Context, name string, err error, now time.Time) {
	var providerErr *search.Error
	if !errors.As(err, &providerErr) || providerErr.Kind != search.KindQuotaExceeded {
		return
	}
	s.markProviderExhausted(ctx, name, now)
}

// markProviderExhausted pushes a provider's monthly counter up to its
// configured ceiling, so quota filtering reflects what the upstream just told
// us rather than what we had counted ourselves.
func (s *Service) markProviderExhausted(ctx context.Context, name string, now time.Time) {
	runtime := s.runtime(name)
	if runtime == nil || runtime.config.MonthlyQuota <= 0 {
		return
	}
	period := currentPeriod(now)
	usage, err := s.repo.usageFor(ctx, period, []string{providerSubject(name)})
	if err != nil {
		return
	}
	remaining := runtime.config.MonthlyQuota - usage[providerSubject(name)].Count
	if remaining <= 0 {
		return
	}
	if err := s.repo.addUsage(ctx, providerSubject(name), period, remaining, 0, 0); err != nil {
		logrus.Warnf("标记供应商 %s 配额耗尽失败: %v", name, err)
	}
}

// plan asks the routing policy for an ordered candidate list.
func (s *Service) plan(ctx context.Context, key *APIKey, req SearchRequest, now time.Time) ([]string, error) {
	runtimes := s.snapshot()
	if len(runtimes) == 0 {
		return nil, newGatewayError(http.StatusServiceUnavailable, "no_provider_available", ErrNoProviderAvailable.Error(), "")
	}

	subjects := make([]string, 0, len(runtimes))
	for _, runtime := range runtimes {
		subjects = append(subjects, providerSubject(runtime.config.Name))
	}
	usage, err := s.repo.usageFor(ctx, currentPeriod(now), subjects)
	if err != nil {
		return nil, err
	}

	candidates := make([]candidate, 0, len(runtimes))
	for _, runtime := range runtimes {
		if !runtime.config.Enabled {
			continue
		}
		candidates = append(candidates, candidate{
			Name:         runtime.config.Name,
			Capability:   runtime.provider.Capabilities(),
			Priority:     runtime.config.Priority,
			Weight:       runtime.config.Weight,
			MonthlyQuota: runtime.config.MonthlyQuota,
			Used:         usage[providerSubject(runtime.config.Name)].Count,
			Healthy:      runtime.breaker.State() != search.BreakerOpen,
		})
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
		capability := runtime.provider.Capabilities()
		meta := search.MetaFor(runtime.config.Name)
		statuses = append(statuses, providerStatus{
			Name:           runtime.config.Name,
			DisplayName:    runtime.config.DisplayName,
			HomeURL:        meta.HomeURL,
			DocsURL:        meta.DocsURL,
			LogoURL:        meta.LogoURL,
			Healthy:        runtime.breaker.State() != search.BreakerOpen,
			MonthlyQuota:   runtime.config.MonthlyQuota,
			MonthlyUsed:    usage[providerSubject(runtime.config.Name)].Count,
			SupportsAnswer: capability.Answer,
			SupportsRaw:    capability.RawContent,
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
