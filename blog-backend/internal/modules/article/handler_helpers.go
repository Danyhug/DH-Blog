package article

import (
	"fmt"
	"net/http"
	"strconv"

	"dh-blog/internal/model"
	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) getID(c *gin.Context, key string) (int, error) {
	id, err := strconv.Atoi(c.Param(key))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidID, err)
	}
	return id, nil
}

func (h *Handler) bindJSON(c *gin.Context, value interface{}) error {
	if err := c.ShouldBindJSON(value); err != nil {
		return fmt.Errorf("%w: %v", ErrParamBinding, err)
	}
	return nil
}

func (h *Handler) getPageRequest(c *gin.Context) (*model.PageRequest, error) {
	var req model.PageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if err := c.ShouldBindQuery(&req); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPageParamBinding, err)
		}
	}
	if req.PageNum == 0 {
		req.PageNum = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}
	return &req, nil
}

// These response helpers intentionally preserve the article controller's
// historical status-code behavior.
func (h *Handler) Error(c *gin.Context, err error) {
	c.JSON(http.StatusOK, response.Error(err.Error()))
}

func (h *Handler) Success(c *gin.Context) {
	c.JSON(http.StatusOK, response.Success())
}

func (h *Handler) SuccessWithData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response.SuccessWithData(data))
}

func (h *Handler) SuccessWithMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, response.SuccessWithData(message))
}

func (h *Handler) SuccessWithPage(c *gin.Context, data interface{}, total int64, page int) {
	c.JSON(http.StatusOK, response.SuccessWithData(response.Page(total, int64(page), data)))
}
