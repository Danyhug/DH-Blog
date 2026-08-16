package files

import (
	"fmt"
	"io"
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
	if err := validateUploadID(uploadId); err != nil {
		c.JSON(http.StatusOK, response.Error(err.Error()))
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		c.JSON(http.StatusOK, response.Error("chunkIndex格式错误"))
		return
	}

	baseDir := h.fileService.GetStoragePath()
	tempDir := filepath.Join(baseDir, tempDirName, uploadId)
	info, err := readChunkSessionInfo(tempDir)
	if err != nil {
		c.JSON(http.StatusOK, response.Error("上传会话不存在"))
		return
	}
	// 旧格式会话没有归属信息，续传会破坏新版校验，拒绝并引导重新初始化。
	if !info.hasUserID {
		c.JSON(http.StatusOK, response.Error("上传会话已失效，请重新初始化"))
		return
	}
	if !info.belongsTo(h.getUserID(c)) {
		c.JSON(http.StatusForbidden, response.Error("无权操作此上传会话"))
		return
	}
	if chunkIndex < 0 || chunkIndex >= info.totalChunks {
		c.JSON(http.StatusOK, response.Error("chunkIndex超出范围"))
		return
	}

	file, err := c.FormFile("chunk")
	if err != nil {
		c.JSON(http.StatusOK, response.Error("获取分片数据失败"))
		return
	}

	// 声明的大小超过分片上限直接拒绝，省掉落盘再删的开销。
	if file.Size > info.chunkSize {
		c.JSON(http.StatusOK, response.Error("分片大小超过限制"))
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusOK, response.Error("读取分片数据失败"))
		return
	}
	defer func() { _ = src.Close() }()

	chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", chunkIndex))
	dst, err := os.Create(chunkFile)
	if err != nil {
		c.JSON(http.StatusOK, response.Error("保存分片失败"))
		return
	}

	// 以会话分片大小为硬上限流式拷贝，防止客户端谎报 Content-Length
	// 把任意大小的内容写进临时目录。
	written, err := io.CopyN(dst, src, info.chunkSize+1)
	closeErr := dst.Close()
	if err != nil && err != io.EOF {
		_ = os.Remove(chunkFile)
		c.JSON(http.StatusOK, response.Error("保存分片失败"))
		return
	}
	if written > info.chunkSize {
		_ = os.Remove(chunkFile)
		c.JSON(http.StatusOK, response.Error("分片大小超过限制"))
		return
	}
	if closeErr != nil {
		_ = os.Remove(chunkFile)
		c.JSON(http.StatusOK, response.Error("保存分片失败"))
		return
	}

	c.JSON(http.StatusOK, response.SuccessWithData(gin.H{
		"success":    true,
		"chunkIndex": chunkIndex,
		"uploadId":   uploadId,
	}))
}
