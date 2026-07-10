package files

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// 生成存储路径
func (s *fileService) getStoragePath(ctx context.Context, parentID string, name string) (string, string, error) {
	normalizedParentID := normalizeParentID(parentID)

	parentPath, err := s.resolveParentStoragePath(ctx, normalizedParentID)
	if err != nil {
		return "", "", err
	}

	relative := parentPath
	if name != "" {
		relative = filepath.Join(parentPath, name)
	}

	relative = filepath.Clean(relative)
	if relative == "." {
		relative = ""
	}

	if strings.HasPrefix(relative, "..") || filepath.IsAbs(relative) {
		return "", "", fmt.Errorf("检测到非法存储路径")
	}

	fullPath := filepath.Join(s.filePath, relative)
	return relative, fullPath, nil
}

func (s *fileService) resolveParentStoragePath(ctx context.Context, parentID string) (string, error) {
	if parentID == "" {
		return "", nil
	}

	id, err := parseFileID(parentID)
	if err != nil {
		return "", fmt.Errorf("无效的父目录ID")
	}

	parent, err := s.repo.FindByID(ctx, id)
	if err != nil {
		logrus.Errorf("获取父目录失败: %v", err)
		return "", fmt.Errorf("父目录不存在")
	}

	if !parent.IsFolder {
		return "", fmt.Errorf("父目录不是文件夹")
	}

	return parent.StoragePath, nil
}

func normalizeParentID(parentID string) string {
	trimmed := strings.TrimSpace(parentID)
	if trimmed == "." {
		return ""
	}
	return trimmed
}

func (s *fileService) ApplyStorageConfig(ctx context.Context, newPath string, chunkSizeKB int) error {
	if newPath == "" {
		return fmt.Errorf("文件存储路径不能为空")
	}
	if chunkSizeKB <= 0 {
		return fmt.Errorf("WebDAV 分片大小必须大于 0")
	}
	oldPath := s.GetStoragePath()
	oldChunkSize := s.ChunkSizeKB()
	if filepath.Clean(newPath) == filepath.Clean(oldPath) {
		s.pathMu.Lock()
		s.chunkSizeKB = chunkSizeKB
		s.pathMu.Unlock()
		return nil
	}
	info, err := os.Stat(newPath)
	if err != nil {
		return fmt.Errorf("存储路径不可用: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("存储路径不是目录: %s", newPath)
	}
	snapshot, err := s.repo.Snapshot(ctx)
	if err != nil {
		return fmt.Errorf("保存旧文件索引失败: %w", err)
	}
	// 清空文件表，因为不同路径存储的数据不同
	if err := s.repo.TruncateFiles(ctx); err != nil {
		logrus.Errorf("清空文件表失败: %v", err)
		return fmt.Errorf("清空文件表失败")
	}
	logrus.Info("文件表已清空")

	logrus.Infof("新文件存储路径已创建: %s", newPath)

	// 更新运行时配置
	s.pathMu.Lock()
	s.filePath = newPath
	s.chunkSizeKB = chunkSizeKB
	s.pathMu.Unlock()
	logrus.Infof("服务实例中的filePath已更新为: %s", s.filePath)

	// 5. 扫描目录并添加文件记录
	if err := s.scanAndAddFiles(ctx); err != nil {
		logrus.Warnf("扫描并添加文件记录失败: %v", err)
		s.pathMu.Lock()
		s.filePath = oldPath
		s.chunkSizeKB = oldChunkSize
		s.pathMu.Unlock()
		if rollbackErr := s.repo.RestoreSnapshot(context.Background(), snapshot); rollbackErr != nil {
			return fmt.Errorf("扫描新存储路径失败: %v；恢复旧索引失败: %w", err, rollbackErr)
		}
		return fmt.Errorf("扫描并添加文件记录失败，已恢复旧存储配置: %w", err)
	}

	return nil
}

// InitializeStorageConfig applies persisted runtime settings without rebuilding
// the file index. Startup must preserve existing file IDs because shares refer
// to them by ID.
func (s *fileService) InitializeStorageConfig(_ context.Context, path string, chunkSizeKB int) error {
	if path == "" {
		return fmt.Errorf("文件存储路径不能为空")
	}
	if chunkSizeKB <= 0 {
		return fmt.Errorf("WebDAV 分片大小必须大于 0")
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("创建文件存储路径失败: %w", err)
	}
	s.pathMu.Lock()
	s.filePath = path
	s.chunkSizeKB = chunkSizeKB
	s.pathMu.Unlock()
	return nil
}

// GetStoragePath 获取当前存储路径
func (s *fileService) GetStoragePath() string {
	s.pathMu.RLock()
	defer s.pathMu.RUnlock()
	return s.filePath
}

func (s *fileService) ChunkSizeKB() int {
	s.pathMu.RLock()
	defer s.pathMu.RUnlock()
	return s.chunkSizeKB
}

func (s *fileService) ProtectedDirectoryNames() []string {
	return append([]string(nil), protectedDirectories[:]...)
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
