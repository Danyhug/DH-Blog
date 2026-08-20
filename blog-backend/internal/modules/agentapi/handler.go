package agentapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// grantHandler serves the admin surface for temporary edit grants. Every
// endpoint sits behind the AdminAPI JWT middleware; the agent-facing side
// reaches the same logic through GrantService, never through these routes.
type grantHandler struct {
	service *grantService
	events  ContentReporter
}

func newGrantHandler(service *grantService, events ContentReporter) *grantHandler {
	return &grantHandler{service: service, events: events}
}

type createGrantRequest struct {
	Note      string `json:"note"`
	ArticleID int    `json:"articleId"`
}

// createGrant issues a grant and returns the plaintext token exactly once.
// articleId binds the grant to a single article; 0 means any article, which the
// admin UI makes an explicit choice rather than the default.
func (h *grantHandler) createGrant(c *gin.Context) {
	var req createGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	grant, err := h.service.Grant(req.ArticleID, req.Note)
	if errors.Is(err, ErrGrantInvalidArticleID) {
		response.FailWithCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		// Grant failures other than the client-side articleId guard wrap
		// internal details (token generation, persistence), so the client gets
		// a stable Chinese message and the real error stays in the server log.
		logrus.Errorf("签发授权失败: %v", err)
		response.FailWithCode(c, http.StatusInternalServerError, "签发授权失败")
		return
	}
	// The feed is the only place the admin's own consent is visibly on the
	// record — the agent's later edits would otherwise look self-authorized.
	h.events.GrantIssued(grant.ArticleID, grant.Note)
	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"id":        grant.ID,
		"token":     grant.TokenPlain,
		"expireAt":  grant.ExpireAt.Format(grantTimeFormat),
		"articleId": grant.ArticleID,
		"note":      grant.Note,
	}))
}

// grantView is what the list endpoint exposes: the public prefix and the usage
// counters, never the plaintext. Times are formatted by hand because
// JSONTime's MarshalJSON only fires when the value is addressable, and values
// inside response maps and slices are not — the repo's date format must not
// silently degrade to RFC3339 on the wire.
type grantView struct {
	ID          int    `json:"id"`
	TokenPrefix string `json:"tokenPrefix"`
	ExpireAt    string `json:"expireAt"`
	ArticleID   int    `json:"articleId"`
	Revoked     bool   `json:"revoked"`
	UsedCount   int    `json:"usedCount"`
	LastUsedAt  any    `json:"lastUsedAt"`
	Note        string `json:"note"`
	CreateTime  string `json:"createTime"`
}

const grantTimeFormat = "2006-01-02 15:04:05"

func formatGrantTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(grantTimeFormat)
}

func toGrantView(grant EditGrant) grantView {
	return grantView{
		ID:          grant.ID,
		TokenPrefix: grant.TokenPrefix,
		ExpireAt:    grant.ExpireAt.Format(grantTimeFormat),
		ArticleID:   grant.ArticleID,
		Revoked:     grant.Revoked,
		UsedCount:   grant.UsedCount,
		LastUsedAt:  formatGrantTime(grant.LastUsedAt),
		Note:        grant.Note,
		CreateTime:  grant.CreatedAt.Format(grantTimeFormat),
	}
}

func (h *grantHandler) listGrants(c *gin.Context) {
	grants, err := h.service.List(h.service.now())
	if err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}
	views := make([]grantView, 0, len(grants))
	for _, grant := range grants {
		views = append(views, toGrantView(grant))
	}
	c.JSON(http.StatusOK, response.SuccessWithData(views))
}

// revealGrant hands the plaintext back so the same token can be copied again.
func (h *grantHandler) revealGrant(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	plain, err := h.service.Reveal(id)
	switch {
	case errors.Is(err, ErrGrantNotFound):
		response.FailWithCode(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrGrantRevoked):
		response.FailWithCode(c, http.StatusForbidden, err.Error())
	case err != nil:
		response.FailWithCode(c, http.StatusInternalServerError, err.Error())
	default:
		c.JSON(http.StatusOK, response.SuccessWithData(gin.H{"id": id, "token": plain}))
	}
}

func (h *grantHandler) revokeGrant(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := h.service.Revoke(id); err != nil {
		if errors.Is(err, ErrGrantNotFound) {
			response.FailWithCode(c, http.StatusNotFound, err.Error())
			return
		}
		response.FailWithCode(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, response.Success())
}
