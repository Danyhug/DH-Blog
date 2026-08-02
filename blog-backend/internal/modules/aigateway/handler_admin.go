package aigateway

import (
	"encoding/json"
	"errors"
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
	// 没接入的调度方式一律拒收。之前允许保存再回落到负载均衡，等于后台显示的和
	// 实际执行的不是一回事，还不如直接不让选。
	if !strategy.Implemented() {
		adminFailure(c, http.StatusBadRequest, "该调度方式尚未接入，暂时不能启用")
		return
	}
	if err := h.service.SetStrategy(c.Request.Context(), strategy); err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminSuccess(c)
}

// providerKeyView is one upstream credential as the admin page sees it.
type providerKeyView struct {
	ID         int        `json:"id"`
	Label      string     `json:"label"`
	Masked     string     `json:"masked"`
	Enabled    bool       `json:"enabled"`
	Status     string     `json:"status"`
	LastError  string     `json:"lastError"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	DisabledAt *time.Time `json:"disabledAt"`
	// InRotation folds enabled + status + the monthly self-recovery into the
	// one thing the page actually needs to show.
	InRotation bool `json:"inRotation"`
}

// providerView is a provider row as the admin page sees it: never the raw key.
type providerView struct {
	Name         string            `json:"name"`
	DisplayName  string            `json:"displayName"`
	HomeURL      string            `json:"homeUrl"`
	DocsURL      string            `json:"docsUrl"`
	ConsoleURL   string            `json:"consoleUrl"`
	LogoURL      string            `json:"logoUrl"`
	Billing      string            `json:"billing"`
	Enabled      bool              `json:"enabled"`
	Keys         []providerKeyView `json:"keys"`
	ActiveKeys   int               `json:"activeKeys"`
	BaseURL      string            `json:"baseUrl"`
	Priority     int               `json:"priority"`
	Weight       int               `json:"weight"`
	RPS          float64           `json:"rps"`
	MonthlyQuota int               `json:"monthlyQuota"`
	MonthlyUsed  int               `json:"monthlyUsed"`
	MonthlyCost  int               `json:"monthlyCostMicroUsd"`
	Extra        string            `json:"extra"`
	Health       string            `json:"health"`
}

func (h *handler) listProviders(c *gin.Context) {
	views, err := h.service.providerViews(c.Request.Context())
	if err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	adminSuccess(c, views)
}

// providerPatch is the admin update payload. Credentials are deliberately not
// part of it: they live in their own table and are managed through the
// /keys endpoints, so there is exactly one way to add or retire one.
type providerPatch struct {
	DisplayName  *string  `json:"displayName"`
	Enabled      *bool    `json:"enabled"`
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
// without waiting for an agent to hit the gateway. The body is optional: send a
// draft key to check it before saving, a keyId to check one stored credential,
// or nothing to check whichever credential is next in rotation.
func (h *handler) testProvider(c *gin.Context) {
	name := strings.ToLower(strings.TrimSpace(c.Param("name")))
	var probe ProviderProbe
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&probe); err != nil {
			adminFailure(c, http.StatusBadRequest, "参数错误: "+err.Error())
			return
		}
	}

	result, err := h.service.TestProvider(c.Request.Context(), name, probe)
	if err != nil {
		adminFailure(c, providerErrorStatus(err), providerErrorMessage(err))
		return
	}
	adminSuccess(c, result)
}

func providerErrorStatus(err error) int {
	switch {
	case errors.Is(err, ErrProviderNotFound), errors.Is(err, ErrProviderKeyNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func providerErrorMessage(err error) string {
	if errors.Is(err, ErrProviderKeyNotFound) {
		return "该供应商还没有可用的密钥，请先填一个再测试"
	}
	return err.Error()
}

// providerKeyPayload creates or updates one upstream credential.
type providerKeyPayload struct {
	Label   *string `json:"label"`
	APIKey  *string `json:"apiKey"`
	Enabled *bool   `json:"enabled"`
	// Revive puts a parked credential back into rotation and clears the reason.
	Revive bool `json:"revive"`
}

func (h *handler) createProviderKey(c *gin.Context) {
	name := strings.ToLower(strings.TrimSpace(c.Param("name")))
	if _, err := h.service.repo.providerByName(c.Request.Context(), name); err != nil {
		adminFailure(c, providerErrorStatus(err), err.Error())
		return
	}

	var payload providerKeyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		adminFailure(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	secret := ""
	if payload.APIKey != nil {
		secret = strings.TrimSpace(*payload.APIKey)
	}
	if secret == "" {
		adminFailure(c, http.StatusBadRequest, "密钥不能为空")
		return
	}

	key := ProviderKey{
		Provider: name,
		Label:    labelOrDefault(payload.Label),
		APIKey:   secret,
		Enabled:  payload.Enabled == nil || *payload.Enabled,
		Status:   ProviderKeyActive,
	}
	if err := h.service.repo.createProviderKey(c.Request.Context(), &key); err != nil {
		adminFailure(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.reloadAfterChange(c, key.ID)
}

func (h *handler) updateProviderKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		adminFailure(c, http.StatusBadRequest, "非法的密钥 ID")
		return
	}
	var payload providerKeyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		adminFailure(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	updates := map[string]any{}
	if payload.Label != nil {
		updates["label"] = strings.TrimSpace(*payload.Label)
	}
	if payload.APIKey != nil {
		if secret := strings.TrimSpace(*payload.APIKey); secret != "" {
			// 换了新密钥就当它是好的，顺手清掉上一把留下的停用原因
			updates["api_key"] = secret
			updates["status"] = ProviderKeyActive
			updates["last_error"] = ""
			updates["disabled_at"] = nil
		}
	}
	if payload.Enabled != nil {
		updates["enabled"] = *payload.Enabled
	}
	if payload.Revive {
		updates["status"] = ProviderKeyActive
		updates["last_error"] = ""
		updates["disabled_at"] = nil
	}
	if len(updates) == 0 {
		adminSuccess(c)
		return
	}

	if err := h.service.repo.updateProviderKey(c.Request.Context(), id, updates); err != nil {
		adminFailure(c, providerErrorStatus(err), err.Error())
		return
	}
	h.reloadAfterChange(c, id)
}

func (h *handler) deleteProviderKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		adminFailure(c, http.StatusBadRequest, "非法的密钥 ID")
		return
	}
	if err := h.service.repo.deleteProviderKey(c.Request.Context(), id); err != nil {
		adminFailure(c, providerErrorStatus(err), err.Error())
		return
	}
	h.reloadAfterChange(c, id)
}

// reloadAfterChange rebuilds the runtimes so a credential change takes effect
// on the next request instead of after a restart.
func (h *handler) reloadAfterChange(c *gin.Context, id int) {
	if err := h.service.Reload(c.Request.Context()); err != nil {
		adminFailure(c, http.StatusInternalServerError, "已保存但重新加载失败: "+err.Error())
		return
	}
	adminSuccess(c, gin.H{"id": id})
}

func labelOrDefault(label *string) string {
	if label == nil {
		return ""
	}
	return strings.TrimSpace(*label)
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
		KeyPlain:         plain,
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

// revealAPIKey hands the plaintext back so the same key can be copied again.
//
// The gateway stores it alongside the hash on purpose: a key that can only be
// copied once means a mislaid key forces a reissue plus reconfiguring every
// agent that used it. Verification still runs off the hash, and this endpoint
// is the only reader of the column — the list endpoint never carries it.
func (h *handler) revealAPIKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		adminFailure(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	key, err := h.service.repo.apiKeyByID(c.Request.Context(), id)
	if err != nil {
		adminFailure(c, http.StatusNotFound, err.Error())
		return
	}
	if key.KeyPlain == "" {
		adminFailure(c, http.StatusGone, "这把 Key 签发于只存哈希的版本，明文无法找回，请重新签发")
		return
	}
	adminSuccess(c, gin.H{"id": key.ID, "name": key.Name, "apiKey": key.KeyPlain})
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
