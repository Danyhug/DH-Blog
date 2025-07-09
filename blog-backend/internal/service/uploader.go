package service

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"dh-blog/internal/config"
	"github.com/sirupsen/logrus"
)

// Uploader 定义了所有上传方式都必须实现的接口
type Uploader interface {
	Upload(file *multipart.FileHeader) (string, error)
}

// LocalUploader 实现了将文件保存到本地的策略
type LocalUploader struct {
	BaseDir string
	SubPath string
}

func NewLocalUploader(baseDir, subPath string) Uploader {
	return &LocalUploader{BaseDir: baseDir, SubPath: subPath}
}

func (u *LocalUploader) Upload(file *multipart.FileHeader) (string, error) {
	// 构建完整的上传目录路径
	uploadDir := filepath.Join(u.BaseDir, "upload", u.SubPath)

	// 确保上传目录存在
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return "", fmt.Errorf("创建上传目录失败: %w", err)
		}
	}

	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	// 构建文件保存路径
	dst := filepath.Join(uploadDir, fileName)

	// 保存文件
	if err := SaveUploadedFile(file, dst); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	return "uploads/" + filepath.Join(u.SubPath, fileName), nil
}

// WebdavUploader 实现了将文件保存到 WebDAV 的策略
type WebdavUploader struct {
	Config *config.Config
}

func NewWebdavUploader(cfg *config.Config) Uploader {
	return &WebdavUploader{Config: cfg}
}

func (u *WebdavUploader) Upload(file *multipart.FileHeader) (string, error) {
	// 这里需要实现连接 WebDAV 并上传文件的逻辑
	// 这部分需要 WebDAV 客户端库的支持，这里只是一个占位符
	return "", fmt.Errorf("WebDAV 上传功能尚未实现")
}

// SaveUploadedFile 是一个辅助函数，用于保存上传的文件
// 实际项目中，这个函数可能在 gin.Context 中，或者需要手动实现
func SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			logrus.Error("关闭文件失败: %w", err)
		}
	}(src)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			logrus.Error("关闭文件失败: %w", err)
		}
	}(out)

	_, err = out.ReadFrom(src)
	return err
}
