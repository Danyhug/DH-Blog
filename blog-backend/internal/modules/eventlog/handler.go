package eventlog

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler { return &handler{service: service} }

// helloPayload tells a freshly connected page where the feed stands, so it can
// decide whether to replay or just start listening.
type helloPayload struct {
	Cursor     int64  `json:"cursor"`
	LogCursor  int64  `json:"logCursor"`
	Clients    int    `json:"clients"`
	ServerTime string `json:"serverTime"`
}

// logs serves the console's scrollback. The socket only carries lines produced
// from now on, so without this the console starts empty on every page load.
func (h *handler) logs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	lines := h.service.LogSnapshot(limit)
	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"lines":  lines,
		"cursor": h.service.LogCursor(),
	}))
}

// list serves the paged history that backs the page's first render. The socket
// only carries what happens from now on, so this is what makes the feed
// survive a reload.
func (h *handler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "30"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 30
	}

	events, total, err := h.service.repo.list(c.Request.Context(), listFilter{
		Source:   strings.TrimSpace(c.Query("source")),
		Kind:     strings.TrimSpace(c.Query("kind")),
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, "获取后台事件失败")
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), events)))
}

// replay returns everything after a client's last seen id. A page that lost
// its socket calls this on reconnect instead of reloading the whole feed.
func (h *handler) replay(c *gin.Context) {
	sinceID, err := strconv.ParseInt(c.DefaultQuery("sinceId", "0"), 10, 64)
	if err != nil || sinceID < 0 {
		response.FailWithCode(c, http.StatusBadRequest, "sinceId 参数无效")
		return
	}

	events, err := h.service.repo.since(c.Request.Context(), sinceID, replayLimit)
	if err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, "获取后台事件失败")
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"events": events,
		"cursor": h.service.Cursor(),
		// Truncated tells the client its replay window was too wide and the
		// list it just got is not the whole gap.
		"truncated": len(events) == replayLimit,
	}))
}

// stream upgrades to a WebSocket. Auth already happened in the JWT middleware
// this route is mounted behind, which reads the token from the query string —
// the browser's WebSocket API cannot set an Authorization header.
func (h *handler) stream(c *gin.Context) {
	h.service.hub.serve(c.Writer, c.Request, func(clients int) any {
		return helloPayload{
			Cursor:     h.service.Cursor(),
			LogCursor:  h.service.LogCursor(),
			Clients:    clients,
			ServerTime: time.Now().Format("2006-01-02 15:04:05"),
		}
	})
}
