package aigateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

// The admin surface keeps the blog's {code,msg,data} contract so the existing
// frontend axios interceptor works unchanged. Only the agent-facing gateway
// endpoints use the bare JSON contract.
func adminSuccess(c *gin.Context, data ...any) {
	if len(data) == 0 {
		c.JSON(http.StatusOK, response.Success())
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(data[0]))
}

func adminFailure(c *gin.Context, status int, message string) {
	c.JSON(status, response.Error(message))
}

// strategyOption describes one scheduling mode for the admin page, so the UI
// does not have to hardcode copy that could drift from the engine's behaviour.
type strategyOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Implemented bool   `json:"implemented"`
}

type settingsView struct {
	RoutingStrategy string           `json:"routingStrategy"`
	Strategies      []strategyOption `json:"strategies"`
}

func strategyOptions() []strategyOption {
	return []strategyOption{
		{
			Value: string(StrategyBalanced), Label: "负载均衡", Implemented: true,
			Description: "按各家剩余月配额比例与权重打分排序，优先级字段不参与。适合让多家免费额度均匀消耗。",
		},
		{
			Value: string(StrategyPriority), Label: "按优先级", Implemented: true,
			Description: "严格按优先级从小到大选择，同级看权重。剩余配额只用于剔除已用尽的供应商。适合主备场景。",
		},
		{
			Value: string(StrategyModel), Label: "模型判断", Implemented: false,
			Description: "由小模型判断问题适合交给哪家。尚未接入，选中后实际仍按负载均衡执行。",
		},
	}
}

func (h *handler) getSettings(c *gin.Context) {
	adminSuccess(c, settingsView{
		RoutingStrategy: string(h.service.Strategy()),
		Strategies:      strategyOptions(),
	})
}

type updateSettingsRequest struct {
	RoutingStrategy *string `json:"routingStrategy"`
}

func (h *handler) updateSettings(c *gin.Context) {
	var req updateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		adminFailure(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if req.RoutingStrategy == nil {
		adminSuccess(c)
		return
	}

	strategy := RoutingStrategy(strings.TrimSpace(*req.RoutingStrategy))
	if !strategy.Valid() {
		adminFailure(c, http.StatusBadRequest, "未知的调度方式: "+*req.RoutingStrategy)
		return
	}
	if err := h.service.SetStrategy(c.Request.Context(), strategy); err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminSuccess(c)
}

// providerView is a provider row as the admin page sees it: never the raw key.
type providerView struct {
	Name          string  `json:"name"`
	DisplayName   string  `json:"displayName"`
	HomeURL       string  `json:"homeUrl"`
	DocsURL       string  `json:"docsUrl"`
	ConsoleURL    string  `json:"consoleUrl"`
	LogoURL       string  `json:"logoUrl"`
	Billing       string  `json:"billing"`
	Enabled       bool    `json:"enabled"`
	APIKeyMasked  string  `json:"apiKeyMasked"`
	APIKeyPresent bool    `json:"apiKeyPresent"`
	BaseURL       string  `json:"baseUrl"`
	Priority      int     `json:"priority"`
	Weight        int     `json:"weight"`
	RPS           float64 `json:"rps"`
	MonthlyQuota  int     `json:"monthlyQuota"`
	MonthlyUsed   int     `json:"monthlyUsed"`
	MonthlyCost   int     `json:"monthlyCostMicroUsd"`
	Extra         string  `json:"extra"`
	Health        string  `json:"health"`
}

func (h *handler) listProviders(c *gin.Context) {
	views, err := h.service.providerViews(c.Request.Context())
	if err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminSuccess(c, views)
}

// providerPatch is the admin update payload. APIKey is only written when a
// non-empty value arrives, so the masked value the page displays can be posted
// back without erasing the stored credential.
type providerPatch struct {
	DisplayName  *string  `json:"displayName"`
	Enabled      *bool    `json:"enabled"`
	APIKey       *string  `json:"apiKey"`
	BaseURL      *string  `json:"baseUrl"`
	Priority     *int     `json:"priority"`
	Weight       *int     `json:"weight"`
	RPS          *float64 `json:"rps"`
	MonthlyQuota *int     `json:"monthlyQuota"`
	Extra        *string  `json:"extra"`
}

func (h *handler) updateProvider(c *gin.Context) {
	name := strings.ToLower(strings.TrimSpace(c.Param("name")))
	var patch providerPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		adminFailure(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	updates := map[string]any{}
	if patch.DisplayName != nil {
		updates["display_name"] = strings.TrimSpace(*patch.DisplayName)
	}
	if patch.Enabled != nil {
		updates["enabled"] = *patch.Enabled
	}
	if patch.APIKey != nil {
		if key := strings.TrimSpace(*patch.APIKey); key != "" {
			updates["api_key"] = key
		}
	}
	if patch.BaseURL != nil {
		updates["base_url"] = strings.TrimSpace(*patch.BaseURL)
	}
	if patch.Priority != nil {
		updates["priority"] = *patch.Priority
	}
	if patch.Weight != nil {
		if *patch.Weight < 0 {
			adminFailure(c, http.StatusBadRequest, "权重不能为负数")
			return
		}
		updates["weight"] = *patch.Weight
	}
	if patch.RPS != nil {
		if *patch.RPS < 0 {
			adminFailure(c, http.StatusBadRequest, "限速不能为负数")
			return
		}
		updates["rps"] = *patch.RPS
	}
	if patch.MonthlyQuota != nil {
		if *patch.MonthlyQuota < 0 {
			adminFailure(c, http.StatusBadRequest, "月配额不能为负数")
			return
		}
		updates["monthly_quota"] = *patch.MonthlyQuota
	}
	if patch.Extra != nil {
		extra := strings.TrimSpace(*patch.Extra)
		if extra != "" && !json.Valid([]byte(extra)) {
			adminFailure(c, http.StatusBadRequest, "附加配置必须是合法 JSON")
			return
		}
		updates["extra"] = extra
	}
	if len(updates) == 0 {
		adminSuccess(c)
		return
	}

	if err := h.service.repo.updateProvider(c.Request.Context(), name, updates); err != nil {
		if err == ErrProviderNotFound {
			adminFailure(c, http.StatusNotFound, err.Error())
			return
		}
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.service.Reload(c.Request.Context()); err != nil {
		adminFailure(c, http.StatusInternalServerError, "配置已保存但重新加载失败: "+err.Error())
		return
	}
	adminSuccess(c)
}

// testProvider issues one real search so an operator can confirm a credential
// without waiting for an agent to hit the gateway.
func (h *handler) testProvider(c *gin.Context) {
	name := strings.ToLower(strings.TrimSpace(c.Param("name")))
	runtime := h.service.runtime(name)
	if runtime == nil {
		adminFailure(c, http.StatusNotFound, ErrProviderNotFound.Error())
		return
	}

	started := time.Now()
	result, err := runtime.provider.Search(c.Request.Context(), searchProbe())
	latency := int(time.Since(started) / time.Millisecond)
	if err != nil {
		adminSuccess(c, gin.H{"ok": false, "latencyMs": latency, "error": err.Error()})
		return
	}
	adminSuccess(c, gin.H{"ok": true, "latencyMs": latency, "resultCount": len(result.Results)})
}

// apiKeyView never carries the plaintext credential.
type apiKeyView struct {
	ID               int        `json:"id"`
	Name             string     `json:"name"`
	KeyPrefix        string     `json:"keyPrefix"`
	Enabled          bool       `json:"enabled"`
	AllowedProviders string     `json:"allowedProviders"`
	RateLimitPerMin  int        `json:"rateLimitPerMin"`
	MonthlyQuota     int        `json:"monthlyQuota"`
	MonthlyUsed      int        `json:"monthlyUsed"`
	ExpireAt         *time.Time `json:"expireAt"`
	LastUsedAt       *time.Time `json:"lastUsedAt"`
	Note             string     `json:"note"`
}

func (h *handler) listAPIKeys(c *gin.Context) {
	views, err := h.service.apiKeyViews(c.Request.Context())
	if err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminSuccess(c, views)
}

type createKeyRequest struct {
	Name             string `json:"name"`
	AllowedProviders string `json:"allowedProviders"`
	RateLimitPerMin  int    `json:"rateLimitPerMin"`
	MonthlyQuota     int    `json:"monthlyQuota"`
	ExpireDays       int    `json:"expireDays"`
	Note             string `json:"note"`
}

func (h *handler) createAPIKey(c *gin.Context) {
	var req createKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		adminFailure(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		adminFailure(c, http.StatusBadRequest, "名称不能为空")
		return
	}

	plain, err := GenerateAPIKey()
	if err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}

	key := APIKey{
		Name:             strings.TrimSpace(req.Name),
		KeyPrefix:        APIKeyPrefixOf(plain),
		KeyHash:          HashAPIKey(plain),
		Enabled:          true,
		AllowedProviders: normalizeAllowed(req.AllowedProviders),
		RateLimitPerMin:  req.RateLimitPerMin,
		MonthlyQuota:     req.MonthlyQuota,
		Note:             strings.TrimSpace(req.Note),
	}
	if req.ExpireDays > 0 {
		expire := time.Now().AddDate(0, 0, req.ExpireDays)
		key.ExpireAt = &expire
	}
	if err := h.service.repo.createAPIKey(c.Request.Context(), &key); err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}

	// The plaintext is returned exactly once and never stored.
	adminSuccess(c, gin.H{"id": key.ID, "name": key.Name, "apiKey": plain})
}

type updateKeyRequest struct {
	Name             *string `json:"name"`
	Enabled          *bool   `json:"enabled"`
	AllowedProviders *string `json:"allowedProviders"`
	RateLimitPerMin  *int    `json:"rateLimitPerMin"`
	MonthlyQuota     *int    `json:"monthlyQuota"`
	Note             *string `json:"note"`
}

func (h *handler) updateAPIKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		adminFailure(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req updateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		adminFailure(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.AllowedProviders != nil {
		updates["allowed_providers"] = normalizeAllowed(*req.AllowedProviders)
	}
	if req.RateLimitPerMin != nil {
		updates["rate_limit_per_min"] = *req.RateLimitPerMin
	}
	if req.MonthlyQuota != nil {
		updates["monthly_quota"] = *req.MonthlyQuota
	}
	if req.Note != nil {
		updates["note"] = strings.TrimSpace(*req.Note)
	}
	if len(updates) == 0 {
		adminSuccess(c)
		return
	}

	if err := h.service.updateAPIKey(c.Request.Context(), id, updates); err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminSuccess(c)
}

func (h *handler) deleteAPIKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		adminFailure(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := h.service.deleteAPIKey(c.Request.Context(), id); err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminSuccess(c)
}

func (h *handler) listLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	logs, total, err := h.service.repo.listLogs(c.Request.Context(), logFilter{
		Provider: strings.TrimSpace(c.Query("provider")),
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminSuccess(c, response.Page(total, int64(page), logs))
}

func (h *handler) stats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "1"))
	if days <= 0 || days > 90 {
		days = 1
	}
	stats, err := h.service.stats(c.Request.Context(), days)
	if err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminSuccess(c, stats)
}

func normalizeAllowed(raw string) string {
	parts := strings.Split(raw, ",")
	allowed := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.ToLower(strings.TrimSpace(part)); trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	return strings.Join(allowed, ",")
}
