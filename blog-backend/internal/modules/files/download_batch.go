package files

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"dh-blog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// maxBatchDownloadFiles caps one archive request so a stray selection cannot
// pin a worker on thousands of files. 选中文件夹时按展开后的文件数计算。
const maxBatchDownloadFiles = 500

// errBatchTooManyFiles 是超限的哨兵错误：展开文件夹后才知道总数，
// 它要映射成 400 而不是跟"文件不存在"一起走 404。
var errBatchTooManyFiles = fmt.Errorf("单次最多打包 %d 个文件", maxBatchDownloadFiles)

// batchDownloadItem 是压缩包里的一个条目。
// EntryName 是包内相对路径（始终用 / 分隔），选中文件夹时用它还原目录结构；
// IsDir 的条目是空目录占位，没有对应的磁盘文件。
type batchDownloadItem struct {
	File      *File
	EntryName string
	IsDir     bool
}

// DownloadBatch 打包下载
// @Summary 批量打包下载文件
// @Description 将多个文件流式打包成 zip 返回，避免浏览器拦截并发的单文件下载
// @Tags 文件
// @Produce octet-stream
// @Param ids query string true "逗号分隔的文件ID列表"
// @Success 200 {file} file "zip 内容"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 404 {object} response.Response "文件不存在"
// @Router /api/files/download-batch [get]
func (h *handler) DownloadBatch(c *gin.Context) {
	userID := h.getCurrentUserID(c)
	if userID == 0 {
		response.FailWithCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	ids := parseBatchIDs(c.Query("ids"))
	if len(ids) == 0 {
		response.FailWithCode(c, http.StatusBadRequest, "文件ID不能为空")
		return
	}
	if len(ids) > maxBatchDownloadFiles {
		response.FailWithCode(c, http.StatusBadRequest, errBatchTooManyFiles.Error())
		return
	}

	// 先解析全部条目：任何一个不可下载都在写出响应体之前失败，
	// 否则错误只能以半截 zip 的形式暴露给浏览器。选中的文件夹在这一步递归展开。
	items, err := h.fileService.ResolveBatchDownloadInfo(c.Request.Context(), userID, ids)
	if errors.Is(err, errBatchTooManyFiles) {
		response.FailWithCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		response.FailWithCode(c, http.StatusNotFound, fmt.Sprintf("获取文件失败: %v", err))
		return
	}
	if len(items) == 0 {
		response.FailWithCode(c, http.StatusBadRequest, "没有可下载的文件")
		return
	}

	archiveName := c.Query("name")
	if archiveName == "" {
		archiveName = "download.zip"
	} else if !strings.HasSuffix(archiveName, ".zip") {
		archiveName += ".zip"
	}

	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": archiveName}))
	c.Header("Content-Type", "application/zip")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Writer.WriteHeader(http.StatusOK)

	writer := zip.NewWriter(c.Writer)
	for _, item := range items {
		if err := writeZipEntry(writer, item); err != nil {
			// 响应体已经开始输出，只能中断连接让浏览器把 zip 判为损坏。
			logrus.Errorf("打包下载写入失败 entry=%s: %v", item.EntryName, err)
			_ = writer.Close()
			c.Abort()
			return
		}
		// 每个条目后主动 flush，避免大文件之间反向代理长时间无数据可读而超时。
		if err := writer.Flush(); err != nil {
			logrus.Errorf("打包下载 flush 失败: %v", err)
			c.Abort()
			return
		}
		c.Writer.Flush()
	}
	if err := writer.Close(); err != nil {
		logrus.Errorf("打包下载收尾失败: %v", err)
	}
}

// writeZipEntry streams one file into the archive, keeping memory flat
// regardless of file size.
func writeZipEntry(writer *zip.Writer, item batchDownloadItem) error {
	// 目录项只有一个以 / 结尾的名字，没有内容。
	if item.IsDir {
		_, err := writer.CreateHeader(&zip.FileHeader{Name: item.EntryName, Method: zip.Store})
		return err
	}

	source, err := os.Open(item.File.StoragePath)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	header := &zip.FileHeader{
		Name: item.EntryName,
		// Deflate 对已压缩的媒体文件收益极低，却要吃满 CPU；这里统一存储即可。
		Method: zip.Store,
	}
	// 带上磁盘上的修改时间，否则解压出来全是 zip 的零值时间（1979-11-30）。
	if stat, statErr := source.Stat(); statErr == nil {
		header.Modified = stat.ModTime()
	}
	entry, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, source)
	return err
}

func parseBatchIDs(raw string) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

// uniqueEntryName keeps duplicate names from colliding inside the archive and
// strips path separators, so the result is always a single path segment.
func uniqueEntryName(used map[string]struct{}, name string) string {
	name = strings.ReplaceAll(strings.ReplaceAll(name, "\\", "_"), "/", "_")
	if name == "" || name == "." || name == ".." {
		name = "unnamed"
	}
	candidate := name
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		if _, exists := used[candidate]; !exists {
			used[candidate] = struct{}{}
			return candidate
		}
		candidate = base + "(" + strconv.Itoa(i) + ")" + ext
	}
}
