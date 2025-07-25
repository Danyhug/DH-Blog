package service

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"dh-blog/internal/model"
	"dh-blog/internal/repository"

	"github.com/sirupsen/logrus"
)

type fileService struct {
	repo        repository.IFileRepository         // 文件存储库
	settingRepo repository.SystemSettingRepository // 系统设置仓库
	filePath    string                             // 实际存储文件的基础路径
	defaultPath string                             // 默认存储路径，当数据库未配置时使用
}

var (
	// 注入了default.go和task.go
	// filePathSettingKey = "file_storage_path" // 文件存储路径在数据库中的键名
	filePathSettingKey = model.SettingKeyFileStoragePath // 文件存储路径在数据库中的键名
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

	// UpdateStoragePath 7. 更新文件存储路径
	// 更新系统的文件存储路径，并迁移现有文件
	UpdateStoragePath(newPath string) error

	// GetSystemDirectoryTree 8. 获取系统目录树
	// 获取系统目录结构，用于前端选择存储路径
	GetSystemDirectoryTree(ctx context.Context, rootPath string, maxDepth int) (*DirectoryNode, error)
}

// NewFileService 创建新的文件服务
// 参数:
//   - repo: 文件存储库接口
//   - settingRepo: 系统设置仓库接口
//
// 返回:
//   - IFileService: 文件服务接口
func NewFileService(repo repository.IFileRepository, settingRepo repository.SystemSettingRepository) IFileService {
	// 默认路径应该是 可执行文件/data/webdav
	executable, err := os.Executable()
	if err != nil {
		logrus.Errorf("无法获取可执行文件的路径: %v", err)
		return nil
	}
	defaultPath := filepath.Join(filepath.Dir(executable), "data", "webdav")

	service := &fileService{
		repo:        repo,
		settingRepo: settingRepo,
		defaultPath: defaultPath,
	}

	// 尝试从数据库加载存储路径
	service.loadStoragePathFromDB()

	return service
}

// loadStoragePathFromDB 从数据库加载文件存储路径
func (s *fileService) loadStoragePathFromDB() {
	// 尝试从系统设置获取存储路径
	pathFromDB, err := s.settingRepo.GetSetting(filePathSettingKey)
	if err != nil {
		logrus.Warnf("从数据库获取文件存储路径失败: %v，将使用默认路径: %s", err, s.defaultPath)

		// 如果设置不存在，尝试创建该设置项
		err = s.settingRepo.UpdateSetting(filePathSettingKey, s.defaultPath)
		if err != nil {
			logrus.Errorf("创建文件存储路径设置失败: %v", err)
		}
		return
	}

	// 如果获取到了路径，但是为空，仍使用默认路径
	if pathFromDB == "" {
		logrus.Warn("数据库中文件存储路径为空，将使用默认路径")
		return
	}

	// 使用从数据库获取的路径
	s.filePath = pathFromDB
	logrus.Infof("已从数据库加载文件存储路径: %s", s.filePath)

	// 确保存储路径存在
	if err := os.MkdirAll(s.filePath, os.ModePerm); err != nil {
		logrus.Warnf("创建文件存储路径失败: %v，可能影响文件上传", err)
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
	// 先刷新存储路径设置
	s.loadStoragePathFromDB()

	// 使用存储库查询数据库
	files, err := s.repo.ListByParentID(ctx, userID, parentID)
	if err != nil {
		logrus.Errorf("列出文件失败: %v", err)
		return nil, fmt.Errorf("列出文件失败")
	}

	return files, nil
}

func (s *fileService) CreateFolder(ctx context.Context, userID uint64, parentID string, folderName string) (*model.File, error) {
	// 先刷新存储路径设置
	s.loadStoragePathFromDB()

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
	// 先刷新存储路径设置
	s.loadStoragePathFromDB()

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
	// 先刷新存储路径设置
	s.loadStoragePathFromDB()

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
	// 先刷新存储路径设置
	s.loadStoragePathFromDB()

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
	// 先刷新存储路径设置
	s.loadStoragePathFromDB()

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

// UpdateStoragePath 更新文件存储路径
func (s *fileService) UpdateStoragePath(newPath string) error {
	// 1. 更新系统设置中的存储路径
	err := s.settingRepo.UpdateSetting(filePathSettingKey, newPath)
	if err != nil {
		logrus.Errorf("更新文件存储路径设置失败: %v", err)
		return fmt.Errorf("更新文件存储路径失败")
	}
	logrus.Infof("文件存储路径已更新为: %s", newPath)

	// 2. 清空文件表，因为不同路径存储的数据不同
	ctx := context.Background()
	if err := s.repo.TruncateFiles(ctx); err != nil {
		logrus.Errorf("清空文件表失败: %v", err)
		return fmt.Errorf("清空文件表失败")
	}
	logrus.Info("文件表已清空")

	// 3. 确保新的存储路径存在
	if err := os.MkdirAll(newPath, os.ModePerm); err != nil {
		logrus.Errorf("创建新文件存储路径失败: %v", err)
		return fmt.Errorf("创建新文件存储路径失败")
	}
	logrus.Infof("新文件存储路径已创建: %s", newPath)

	// 4. 更新服务实例中的filePath
	s.filePath = newPath
	logrus.Infof("服务实例中的filePath已更新为: %s", s.filePath)

	// 5. 扫描目录并添加文件记录
	if err := s.scanAndAddFiles(ctx); err != nil {
		logrus.Warnf("扫描并添加文件记录失败: %v", err)
		return fmt.Errorf("扫描并添加文件记录失败: %v", err)
	}

	return nil
}

// scanAndAddFiles 扫描存储目录并将文件添加到数据库
func (s *fileService) scanAndAddFiles(ctx context.Context) error {
	logrus.Infof("开始扫描目录: %s", s.filePath)

	// 统计扫描结果
	var folderCount, fileCount int

	// 存储目录ID映射，用于建立父子关系
	// 键为目录的相对路径，值为目录在数据库中的ID
	dirIDMap := make(map[string]string)
	// 根目录的ID为空字符串
	dirIDMap[""] = ""

	// 第一次遍历：创建所有目录
	err := filepath.Walk(s.filePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("访问路径失败: %s, 错误: %v", path, err)
			return nil // 继续遍历
		}

		// 跳过根目录本身
		if path == s.filePath {
			return nil
		}

		// 跳过隐藏文件
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir // 跳过整个目录
			}
			return nil
		}

		// 只处理目录
		if !info.IsDir() {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(s.filePath, path)
		if err != nil {
			logrus.Warnf("获取相对路径失败: %s, 错误: %v", path, err)
			return nil
		}

		// 获取父目录路径
		parentPath := filepath.Dir(relPath)
		if parentPath == "." {
			parentPath = "" // 根目录
		}

		// 检查父目录ID是否存在
		parentID, exists := dirIDMap[parentPath]
		if !exists {
			logrus.Warnf("父目录ID不存在: %s", parentPath)
			return nil
		}

		// 创建目录记录
		folder := &model.File{
			UserID:   1, // 默认用户ID为1，表示系统用户
			ParentID: parentID,
			Name:     filepath.Base(path),
			IsFolder: true,
			Size:     0,
		}

		// 保存到数据库
		if err := s.repo.Create(ctx, folder); err != nil {
			logrus.Warnf("添加目录记录失败: %s, 错误: %v", path, err)
			return nil
		}

		// 保存目录ID到映射
		dirIDMap[relPath] = fmt.Sprintf("%d", folder.ID)
		folderCount++

		return nil
	})

	if err != nil {
		return fmt.Errorf("遍历目录失败: %v", err)
	}

	// 第二次遍历：添加所有文件
	err = filepath.Walk(s.filePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("访问路径失败: %s, 错误: %v", path, err)
			return nil // 继续遍历
		}

		// 跳过根目录本身
		if path == s.filePath {
			return nil
		}

		// 跳过隐藏文件
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir // 跳过整个目录
			}
			return nil
		}

		// 只处理文件
		if info.IsDir() {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(s.filePath, path)
		if err != nil {
			logrus.Warnf("获取相对路径失败: %s, 错误: %v", path, err)
			return nil
		}

		// 获取父目录路径
		parentPath := filepath.Dir(relPath)
		if parentPath == "." {
			parentPath = "" // 根目录
		}

		// 检查父目录ID是否存在
		parentID, exists := dirIDMap[parentPath]
		if !exists {
			logrus.Warnf("父目录ID不存在: %s", parentPath)
			return nil
		}

		// 创建文件记录
		file := &model.File{
			UserID:      1, // 默认用户ID为1，表示系统用户
			ParentID:    parentID,
			Name:        filepath.Base(path),
			IsFolder:    false,
			Size:        info.Size(),
			StoragePath: relPath,
			MimeType:    getMimeType(filepath.Base(path)),
		}

		// 保存到数据库
		if err := s.repo.Create(ctx, file); err != nil {
			logrus.Warnf("添加文件记录失败: %s, 错误: %v", path, err)
			return nil
		}

		fileCount++
		return nil
	})

	if err != nil {
		return fmt.Errorf("遍历目录失败: %v", err)
	}

	logrus.Infof("扫描完成，共添加 %d 个文件夹和 %d 个文件", folderCount, fileCount)
	return nil
}

// DirectoryNode 表示目录树中的一个节点
type DirectoryNode struct {
	Path     string           `json:"path"`     // 路径
	Name     string           `json:"name"`     // 目录名称
	IsDir    bool             `json:"isDir"`    // 是否是目录
	Children []*DirectoryNode `json:"children"` // 子目录
}

// GetSystemDirectoryTree 获取系统目录树
func (s *fileService) GetSystemDirectoryTree(ctx context.Context, rootPath string, maxDepth int) (*DirectoryNode, error) {
	// 如果没有指定根路径，使用系统根目录
	if rootPath == "" {
		if runtime.GOOS == "windows" {
			// Windows系统使用驱动器列表作为根目录
			return s.getWindowsDrives(ctx)
		} else {
			// Unix系统使用根目录
			rootPath = "/"
		}
	}

	// 检查路径是否存在
	info, err := os.Stat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("访问路径失败: %v", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("指定路径不是目录")
	}

	// 创建根节点
	root := &DirectoryNode{
		Path:  rootPath,
		Name:  filepath.Base(rootPath),
		IsDir: true,
	}

	// 递归构建目录树
	err = s.buildDirectoryTree(ctx, root, 0, maxDepth)
	if err != nil {
		return nil, err
	}

	return root, nil
}

// getWindowsDrives 获取Windows系统的驱动器列表
func (s *fileService) getWindowsDrives(ctx context.Context) (*DirectoryNode, error) {
	root := &DirectoryNode{
		Path:  "",
		Name:  "计算机",
		IsDir: true,
	}

	// 获取可用的驱动器
	for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		drivePath := string(drive) + ":\\"
		_, err := os.Stat(drivePath)
		if err == nil {
			driveNode := &DirectoryNode{
				Path:  drivePath,
				Name:  drivePath,
				IsDir: true,
			}
			root.Children = append(root.Children, driveNode)
		}
	}

	return root, nil
}

// buildDirectoryTree 递归构建目录树
func (s *fileService) buildDirectoryTree(ctx context.Context, node *DirectoryNode, currentDepth, maxDepth int) error {
	// 如果达到最大深度，不再继续
	if maxDepth > 0 && currentDepth >= maxDepth {
		return nil
	}

	// 读取目录内容
	entries, err := os.ReadDir(node.Path)
	if err != nil {
		return fmt.Errorf("读取目录失败: %v", err)
	}

	// 只保留目录
	var dirs []fs.DirEntry
	for _, entry := range entries {
		// 跳过隐藏文件和目录
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// 只处理目录
		if entry.IsDir() {
			dirs = append(dirs, entry)
		}
	}

	// 按名称排序
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name() < dirs[j].Name()
	})

	// 添加子目录
	for _, dir := range dirs {
		childPath := filepath.Join(node.Path, dir.Name())
		childNode := &DirectoryNode{
			Path:  childPath,
			Name:  dir.Name(),
			IsDir: true,
		}

		// 添加到当前节点的子节点
		node.Children = append(node.Children, childNode)

		// 递归处理子目录
		err := s.buildDirectoryTree(ctx, childNode, currentDepth+1, maxDepth)
		if err != nil {
			logrus.Warnf("处理子目录失败: %s, 错误: %v", childPath, err)
			// 继续处理其他目录
		}
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
