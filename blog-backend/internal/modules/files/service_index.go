package files

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

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
		folder := &File{
			UserID:      1, // 默认用户ID为1，表示系统用户
			ParentID:    parentID,
			Name:        filepath.Base(path),
			IsFolder:    true,
			Size:        0,
			StoragePath: relPath,
		}

		// 保存到数据库
		if err := s.repo.Create(ctx, folder); err != nil {
			logrus.Warnf("添加目录记录失败: %s, 错误: %v", path, err)
			return fmt.Errorf("添加目录记录 %s: %w", path, err)
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
		file := &File{
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
			return fmt.Errorf("添加文件记录 %s: %w", path, err)
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

// SyncFilesFromDisk 从磁盘同步文件到数据库（立即执行）
func (s *fileService) SyncFilesFromDisk() error {
	return s.doSyncFilesFromDisk()
}

// SyncFilesFromDiskDebounced 从磁盘同步文件到数据库（防抖，非阻塞）
// 每次调用会重置 5 秒定时器，适合 WebDAV 批量操作时频繁触发
func (s *fileService) SyncFilesFromDiskDebounced() {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	// 如果已有等待中的同步任务，重置定时器
	if s.syncTimer != nil {
		s.syncTimer.Stop()
	}

	s.syncTimer = time.AfterFunc(5*time.Second, func() {
		s.syncMu.Lock()
		s.syncTimer = nil
		s.syncMu.Unlock()

		if err := s.doSyncFilesFromDisk(); err != nil {
			logrus.Warnf("防抖同步文件失败: %v", err)
		}
	})
}

// doSyncFilesFromDisk 实际执行文件同步
func (s *fileService) doSyncFilesFromDisk() error {
	s.syncExecMu.Lock()
	defer s.syncExecMu.Unlock()

	logrus.Info("开始从磁盘同步文件到数据库")

	ctx := context.Background()

	// 1. 清空文件表
	if err := s.repo.TruncateFiles(ctx); err != nil {
		logrus.Errorf("同步文件时清空文件表失败: %v", err)
		return fmt.Errorf("清空文件表失败: %v", err)
	}

	// 2. 重新扫描磁盘目录
	if err := s.scanAndAddFiles(ctx); err != nil {
		logrus.Errorf("同步文件时扫描目录失败: %v", err)
		return fmt.Errorf("扫描目录失败: %v", err)
	}

	// 3. 确保固定目录存在
	if err := s.EnsureProtectedDirectories(ctx); err != nil {
		logrus.Warnf("同步文件时创建固定目录失败: %v", err)
	}

	logrus.Info("磁盘文件同步完成")
	return nil
}
