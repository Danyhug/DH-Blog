package files

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
)

// UploadChunk 上传分片
// @Summary 上传文件分片
// @Description 上传文件的一个分片
// @Tags 文件上传
// @Accept multipart/form-data
// @Produce json
// @Param uploadId formData string true "上传会话ID"
// @Param chunkIndex formData int true "分片索引"
// @Param chunk formData file true "分片数据"
// @Success http.StatusOK {object} map[string]interface{} "{"success": true}"
// @Failure http.StatusOK {object} map[string]string "{"error": "错误信息"}"
// @Router /files/upload/chunk [post]
func (h *chunkUploadHandler) UploadChunk(c *gin.Context) {
	uploadId := c.PostForm("uploadId")
	chunkIndexStr := c.PostForm("chunkIndex")

	if uploadId == "" || chunkIndexStr == "" {
		c.JSON(http.StatusOK, response.Error("uploadId和chunkIndex不能为空"))
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		c.JSON(http.StatusOK, response.Error("chunkIndex格式错误"))
		return
	}

	// 获取配置的存储路径
	storagePath := h.fileService.GetStoragePath()
	baseDir := storagePath
	tempDir := filepath.Join(baseDir, "temp", uploadId)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, response.Error("上传会话不存在"))
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusOK, response.Error("获取分片数据失败"))
		return
	}

	// 保存分片文件
	chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", chunkIndex))
	if err := c.SaveUploadedFile(file, chunkFile); err != nil {
		c.JSON(http.StatusOK, response.Error("保存分片失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"success":    true,
		"chunkIndex": chunkIndex,
		"uploadId":   uploadId,
	}))
}
