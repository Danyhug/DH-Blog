package aigateway

import (
	"context"
	"errors"
	"fmt"
	"time"

	"dh-blog/internal/platform/search"

	"github.com/sirupsen/logrus"
)

// usageSyncInterval is how often the gateway asks the upstreams what they think
// has been spent. Hourly is the compromise the drift deserves: a monthly quota
// does not move fast enough for a tighter loop to tell anyone anything new, and
// the endpoints that report it are somebody else's infrastructure.
const usageSyncInterval = time.Hour

// usageSyncTimeout bounds one credential's refresh. A slow provider must not
// hold up the rest of the sweep.
const usageSyncTimeout = 10 * time.Second

// UsageSyncResult summarises one sweep for the admin page and the log line.
type UsageSyncResult struct {
	// Synced counts credentials whose numbers were refreshed; Skipped counts
	// the ones whose provider cannot report usage at all, or has not reported
	// any yet. Skipped is not a failure, which is why it is counted apart.
	Synced  int      `json:"synced"`
	Skipped int      `json:"skipped"`
	Failed  int      `json:"failed"`
	Parked  []string `json:"parked"`
	Revived []string `json:"revived"`
}

// usageSyncLoop refreshes upstream usage on a timer. The first tick is a full
// interval away on purpose: startup is when the gateway has the least reason to
// call anyone, and an operator who wants the numbers now has the sync button.
func (s *Service) usageSyncLoop() {
	defer s.workerWG.Done()
	ticker := time.NewTicker(usageSyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			result := s.SyncUsage(context.Background())
			if result.Failed > 0 || len(result.Parked) > 0 || len(result.Revived) > 0 {
				logrus.Infof("上游用量同步完成: 更新 %d, 跳过 %d, 失败 %d, 停用 %v, 恢复 %v",
					result.Synced, result.Skipped, result.Failed, result.Parked, result.Revived)
				// Only the noteworthy sweeps reach the feed. A quiet hourly
				// sync that changed nothing is not news.
				if s.events != nil {
					s.events.UsageSyncFinished(result.Failed, result.Parked, result.Revived)
				}
			}
		}
	}
}

// SyncUsage asks every credential's upstream what it has consumed and writes the
// answer back.
//
// The numbers are stored beside the local counter rather than replacing it: they
// are in the provider's own unit and cover the provider's own window, so folding
// them into a request count would produce a figure that means nothing. What they
// do drive is rotation — a credential the upstream says is spent leaves the
// rotation immediately instead of waiting for a caller's request to fail.
func (s *Service) SyncUsage(ctx context.Context) UsageSyncResult {
	var result UsageSyncResult
	for _, runtime := range s.snapshot() {
		for _, credential := range runtime.credentials() {
			// A credential the operator switched off is not something the
			// gateway should be spending calls on.
			if !credential.enabled {
				result.Skipped++
				continue
			}
			s.syncCredential(ctx, runtime, credential, &result)
		}
	}
	return result
}

// syncCredential refreshes one credential and reacts to what came back.
func (s *Service) syncCredential(ctx context.Context, runtime *providerRuntime,
	credential credentialSnapshot, result *UsageSyncResult) {

	reporter, ok := credential.provider.(search.UsageReporter)
	if !ok {
		result.Skipped++
		return
	}

	callCtx, cancel := context.WithTimeout(ctx, usageSyncTimeout)
	report, err := reporter.Usage(callCtx)
	cancel()

	if err != nil {
		// Brave only learns its allowance from real traffic, so "nothing to
		// report yet" is the normal state of a quiet gateway, not a fault.
		if errors.Is(err, search.ErrUsageUnavailable) {
			result.Skipped++
			return
		}
		result.Failed++
		s.recordUsageError(ctx, runtime, credential, err, result)
		return
	}

	now := s.now()
	s.storeUsageReport(ctx, runtime, credential, report, now)
	result.Synced++

	label := credential.name(runtime.config.Name)
	switch {
	case report.Exhausted() && credential.status != ProviderKeyQuotaExceeded:
		reason := fmt.Sprintf("上游用量同步：已用 %d/%d", report.Used, report.Limit)
		if s.parkFromSync(ctx, runtime, credential.id, ProviderKeyQuotaExceeded, reason, now) {
			result.Parked = append(result.Parked, label)
		}
	case !report.Exhausted() && credential.status == ProviderKeyQuotaExceeded:
		// The upstream's own number is the authority on recovery, so a renewed
		// plan puts the key back without waiting for the calendar.
		if s.reviveFromSync(ctx, runtime, credential.id) {
			result.Revived = append(result.Revived, label)
		}
	}
}

// recordUsageError writes down why a refresh failed, and parks the credential
// when the upstream rejected it outright. A revoked key is worth catching here:
// otherwise it stays in rotation until a caller's request runs into it.
func (s *Service) recordUsageError(ctx context.Context, runtime *providerRuntime,
	credential credentialSnapshot, err error, result *UsageSyncResult) {

	message := truncateQuery(err.Error())
	runtime.recordUsageError(credential.id, message)
	if writeErr := s.repo.updateProviderKey(ctx, credential.id, map[string]any{
		"upstream_error": message,
	}); writeErr != nil {
		logrus.Warnf("记录供应商 %s 用量同步错误失败: %v", runtime.config.Name, writeErr)
	}

	var providerErr *search.Error
	if !errors.As(err, &providerErr) || providerErr.Kind != search.KindAuthFailed {
		return
	}
	if s.parkFromSync(ctx, runtime, credential.id, ProviderKeyAuthFailed, "上游用量同步：密钥被拒绝", s.now()) {
		result.Parked = append(result.Parked, credential.name(runtime.config.Name))
	}
}

// storeUsageReport persists a fresh report and mirrors it into the runtime, so
// routing decisions taken before the next Reload see the same numbers the admin
// page does.
func (s *Service) storeUsageReport(ctx context.Context, runtime *providerRuntime,
	credential credentialSnapshot, report search.UsageReport, now time.Time) {

	runtime.recordUsage(credential.id, report, now)
	if err := s.repo.updateProviderKey(ctx, credential.id, map[string]any{
		"upstream_used":      report.Used,
		"upstream_limit":     report.Limit,
		"upstream_unit":      report.Unit,
		"upstream_scope":     report.Scope,
		"upstream_window":    report.Window,
		"upstream_synced_at": now,
		"upstream_error":     "",
	}); err != nil {
		logrus.Warnf("写入供应商 %s 的上游用量失败: %v", runtime.config.Name, err)
	}
}

func (s *Service) parkFromSync(ctx context.Context, runtime *providerRuntime,
	id int, status, reason string, now time.Time) bool {

	if !runtime.park(id, status, reason, now) {
		return false
	}
	if err := s.repo.parkProviderKey(ctx, id, status, reason, now); err != nil {
		logrus.Warnf("停用供应商 %s 的密钥失败: %v", runtime.config.Name, err)
	}
	return true
}

func (s *Service) reviveFromSync(ctx context.Context, runtime *providerRuntime, id int) bool {
	runtime.revive(id)
	if err := s.repo.reviveProviderKey(ctx, id); err != nil {
		logrus.Warnf("恢复供应商 %s 的密钥失败: %v", runtime.config.Name, err)
		return false
	}
	return true
}
