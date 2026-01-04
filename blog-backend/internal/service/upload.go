package service

import (
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"dh-blog/internal/config"

	"github.com/sirupsen/logrus"
)

// FileService 定义了基本的文件服务接口
type FileService interface {
	Upload(fileName string, fileContent []byte)
	Download(fileName string)
}

// Uploader 定义了所有上传方式都必须实现的接口
type Uploader interface {
	Upload(file *multipart.FileHeader) (string, error)
}

// UploadType 定义了上传类型的枚举
type UploadType string

const (
	ArticleUpload UploadType = "blog"
	WebdavUpload  UploadType = "webdav"
)

// UploadService 是我们的主服务，它管理着所有具体的 Uploader
type UploadService struct {
	Uploaders map[UploadType]Uploader
}

// NewUploadService 创建并初始化 UploadService
// 这里通过依赖注入，传入所有具体的 uploader 实现
func NewUploadService(cfg *config.Config, dataDir string, fileService IFileService) *UploadService {
	return &UploadService{
		Uploaders: map[UploadType]Uploader{
			ArticleUpload: NewBlogImageUploader(fileService),
			WebdavUpload:  NewWebdavUploader(cfg),
		},
	}
}

// UploadFile 是暴露给 Handler 的统一方法
func (s *UploadService) UploadFile(file *multipart.FileHeader, uploadType UploadType) (string, error) {
	uploader, ok := s.Uploaders[uploadType]
	if !ok {
		return "", errors.New("unsupported upload type")
	}
	return uploader.Upload(file)
}

// LocalUploader 实现了将文件保存到本地的策略
type LocalUploader struct {
	BaseDir string
	SubPath string
}

// NewLocalUploader 创建本地上传器
func NewLocalUploader(baseDir, subPath string) Uploader {
	return &LocalUploader{BaseDir: baseDir, SubPath: subPath}
}

// Upload 实现Uploader接口的上传方法
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

// BlogImageUploader 将博客图片保存到WebDAV存储目录
type BlogImageUploader struct {
	fileService IFileService
}

// NewBlogImageUploader 创建博客图片上传器
func NewBlogImageUploader(fileService IFileService) Uploader {
	return &BlogImageUploader{fileService: fileService}
}

// Upload 实现Uploader接口的上传方法
func (u *BlogImageUploader) Upload(file *multipart.FileHeader) (string, error) {
	// 获取WebDAV存储路径
	storagePath := u.fileService.GetStoragePath()
	if storagePath == "" {
		return "", fmt.Errorf("WebDAV存储路径未配置")
	}

	// 构建博客图片目录路径
	blogImagesDir := filepath.Join(storagePath, "blog-images")

	// 确保目录存在
	if err := os.MkdirAll(blogImagesDir, 0755); err != nil {
		return "", fmt.Errorf("创建博客图片目录失败: %w", err)
	}

	// 生成文件名
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	dst := filepath.Join(blogImagesDir, fileName)

	// 保存文件
	if err := SaveUploadedFile(file, dst); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	// 返回公开访问URL
	return "blog-images/" + fileName, nil
}

// WebdavUploader 实现了将文件保存到 WebDAV 的策略
type WebdavUploader struct {
	Config *config.Config
}

// NewWebdavUploader 创建WebDAV上传器
func NewWebdavUploader(cfg *config.Config) Uploader {
	return &WebdavUploader{Config: cfg}
}

// Upload 实现Uploader接口的上传方法
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
