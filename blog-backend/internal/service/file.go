package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"

	"github.com/sirupsen/logrus"
)

type fileService struct {
	repo     repository.IFileRepository // 文件存储库
	filePath string                     // 实际存储文件的基础路径
}

const (
	defaultFilePath = `/Users/danyhug/GolandProjects/DH-Blog/blog-deploy/backend`
)

// IFileService 定义了网盘核心功能的业务逻辑合同 (MVP版本)
type IFileService interface {

	// ListFiles 1. 获取文件列表
	// 查看一个文件夹里有什么。这是最基础的浏览功能。
	ListFiles(ctx context.Context, userID uint64, parentID string) ([]*model.File, error)

	// CreateFolder 2. 创建文件夹
	// 组织文件的基本手段。
	CreateFolder(ctx context.Context, userID uint64, parentID string, folderName string) (*model.File, error)

	// UploadFile 3. 文件上传 (简化版)
	// 我们暂时不考虑"秒传"，先实现最直接的上传流程。
	// fileContent 是文件的二进制流。
	UploadFile(ctx context.Context, userID uint64, parentID string, fileName string, fileSize int64, fileContent io.Reader) (*model.File, error)

	// GetDownloadInfo 4. 获取下载信息
	// 用户需要能把自己上传的东西下载回来。
	// Service层返回文件元信息，由Handler层去处理真正的文件流响应。
	GetDownloadInfo(ctx context.Context, userID uint64, fileID string) (*model.File, error)

	// RenameFile 5. 重命名文件或文件夹
	// 非常高频且基础的整理操作。
	RenameFile(ctx context.Context, userID uint64, fileID string, newName string) error

	// DeleteFile 6. 删除文件或文件夹 (简化版)
	// 先实现直接删除（硬删除），回收站功能作为未来的增强项。
	DeleteFile(ctx context.Context, userID uint64, fileID string) error
}

// NewFileService 创建新的文件服务
// 参数:
//   - repo: 文件存储库接口
//   - filePath: 可选，实际文件存储路径
//
// 返回:
//   - IFileService: 文件服务接口
func NewFileService(repo repository.IFileRepository, filePath ...string) IFileService {
	storagePath := defaultFilePath
	if len(filePath) > 0 && filePath[0] != "" {
		storagePath = filePath[0]
	}

	// 确保存储路径存在
	if err := os.MkdirAll(storagePath, os.ModePerm); err != nil {
		logrus.Warnf("创建文件存储路径失败: %v，将使用默认路径", err)
	}

	return &fileService{
		repo:     repo,
		filePath: storagePath,
	}
}

// 生成存储路径
func (s *fileService) getStoragePath(userID uint64, parentID string, fileName string) string {
	// 使用用户ID和父目录创建唯一的存储路径
	userPath := fmt.Sprintf("user_%d", userID)
	relativePath := filepath.Join(userPath, parentID)

	// 创建完整路径
	fullPath := filepath.Join(s.filePath, relativePath)

	// 确保目录存在
	_ = os.MkdirAll(fullPath, os.ModePerm)

	// 如果提供了文件名，则返回包含文件名的完整路径
	if fileName != "" {
		return filepath.Join(fullPath, fileName)
	}

	return relativePath
}

func (s *fileService) ListFiles(ctx context.Context, userID uint64, parentID string) ([]*model.File, error) {
	// 使用存储库查询数据库
	files, err := s.repo.ListByParentID(ctx, userID, parentID)
	if err != nil {
		logrus.Errorf("列出文件失败: %v", err)
		return nil, fmt.Errorf("列出文件失败")
	}

	return files, nil
}

func (s *fileService) CreateFolder(ctx context.Context, userID uint64, parentID string, folderName string) (*model.File, error) {
	// 检查是否已存在同名文件夹
	existing, err := s.repo.FindByUserIDAndName(ctx, userID, parentID, folderName)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("同名文件夹已存在")
	}

	// 创建文件夹记录
	folder := &model.File{
		UserID:   userID,
		ParentID: parentID,
		Name:     folderName,
		IsFolder: true,
		Size:     0,
	}

	// 保存到数据库
	err = s.repo.Create(ctx, folder)
	if err != nil {
		logrus.Errorf("创建文件夹失败: %v", err)
		return nil, fmt.Errorf("创建文件夹失败")
	}

	// 创建实际的文件系统目录
	storagePath := s.getStoragePath(userID, parentID, folderName)
	err = os.MkdirAll(storagePath, os.ModePerm)
	if err != nil {
		logrus.Warnf("创建实际文件夹失败，但数据库记录已创建: %v", err)
		// 即使物理目录创建失败，我们仍然返回数据库记录
	}

	return folder, nil
}

func (s *fileService) UploadFile(ctx context.Context, userID uint64, parentID string, fileName string, fileSize int64, fileContent io.Reader) (*model.File, error) {
	// 检查是否已存在同名文件
	existing, err := s.repo.FindByUserIDAndName(ctx, userID, parentID, fileName)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("同名文件已存在")
	}

	// 创建物理文件存储路径
	relativePath := s.getStoragePath(userID, parentID, "")
	storagePath := filepath.Join(s.filePath, relativePath, fileName)

	// 创建目标文件
	outFile, err := os.Create(storagePath)
	if err != nil {
		logrus.Errorf("创建文件失败: %v", err)
		return nil, fmt.Errorf("创建文件失败")
	}
	defer outFile.Close()

	// 写入文件内容
	written, err := io.Copy(outFile, fileContent)
	if err != nil {
		logrus.Errorf("写入文件内容失败: %v", err)
		// 删除可能已创建的文件
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("写入文件内容失败")
	}

	// 创建文件数据库记录
	file := &model.File{
		UserID:      userID,
		ParentID:    parentID,
		Name:        fileName,
		IsFolder:    false,
		Size:        written,
		StoragePath: filepath.Join(relativePath, fileName),
		// 确定MIME类型
		MimeType: getMimeType(fileName),
	}

	// 保存到数据库
	err = s.repo.Create(ctx, file)
	if err != nil {
		logrus.Errorf("保存文件记录失败: %v", err)
		// 删除已创建的文件
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("保存文件记录失败")
	}

	return file, nil
}

func (s *fileService) GetDownloadInfo(ctx context.Context, userID uint64, fileID string) (*model.File, error) {
	// 解析fileID，在MVP版本中fileID可以是文件记录ID或文件路径
	var file *model.File
	var err error

	// 尝试按ID查找文件
	id, parseErr := parseFileID(fileID)
	if parseErr == nil {
		file, err = s.repo.FindByID(ctx, id)
	} else {
		// 如果ID解析失败，按路径查找
		file, err = s.repo.FindByPath(ctx, userID, fileID)
	}

	if err != nil {
		logrus.Errorf("查找文件失败: %v", err)
		return nil, fmt.Errorf("文件不存在")
	}

	// 确认是否为用户自己的文件
	if file.UserID != userID {
		return nil, fmt.Errorf("无权访问此文件")
	}

	// 检查是否是文件夹
	if file.IsFolder {
		return nil, fmt.Errorf("不能下载文件夹")
	}

	// 构建实际的文件路径
	fullPath := filepath.Join(s.filePath, file.StoragePath)

	// 检查文件是否物理存在
	if _, err := os.Stat(fullPath); err != nil {
		logrus.Errorf("物理文件不存在: %v", err)
		return nil, fmt.Errorf("文件已损坏或不存在")
	}

	// 设置文件的完整路径
	file.StoragePath = fullPath

	return file, nil
}

func (s *fileService) RenameFile(ctx context.Context, userID uint64, fileID string, newName string) error {
	// 解析fileID
	id, err := parseFileID(fileID)
	if err != nil {
		logrus.Errorf("解析文件ID失败: %v", err)
		return fmt.Errorf("无效的文件ID")
	}

	// 获取文件信息
	file, err := s.repo.FindByID(ctx, id)
	if err != nil {
		logrus.Errorf("查找文件失败: %v", err)
		return fmt.Errorf("文件不存在")
	}

	// 检查权限
	if file.UserID != userID {
		return fmt.Errorf("无权操作此文件")
	}

	// 检查同名文件是否存在
	existing, err := s.repo.FindByUserIDAndName(ctx, userID, file.ParentID, newName)
	if err == nil && existing != nil && existing.ID != file.ID {
		return fmt.Errorf("同名文件或文件夹已存在")
	}

	// 获取当前存储路径
	oldPath := filepath.Join(s.filePath, file.StoragePath)

	// 计算新的存储路径
	dir := filepath.Dir(file.StoragePath)
	newRelativePath := filepath.Join(dir, newName)
	newPath := filepath.Join(s.filePath, newRelativePath)

	// 执行文件系统重命名
	if err := os.Rename(oldPath, newPath); err != nil {
		logrus.Errorf("重命名文件失败: %v", err)
		return fmt.Errorf("重命名文件失败")
	}

	// 更新数据库记录
	file.Name = newName
	file.StoragePath = newRelativePath
	if !file.IsFolder {
		file.MimeType = getMimeType(newName)
	}

	if err := s.repo.Update(ctx, file); err != nil {
		logrus.Errorf("更新文件记录失败: %v", err)
		// 尝试恢复文件名
		_ = os.Rename(newPath, oldPath)
		return fmt.Errorf("更新文件信息失败")
	}

	return nil
}

func (s *fileService) DeleteFile(ctx context.Context, userID uint64, fileID string) error {
	// 解析fileID
	id, err := parseFileID(fileID)
	if err != nil {
		logrus.Errorf("解析文件ID失败: %v", err)
		return fmt.Errorf("无效的文件ID")
	}

	// 获取文件信息
	file, err := s.repo.FindByID(ctx, id)
	if err != nil {
		logrus.Errorf("查找文件失败: %v", err)
		return fmt.Errorf("文件不存在")
	}

	// 检查权限
	if file.UserID != userID {
		return fmt.Errorf("无权删除此文件")
	}

	// 实际路径
	fullPath := filepath.Join(s.filePath, file.StoragePath)

	// 删除物理文件或目录
	var fileErr error
	if file.IsFolder {
		fileErr = os.RemoveAll(fullPath)
	} else {
		fileErr = os.Remove(fullPath)
	}

	if fileErr != nil {
		logrus.Warnf("删除物理文件失败: %v，继续删除数据库记录", fileErr)
	}

	// 删除数据库记录
	if err := s.repo.Delete(ctx, id); err != nil {
		logrus.Errorf("删除文件记录失败: %v", err)
		return fmt.Errorf("删除文件记录失败")
	}

	return nil
}

// 辅助函数：根据文件名获取MIME类型
func getMimeType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))

	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".doc", ".docx":
		return "application/msword"
	case ".xls", ".xlsx":
		return "application/vnd.ms-excel"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

// 辅助函数：解析文件ID
func parseFileID(fileID string) (int, error) {
	var id int
	n, err := fmt.Sscanf(fileID, "%d", &id)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("无效的文件ID格式")
	}
	return id, nil
}
