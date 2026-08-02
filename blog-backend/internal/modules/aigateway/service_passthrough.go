package aigateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"dh-blog/internal/platform/search"

	"github.com/sirupsen/logrus"
)

// maxCachedPassthroughBody keeps a large upstream payload out of the in-memory
// result cache.
const maxCachedPassthroughBody = 512 << 10

// PassthroughResult is what the handler writes back to the caller: the
// upstream's own status, content type and bytes.
type PassthroughResult struct {
	Status      int
	ContentType string
	Body        []byte
	Cached      bool
	Provider    string
}

// cachedPassthrough is the payload stored in the result cache.
type cachedPassthrough struct {
	Status       int
	ContentType  string
	Body         []byte
	Credits      int
	CostMicroUSD int
}

// Passthrough forwards a request to one named provider in that provider's own
// wire format.
//
// Unlike Search there is deliberately no routing and no fallback: the caller
// picked this endpoint precisely because it wants that provider's schema, and
// answering with a different provider's response shape would break the SDK on
// the other end. Metering, quota, pacing, health and accounting still apply.
func (s *Service) Passthrough(ctx context.Context, key *APIKey, provider, endpoint string,
	req search.PassthroughRequest, clientIP string) (PassthroughResult, error) {

	started := s.now()
	entry := RequestLog{
		CreatedAt: started,
		Endpoint:  endpoint,
		Provider:  provider,
		ClientIP:  clientIP,
	}
	if key != nil {
		entry.APIKeyID = key.ID
	}

	result, credits, costMicroUSD, err := s.passthrough(ctx, key, provider, req, started)
	entry.LatencyMS = int(s.now().Sub(started) / time.Millisecond)

	if err != nil {
		gatewayErr := asGatewayError(err)
		entry.Status = gatewayErr.LogStatus()
		entry.HTTPStatus = gatewayErr.Status
		entry.Error = gatewayErr.Message
		s.enqueueLog(entry)
		return PassthroughResult{}, gatewayErr
	}

	entry.HTTPStatus = result.Status
	entry.Cached = result.Cached
	entry.Credits = credits
	entry.CostMicroUSD = costMicroUSD
	if result.Status >= 200 && result.Status < 300 {
		entry.Status = StatusOK
	} else {
		// The body still goes back to the caller untouched; the log records
		// what the upstream actually said.
		entry.Status = StatusProviderError
		entry.Error = truncateQuery(string(result.Body))
	}
	s.enqueueLog(entry)
	return result, nil
}

func (s *Service) passthrough(ctx context.Context, key *APIKey, provider string,
	req search.PassthroughRequest, now time.Time) (PassthroughResult, int, int, error) {

	if key != nil {
		if !s.rates.allow(key.ID, key.RateLimitPerMin, now) {
			return PassthroughResult{}, 0, 0, newGatewayError(http.StatusTooManyRequests, "rate_limit_exceeded", ErrRateLimited.Error(), "")
		}
		if err := s.checkKeyQuota(ctx, key, now); err != nil {
			return PassthroughResult{}, 0, 0, err
		}
		if !key.Allows(provider) {
			return PassthroughResult{}, 0, 0, newGatewayError(http.StatusForbidden, "provider_not_allowed", ErrProviderNotAllowed.Error(), provider)
		}
	}

	runtime := s.runtime(provider)
	if runtime == nil || !runtime.config.Enabled {
		return PassthroughResult{}, 0, 0, newGatewayError(http.StatusNotFound, "provider_not_found", ErrProviderNotFound.Error(), provider)
	}
	if !runtime.passthrough {
		return PassthroughResult{}, 0, 0, newGatewayError(http.StatusNotFound, "provider_not_found", "该供应商不支持原生透传", provider)
	}
	// 透传不做跨供应商回退，但换一把自家的密钥是合理的，所以这里同样走轮换
	picked := runtime.pick(now)
	if picked == nil {
		return PassthroughResult{}, 0, 0, newGatewayError(http.StatusServiceUnavailable, "no_provider_available",
			"该供应商没有可用的密钥", provider)
	}
	forwarder, ok := picked.provider.(search.Forwarder)
	if !ok {
		return PassthroughResult{}, 0, 0, newGatewayError(http.StatusNotFound, "provider_not_found", "该供应商不支持原生透传", provider)
	}

	if quotaExhausted(candidate{MonthlyQuota: runtime.config.MonthlyQuota, Used: s.providerUsed(ctx, provider, now)}) {
		return PassthroughResult{}, 0, 0, newGatewayError(http.StatusServiceUnavailable, "no_provider_available",
			"该供应商本月配额已用尽", provider)
	}

	cacheKey := passthroughCacheKey(provider, req)
	if s.options.CacheTTL > 0 {
		if hit, found := s.cache.Get(cacheKey); found {
			if payload, valid := hit.(cachedPassthrough); valid {
				return PassthroughResult{
					Status: payload.Status, ContentType: payload.ContentType,
					Body: payload.Body, Cached: true, Provider: provider,
				}, 0, 0, nil
			}
		}
	}

	if !runtime.breaker.Allow() {
		return PassthroughResult{}, 0, 0, newGatewayError(http.StatusServiceUnavailable, "no_provider_available",
			"该供应商已熔断，暂不可用", provider)
	}
	if err := runtime.limiter.Wait(ctx, s.options.QueueWait); err != nil {
		runtime.breaker.Release()
		if errors.Is(err, search.ErrLimiterBusy) {
			return PassthroughResult{}, 0, 0, newGatewayError(http.StatusServiceUnavailable, "no_provider_available",
				"该供应商正在限速排队，请稍后重试", provider)
		}
		return PassthroughResult{}, 0, 0, newGatewayError(http.StatusGatewayTimeout, "provider_timeout", err.Error(), provider)
	}

	response, err := forwarder.Forward(ctx, req)
	if err != nil {
		runtime.breaker.Report(false)
		return PassthroughResult{}, 0, 0, err
	}

	// A 4xx caused by the caller's own body says nothing about the upstream's
	// health, so only server-side failures feed the breaker.
	healthy := response.OK() || !retryableKind(response.Kind)
	runtime.breaker.Report(healthy)
	// 凭据被上游拒了就把这把密钥停掉，下一次透传会自动换到另一把
	if response.Kind == search.KindAuthFailed || response.Kind == search.KindQuotaExceeded {
		s.parkKey(ctx, runtime, picked, &search.Error{
			Provider: provider, Kind: response.Kind, Status: response.Status,
			Message: truncateQuery(string(response.Body)),
		}, now)
	}
	if response.Kind == search.KindQuotaExceeded && runtime.usableKeys(now) == 0 {
		s.markProviderExhausted(ctx, provider, now)
	}

	if response.OK() {
		if err := s.repo.touchProviderKey(ctx, picked.config.ID, now); err != nil {
			logrus.Warnf("更新供应商密钥使用时间失败: %v", err)
		}
		s.accountPassthrough(ctx, key, provider, response.Credits, response.CostMicroUSD, now)
		if s.options.CacheTTL > 0 && len(response.Body) <= maxCachedPassthroughBody {
			_ = s.cache.Set(cacheKey, cachedPassthrough{
				Status: response.Status, ContentType: response.ContentType,
				Body: response.Body, Credits: response.Credits, CostMicroUSD: response.CostMicroUSD,
			}, s.options.CacheTTL)
		}
	}

	return PassthroughResult{
		Status: response.Status, ContentType: response.ContentType,
		Body: response.Body, Provider: provider,
	}, response.Credits, response.CostMicroUSD, nil
}

func retryableKind(kind search.ErrorKind) bool {
	switch kind {
	case search.KindRateLimited, search.KindQuotaExceeded, search.KindUnavailable,
		search.KindTimeout, search.KindAuthFailed:
		return true
	default:
		return false
	}
}

func (s *Service) accountPassthrough(ctx context.Context, key *APIKey, provider string, credits, costMicroUSD int, now time.Time) {
	period := currentPeriod(now)
	if err := s.repo.addUsage(ctx, providerSubject(provider), period, 1, credits, costMicroUSD); err != nil {
		logrus.Warnf("累计供应商用量失败: %v", err)
	}
	if key == nil {
		return
	}
	if err := s.repo.addUsage(ctx, keySubject(key.ID), period, 1, credits, costMicroUSD); err != nil {
		logrus.Warnf("累计 API Key 用量失败: %v", err)
	}
	if err := s.repo.touchAPIKey(ctx, key.ID, now); err != nil {
		logrus.Warnf("更新 API Key 使用时间失败: %v", err)
	}
}

func (s *Service) providerUsed(ctx context.Context, provider string, now time.Time) int {
	usage, err := s.repo.usageFor(ctx, currentPeriod(now), []string{providerSubject(provider)})
	if err != nil {
		return 0
	}
	return usage[providerSubject(provider)].Count
}

// passthroughCacheKey hashes the forwarded request. Callers reach the same
// cache entry only when they send the same provider, route and parameters.
func passthroughCacheKey(provider string, req search.PassthroughRequest) string {
	parts := []string{provider, req.Method, req.Path, req.Query.Encode()}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00") + "\x00" + string(req.Body)))
	return "gw:pass:" + hex.EncodeToString(digest[:])
}
