package service

import (
	"errors"
	"mime/multipart"

	"dh-blog/internal/config"
)

// UploadService 是我们的主服务，它管理着所有具体的 Uploader
type UploadService struct {
	Uploaders map[UploadType]Uploader
}

// UploadType 定义了上传类型的枚举
type UploadType string

const (
	ArticleUpload UploadType = "blog"
	WebdavUpload  UploadType = "webdav"
)

// NewUploadService 创建并初始化 UploadService
// 这里通过依赖注入，传入所有具体的 uploader 实现
func NewUploadService(cfg *config.Config, dataDir string) *UploadService {
	return &UploadService{
		Uploaders: map[UploadType]Uploader{
			ArticleUpload: NewLocalUploader(dataDir, cfg.Upload.Local.Path),
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
