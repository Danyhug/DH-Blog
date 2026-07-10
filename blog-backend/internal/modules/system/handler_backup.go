package system

import (
	"archive/zip"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *handler) getBackupDirs(c *gin.Context) {
	entries, err := os.ReadDir(h.storage.GetStoragePath())
	if err != nil {
		success(c, []backupDirInfo{})
		return
	}
	protected := map[string]bool{}
	for _, name := range h.storage.ProtectedDirectoryNames() {
		protected[name] = true
	}
	result := []backupDirInfo{}
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, backupDirInfo{entry.Name(), protected[entry.Name()]})
		}
	}
	success(c, result)
}
func (h *handler) backupData(c *gin.Context) {
	databasePath := h.databasePath
	if databasePath == "" {
		databasePath = filepath.Join(h.dataDir, "dhblog.db")
	}
	if _, err := os.Stat(databasePath); err != nil {
		failure(c, 400, fmt.Errorf("数据库文件不存在"))
		return
	}
	temp, err := os.CreateTemp("", "dhblog-backup-*.zip")
	if err != nil {
		failure(c, 500, err)
		return
	}
	path := temp.Name()
	defer os.Remove(path)
	writer := zip.NewWriter(temp)
	writer.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) { return flate.NewWriter(out, flate.BestCompression) })
	if err := addFileToZip(writer, databasePath, "dhblog.db"); err == nil {
		err = h.addBackupDirectories(writer, c.Query("mode"), c.Query("dirs"))
	}
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		failure(c, 500, err)
		return
	}
	name := fmt.Sprintf("dhblog-%s.zip", time.Now().Format("20060102150405"))
	if c.Query("mode") == "full" {
		name = fmt.Sprintf("dhblog-full-%s.zip", time.Now().Format("20060102150405"))
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", name))
	c.File(path)
}
func (h *handler) addBackupDirectories(writer *zip.Writer, mode, dirs string) error {
	root := h.storage.GetStoragePath()
	if mode == "full" {
		return addDirToZip(writer, root, "webdav")
	}
	names := h.storage.ProtectedDirectoryNames()
	if dirs != "" {
		names = strings.Split(dirs, ",")
	}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err := validateBackupDirectoryName(name); err != nil {
			return err
		}
		path := filepath.Join(root, name)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			if err := addDirToZip(writer, path, filepath.Join("webdav", name)); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBackupDirectoryName(name string) error {
	if filepath.IsAbs(name) || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("无效的备份目录: %s", name)
	}
	return nil
}

func addFileToZip(writer *zip.Writer, source, name string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	entry, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, file)
	return err
}
func addDirToZip(writer *zip.Writer, root, zipRoot string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return addFileToZip(writer, path, filepath.Join(zipRoot, relative))
	})
}
