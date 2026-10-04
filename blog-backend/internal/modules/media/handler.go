package media

import (
	"errors"
	"mime"
	"net/http"
	"strings"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type handler struct {
	service *service
}

func newHandler(service *service) *handler {
	return &handler{service: service}
}

func currentUserID(c *gin.Context) uint64 {
	value, exists := c.Get("userID")
	if !exists {
		return 0
	}
	userID, _ := value.(uint64)
	return userID
}

// withUser rejects anonymous requests before the handler body runs.
func withUser(next func(c *gin.Context, userID uint64)) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := currentUserID(c)
		if userID == 0 {
			response.FailWithCode(c, http.StatusUnauthorized, "未授权")
			return
		}
		next(c, userID)
	}
}

// fail maps service errors onto the project's response convention:
// validation errors are 400, a missing file or playlist 404, the rest 500.
func fail(c *gin.Context, action string, err error) {
	switch {
	case errors.Is(err, errBadRequest):
		response.FailWithCode(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, errPlaylistNotFound), errors.Is(err, errMediaNotFound):
		response.FailWithCode(c, http.StatusNotFound, err.Error())
	default:
		logrus.Errorf("%s失败: %v", action, err)
		response.FailWithCode(c, http.StatusInternalServerError, action+"失败")
	}
}

func (h *handler) Videos(c *gin.Context, userID uint64) {
	library, err := h.service.Videos(c.Request.Context(), userID)
	if err != nil {
		fail(c, "加载影视库", err)
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(library))
}

func (h *handler) Music(c *gin.Context, userID uint64) {
	library, err := h.service.Music(c.Request.Context(), userID)
	if err != nil {
		fail(c, "加载音乐库", err)
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(library))
}

// Cover serves embedded artwork. It is loaded by <img>, so errors are plain
// status codes rather than JSON bodies.
func (h *handler) Cover(c *gin.Context, userID uint64) {
	picture, err := h.service.Cover(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		if errors.Is(err, errBadRequest) {
			c.Status(http.StatusBadRequest)
		} else {
			c.Status(http.StatusNotFound)
		}
		return
	}
	if picture == nil {
		c.Status(http.StatusNotFound)
		return
	}
	contentType := strings.ToLower(strings.TrimSpace(picture.MIMEType))
	if contentType == "image/jpg" {
		contentType = "image/jpeg"
	}
	if !strings.HasPrefix(contentType, "image/") || strings.Contains(contentType, "svg") {
		contentType = mime.TypeByExtension("." + strings.ToLower(picture.Ext))
	}
	if !strings.HasPrefix(contentType, "image/") || strings.Contains(contentType, "svg") {
		contentType = http.DetectContentType(picture.Data)
	}
	if !strings.HasPrefix(contentType, "image/") || strings.Contains(contentType, "svg") {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, contentType, picture.Data)
}

func (h *handler) Lyrics(c *gin.Context, userID uint64) {
	text, err := h.service.Lyrics(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		fail(c, "读取歌词", err)
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{"text": text}))
}

func (h *handler) SaveProgress(c *gin.Context, userID uint64) {
	var request struct {
		Position float64 `json:"position"`
		Duration float64 `json:"duration"`
		Started  bool    `json:"started"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.service.SaveProgress(c.Request.Context(), userID, c.Param("id"), request.Position, request.Duration, request.Started); err != nil {
		fail(c, "保存播放进度", err)
		return
	}
	c.JSON(http.StatusOK, response.Success())
}

func (h *handler) RemoveProgress(c *gin.Context, userID uint64) {
	if err := h.service.RemoveProgress(c.Request.Context(), userID, c.Param("id")); err != nil {
		fail(c, "移除播放记录", err)
		return
	}
	c.JSON(http.StatusOK, response.Success())
}

func (h *handler) ReportDurations(c *gin.Context, userID uint64) {
	var request struct {
		Items []DurationReport `json:"items"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.service.ReportDurations(c.Request.Context(), userID, request.Items); err != nil {
		fail(c, "保存媒体时长", err)
		return
	}
	c.JSON(http.StatusOK, response.Success())
}

func (h *handler) ListPlaylists(c *gin.Context, userID uint64) {
	playlists, err := h.service.ListPlaylists(c.Request.Context(), userID)
	if err != nil {
		fail(c, "读取歌单", err)
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(playlists))
}

func (h *handler) CreatePlaylist(c *gin.Context, userID uint64) {
	var input PlaylistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	playlist, err := h.service.CreatePlaylist(c.Request.Context(), userID, input)
	if err != nil {
		fail(c, "创建歌单", err)
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(playlist))
}

func (h *handler) UpdatePlaylist(c *gin.Context, userID uint64) {
	var input PlaylistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	playlist, err := h.service.UpdatePlaylist(c.Request.Context(), userID, c.Param("id"), input)
	if err != nil {
		fail(c, "更新歌单", err)
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(playlist))
}

func (h *handler) DeletePlaylist(c *gin.Context, userID uint64) {
	if err := h.service.DeletePlaylist(c.Request.Context(), userID, c.Param("id")); err != nil {
		fail(c, "删除歌单", err)
		return
	}
	c.JSON(http.StatusOK, response.Success())
}

func (h *handler) Settings(c *gin.Context, userID uint64) {
	settings, err := h.service.Settings(c.Request.Context(), userID)
	if err != nil {
		fail(c, "读取媒体库设置", err)
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(settings))
}

func (h *handler) SaveSettings(c *gin.Context, userID uint64) {
	var request struct {
		MusicFolderIDs []string `json:"music_folder_ids"`
		VideoFolderIDs []string `json:"video_folder_ids"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	settings, err := h.service.SaveSettings(c.Request.Context(), userID, request.MusicFolderIDs, request.VideoFolderIDs)
	if err != nil {
		fail(c, "保存媒体库设置", err)
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(settings))
}
