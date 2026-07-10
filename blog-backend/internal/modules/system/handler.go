package system

import (
	"net/http"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

type handler struct {
	service      *service
	storage      StorageRuntime
	dataDir      string
	databasePath string
}

func newHandler(service *service, storage StorageRuntime, dataDir, databasePath string) *handler {
	return &handler{service: service, storage: storage, dataDir: dataDir, databasePath: databasePath}
}

func success(c *gin.Context, data ...any) {
	if len(data) == 0 {
		c.JSON(http.StatusOK, response.Success())
		return
	}
	c.JSON(http.StatusOK, response.SuccessWithData(data[0]))
}
func failure(c *gin.Context, status int, err error) { c.JSON(status, response.Error(err.Error())) }
