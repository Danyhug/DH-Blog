package service

import (
	"errors"
	"mime/multipart"
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
func NewUploadService(localUploader Uploader, webdavUploader Uploader) *UploadService {
	return &UploadService{
		Uploaders: map[UploadType]Uploader{
			ArticleUpload: localUploader,
			WebdavUpload:  webdavUploader,
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
