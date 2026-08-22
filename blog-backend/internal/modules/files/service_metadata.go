package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func (s *fileService) ListFiles(ctx context.Context, userID uint64, parentID string) ([]*File, error) {
	normalizedParentID := normalizeParentID(parentID)

	if _, err := s.resolveParentStoragePath(ctx, normalizedParentID); err != nil {
		return nil, err
	}

	// 使用存储库查询数据库
	files, err := s.repo.ListByParentID(ctx, userID, normalizedParentID)
	if err != nil {
		logrus.Errorf("列出文件失败: %v", err)
		return nil, fmt.Errorf("列出文件失败")
	}

	filtered := make([]*File, 0, len(files))
	for _, file := range files {
		if normalizeParentID(file.ParentID) == normalizedParentID {
			filtered = append(filtered, file)
		}
	}

	return filtered, nil
}

func (s *fileService) CreateFolder(ctx context.Context, userID uint64, parentID string, folderName string) (*File, error) {
	normalizedParentID := normalizeParentID(parentID)

	// 检查是否已存在同名文件夹
	existing, err := s.repo.FindByUserIDAndName(ctx, userID, normalizedParentID, folderName)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("同名文件夹已存在")
	}

	// 计算存储路径
	relativePath, fullPath, err := s.getStoragePath(ctx, normalizedParentID, folderName)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
		logrus.Errorf("创建实际文件夹失败: %v", err)
		return nil, fmt.Errorf("创建文件夹失败")
	}

	// 创建文件夹记录
	folder := &File{
		UserID:      userID,
		ParentID:    normalizedParentID,
		Name:        folderName,
		IsFolder:    true,
		Size:        0,
		StoragePath: relativePath,
	}

	// 保存到数据库
	if err := s.repo.Create(ctx, folder); err != nil {
		logrus.Errorf("创建文件夹失败: %v", err)
		// 清理已创建的目录，避免孤儿目录
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("创建文件夹失败")
	}

	return folder, nil
}

func (s *fileService) UploadFile(ctx context.Context, userID uint64, parentID string, fileName string, fileSize int64, fileContent io.Reader) (*File, error) {
	normalizedParentID := normalizeParentID(parentID)

	// 检查是否已存在同名文件
	existing, err := s.repo.FindByUserIDAndName(ctx, userID, normalizedParentID, fileName)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("同名文件已存在")
	}

	// 创建物理文件存储路径
	relativePath, fullPath, err := s.getStoragePath(ctx, normalizedParentID, fileName)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
		logrus.Errorf("创建文件目录失败: %v", err)
		return nil, fmt.Errorf("创建文件失败")
	}

	// 创建目标文件
	outFile, err := os.Create(fullPath)
	if err != nil {
		logrus.Errorf("创建文件失败: %v", err)
		return nil, fmt.Errorf("创建文件失败")
	}
	defer func() { _ = outFile.Close() }()

	// 写入文件内容
	written, err := io.Copy(outFile, fileContent)
	if err != nil {
		logrus.Errorf("写入文件内容失败: %v", err)
		// 删除可能已创建的文件
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("写入文件内容失败")
	}

	// 创建文件数据库记录
	file := &File{
		UserID:      userID,
		ParentID:    normalizedParentID,
		Name:        fileName,
		IsFolder:    false,
		Size:        written,
		StoragePath: relativePath,
		// 确定MIME类型
		MimeType: getMimeType(fileName),
	}

	// 保存到数据库
	err = s.repo.Create(ctx, file)
	if err != nil {
		logrus.Errorf("保存文件记录失败: %v", err)
		// 删除已创建的文件
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("保存文件记录失败")
	}

	return file, nil
}

func (s *fileService) GetDownloadInfo(ctx context.Context, userID uint64, fileID string) (*File, error) {
	file, err := s.findOwnedFile(ctx, userID, fileID)
	if err != nil {
		return nil, err
	}

	// 检查是否是文件夹
	if file.IsFolder {
		return nil, fmt.Errorf("不能下载文件夹")
	}

	if err := s.resolveStoragePath(file); err != nil {
		return nil, err
	}

	return file, nil
}

// ResolveBatchDownloadInfo 解析批量下载的条目，跳过其中的文件夹。
// 与 GetDownloadInfo 的差别只在于文件夹的处理：打包下载允许选中集合里混有
// 文件夹（当前实现不递归打包目录，直接忽略），而不是让整个请求失败。
func (s *fileService) ResolveBatchDownloadInfo(ctx context.Context, userID uint64, fileIDs []string) ([]*File, error) {
	items := make([]*File, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		file, err := s.findOwnedFile(ctx, userID, fileID)
		if err != nil {
			return nil, err
		}
		if file.IsFolder {
			continue
		}
		if err := s.resolveStoragePath(file); err != nil {
			return nil, err
		}
		items = append(items, file)
	}
	return items, nil
}

// findOwnedFile 按 ID 或路径查出文件记录，并确认属于该用户。
func (s *fileService) findOwnedFile(ctx context.Context, userID uint64, fileID string) (*File, error) {
	// 解析fileID，在MVP版本中fileID可以是文件记录ID或文件路径
	var file *File
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

	return file, nil
}

// resolveStoragePath 把记录里的相对路径换成磁盘绝对路径，并确认文件确实存在。
func (s *fileService) resolveStoragePath(file *File) error {
	// 构建实际的文件路径
	fullPath := filepath.Join(s.filePath, file.StoragePath)

	// 检查文件是否物理存在
	if _, err := os.Stat(fullPath); err != nil {
		logrus.Errorf("物理文件不存在: %v", err)
		return fmt.Errorf("文件已损坏或不存在")
	}

	// 设置文件的完整路径
	file.StoragePath = fullPath

	return nil
}

func (s *fileService) GetDownloadInfoForShare(ctx context.Context, fileID string) (*File, error) {
	id, err := parseFileID(fileID)
	if err != nil {
		return nil, fmt.Errorf("无效的文件ID")
	}

	file, err := s.repo.FindByID(ctx, id)
	if err != nil {
		logrus.Errorf("查找文件失败: %v", err)
		return nil, fmt.Errorf("文件不存在")
	}

	if file.IsFolder {
		return nil, fmt.Errorf("不能下载文件夹")
	}

	fullPath := filepath.Join(s.GetStoragePath(), file.StoragePath)
	if _, err := os.Stat(fullPath); err != nil {
		logrus.Errorf("物理文件不存在: %v", err)
		return nil, fmt.Errorf("文件已损坏或不存在")
	}

	file.StoragePath = fullPath
	return file, nil
}

// findFolderByID 供 chunk 上传完成流程查询父目录，仅接受文件夹。
func (s *fileService) findFolderByID(ctx context.Context, id int) (*File, error) {
	folder, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("父目录不存在")
	}
	if !folder.IsFolder {
		return nil, fmt.Errorf("父目录不是文件夹")
	}
	return folder, nil
}

// hasNameConflict 检查同一用户同一目录下是否已有同名条目。
// FindByUserIDAndName 用 GORM First 查询，"查无此名"会返回 gorm.ErrRecordNotFound，
// 这是正常路径（无冲突），必须转成 (false, nil)，否则会变成"同名检查失败"。
func (s *fileService) hasNameConflict(ctx context.Context, userID uint64, parentID, name string) (bool, error) {
	existing, err := s.repo.FindByUserIDAndName(ctx, userID, parentID, name)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return existing != nil, nil
}

// createFileRecord 保存 chunk 上传完成后的文件索引记录。
func (s *fileService) createFileRecord(ctx context.Context, file *File) error {
	return s.repo.Create(ctx, file)
}

func (s *fileService) RenameFile(ctx context.Context, userID uint64, fileID string, newName string) error {
	// 解析fileID
	id, err := parseFileID(fileID)
	if err != nil {
		logrus.Errorf("解析文件ID失败: %v", err)
		return fmt.Errorf("无效的文件ID")
	}

	if err := validateFileName(newName); err != nil {
		return fmt.Errorf("新名称无效: %v", err)
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
	existing, err := s.repo.FindByUserIDAndName(ctx, userID, normalizeParentID(file.ParentID), newName)
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

	// 检查是否为固定目录（根目录下的音乐、图片、视频）
	if file.IsFolder && file.ParentID == "" {
		for _, name := range protectedDirectories {
			if file.Name == name {
				return fmt.Errorf("系统目录 '%s' 不能删除", name)
			}
		}
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

// 辅助函数：根据文件名获取MIME类型。
// mime.TypeByExtension 依赖系统 mime 表（Windows 打包环境可能缺失常见音视频类型），
// 因此用回退表兜底，保证 mp3/mp4 等预览类型在所有平台可用。
func getMimeType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext != "" {
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			return mimeType
		}
	}

	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".html", ".htm":
		return "text/html"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a":
		return "audio/mp4"
	case ".flac":
		return "audio/flac"
	case ".opus":
		return "audio/opus"
	case ".wav":
		return "audio/wav"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".ogv":
		return "video/ogg"
	case ".mov":
		return "video/quicktime"
	case ".mkv":
		return "video/x-matroska"
	default:
		// 文本与代码类文件统一按 text/plain 返回，前端才能直接取到内容预览。
		// 放在最后兜底，前面的具体类型优先。
		if isPlainTextExt(ext) {
			return "text/plain"
		}
		return "application/octet-stream"
	}
}

// plainTextExts 是内容为纯文本、可安全按 text/plain 返回的扩展名。
// 与前端 utils/fileType.ts 的文本类型表对应。
var plainTextExts = map[string]bool{
	".log": true, ".md": true, ".markdown": true, ".ini": true, ".conf": true,
	".config": true, ".cfg": true, ".env": true, ".properties": true, ".toml": true,
	".yaml": true, ".yml": true, ".json": true, ".json5": true, ".jsonc": true,
	".srt": true, ".vtt": true, ".lrc": true, ".tsv": true, ".diff": true, ".patch": true,
	".js": true, ".mjs": true, ".cjs": true, ".jsx": true, ".ts": true, ".tsx": true,
	".vue": true, ".svelte": true, ".css": true, ".scss": true, ".sass": true, ".less": true,
	".py": true, ".java": true, ".kt": true, ".kts": true, ".scala": true, ".groovy": true,
	".c": true, ".h": true, ".cpp": true, ".cc": true, ".cxx": true, ".hpp": true,
	".cs": true, ".go": true, ".rs": true, ".swift": true, ".dart": true, ".php": true,
	".rb": true, ".lua": true, ".pl": true, ".pm": true, ".r": true, ".jl": true,
	".ex": true, ".exs": true, ".sh": true, ".bash": true, ".zsh": true, ".fish": true,
	".bat": true, ".cmd": true, ".ps1": true, ".sql": true, ".graphql": true, ".gql": true,
	".proto": true, ".tf": true, ".tfvars": true, ".gradle": true, ".cmake": true, ".mk": true,
}

func isPlainTextExt(ext string) bool {
	return plainTextExts[ext]
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

// EnsureProtectedDirectories 确保固定目录存在
// 在项目启动时调用，自动创建音乐、图片、视频等固定目录
func (s *fileService) EnsureProtectedDirectories(ctx context.Context) error {
	// 使用管理员用户ID (通常为1)
	const adminUserID uint64 = 1

	for _, dirName := range protectedDirectories {
		// 检查目录是否已存在于数据库中
		existingFile, _ := s.repo.FindByUserIDAndName(ctx, adminUserID, "", dirName)
		if existingFile != nil {
			logrus.Debugf("固定目录已存在: %s", dirName)
			// 确保物理目录也存在
			fullPath := filepath.Join(s.filePath, dirName)
			if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
				logrus.Warnf("创建物理目录失败: %s, 错误: %v", fullPath, err)
			}
			continue
		}

		// 创建物理目录
		fullPath := filepath.Join(s.filePath, dirName)
		if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
			logrus.Errorf("创建固定目录失败: %s, 错误: %v", fullPath, err)
			return fmt.Errorf("创建固定目录 %s 失败: %w", dirName, err)
		}

		// 创建数据库记录
		folder := &File{
			UserID:      adminUserID,
			ParentID:    "", // 根目录
			Name:        dirName,
			IsFolder:    true,
			StoragePath: dirName,
			MimeType:    "",
		}

		if err := s.repo.Create(ctx, folder); err != nil {
			logrus.Errorf("创建固定目录数据库记录失败: %s, 错误: %v", dirName, err)
			return fmt.Errorf("创建固定目录 %s 数据库记录失败: %w", dirName, err)
		}

		logrus.Infof("已创建固定目录: %s", dirName)
	}

	return nil
}

// GetProtectedDirectoryID 获取固定目录的数据库ID
func (s *fileService) GetProtectedDirectoryID(ctx context.Context, dirName string) (string, error) {
	// 使用管理员用户ID
	const adminUserID uint64 = 1

	// 检查是否为有效的固定目录名称
	isProtected := false
	for _, name := range protectedDirectories {
		if name == dirName {
			isProtected = true
			break
		}
	}
	if !isProtected {
		return "", fmt.Errorf("'%s' 不是有效的固定目录", dirName)
	}

	// 查找目录
	file, err := s.repo.FindByUserIDAndName(ctx, adminUserID, "", dirName)
	if err != nil {
		return "", fmt.Errorf("查找固定目录失败: %w", err)
	}
	if file == nil {
		// 兜底创建固定目录（防止历史数据不完整）
		fullPath := filepath.Join(s.filePath, dirName)
		if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
			return "", fmt.Errorf("创建固定目录 '%s' 失败: %w", dirName, err)
		}

		folder := &File{
			UserID:      adminUserID,
			ParentID:    "",
			Name:        dirName,
			IsFolder:    true,
			StoragePath: dirName,
			MimeType:    "",
		}

		if err := s.repo.Create(ctx, folder); err != nil {
			return "", fmt.Errorf("创建固定目录 '%s' 失败: %w", dirName, err)
		}

		return fmt.Sprintf("%d", folder.ID), nil
	}

	if file.StoragePath == "" {
		file.StoragePath = dirName
		if err := s.repo.Update(ctx, file); err != nil {
			return "", fmt.Errorf("更新固定目录路径失败: %w", err)
		}
	}

	fullPath := filepath.Join(s.filePath, file.StoragePath)
	if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
		return "", fmt.Errorf("创建固定目录 '%s' 失败: %w", dirName, err)
	}

	return fmt.Sprintf("%d", file.ID), nil
}
