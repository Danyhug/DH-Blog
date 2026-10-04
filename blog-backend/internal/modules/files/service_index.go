package files

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// tempDirName 是分片上传的临时目录名，扫描时跳过它，
// 否则未完成的分片会被当成正式文件索引进网盘列表。
const tempDirName = "temp"

// errParentMissing 表示条目的父目录尚未入库（父目录自身被跳过），
// 调用方应跳过该条目而不是把它登记到根目录。
var errParentMissing = errors.New("父目录未建立索引")

// diskEntry 是磁盘扫描得到的一个路径条目。
type diskEntry struct {
	relPath string
	isDir   bool
	size    int64
}

// scanDiskEntries 遍历存储目录，跳过隐藏条目与分片上传的 temp 目录。
func (s *fileService) scanDiskEntries() ([]diskEntry, error) {
	var entries []diskEntry
	err := filepath.Walk(s.filePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			logrus.Warnf("访问路径失败: %s, 错误: %v", path, err)
			return nil // 继续遍历
		}
		if path == s.filePath {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relPath, err := filepath.Rel(s.filePath, path)
		if err != nil {
			logrus.Warnf("获取相对路径失败: %s, 错误: %v", path, err)
			return nil
		}
		if relPath == tempDirName && info.IsDir() {
			return filepath.SkipDir
		}
		entries = append(entries, diskEntry{relPath: relPath, isDir: info.IsDir(), size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("遍历目录失败: %w", err)
	}
	return entries, nil
}

// scanAndAddFiles 全量扫描存储目录并重建索引，仅在切换存储路径时使用。
func (s *fileService) scanAndAddFiles(ctx context.Context) error {
	logrus.Infof("开始扫描目录: %s", s.filePath)

	entries, err := s.scanDiskEntries()
	if err != nil {
		return err
	}

	// 目录按路径深度排序，保证父目录先于子目录入库。
	var dirEntries []diskEntry
	var fileEntries []diskEntry
	for _, entry := range entries {
		if entry.isDir {
			dirEntries = append(dirEntries, entry)
		} else {
			fileEntries = append(fileEntries, entry)
		}
	}
	sortByPathDepth(dirEntries)

	dirIDMap := map[string]string{"": ""}
	for _, dir := range dirEntries {
		parentID, err := s.createDiskEntry(ctx, dir, dirIDMap)
		if err != nil {
			if errors.Is(err, errParentMissing) {
				logrus.Warnf("跳过磁盘条目: %v", err)
				continue
			}
			return err
		}
		dirIDMap[dir.relPath] = parentID
	}

	// 文件没有父子依赖，但按名称排序让索引顺序稳定。
	sortByPathDepth(fileEntries)
	for _, file := range fileEntries {
		if _, err := s.createDiskEntry(ctx, file, dirIDMap); err != nil {
			if errors.Is(err, errParentMissing) {
				logrus.Warnf("跳过磁盘条目: %v", err)
				continue
			}
			return err
		}
	}

	logrus.Infof("扫描完成，共添加 %d 个文件夹和 %d 个文件", len(dirEntries), len(fileEntries))
	return nil
}

func sortByPathDepth(entries []diskEntry) {
	sort.Slice(entries, func(i, j int) bool {
		depthI := strings.Count(entries[i].relPath, string(os.PathSeparator))
		depthJ := strings.Count(entries[j].relPath, string(os.PathSeparator))
		if depthI != depthJ {
			return depthI < depthJ
		}
		return entries[i].relPath < entries[j].relPath
	})
}

// createDiskEntry 把单个磁盘条目写进索引。dirIDMap 记录已入库目录的相对路径到 ID 的映射。
// 父目录不在 map 里说明其自身入库被跳过，返回错误让调用方跳过本条，
// 而不是把空 parentID 登记进 map、让子项被静默挂到根目录。
func (s *fileService) createDiskEntry(ctx context.Context, entry diskEntry, dirIDMap map[string]string) (string, error) {
	parentPath := filepath.Dir(entry.relPath)
	if parentPath == "." {
		parentPath = ""
	}

	parentID, exists := dirIDMap[parentPath]
	if !exists {
		return "", fmt.Errorf("%w: %s", errParentMissing, entry.relPath)
	}

	name := filepath.Base(entry.relPath)
	if entry.isDir {
		folder := &File{
			UserID:      1,
			ParentID:    parentID,
			Name:        name,
			IsFolder:    true,
			Size:        0,
			StoragePath: entry.relPath,
		}
		if err := s.repo.Create(ctx, folder); err != nil {
			return "", fmt.Errorf("添加目录记录 %s: %w", entry.relPath, err)
		}
		return fmt.Sprintf("%d", folder.ID), nil
	}

	file := &File{
		UserID:      1,
		ParentID:    parentID,
		Name:        name,
		IsFolder:    false,
		Size:        entry.size,
		StoragePath: entry.relPath,
		MimeType:    getMimeType(name),
	}
	if err := s.repo.Create(ctx, file); err != nil {
		return "", fmt.Errorf("添加文件记录 %s: %w", entry.relPath, err)
	}
	return "", nil
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

// SyncStats 是一次磁盘对账的结果，供手动同步按钮向用户报告。
type SyncStats struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Updated int `json:"updated"`
}

// doSyncFilesFromDisk 增量同步磁盘到数据库，保留已有记录的文件 ID，
// 否则每次 WebDAV 写操作都会重建索引、让所有按 ID 引用的分享链接失效。
func (s *fileService) doSyncFilesFromDisk() error {
	_, err := s.SyncFilesFromDisk()
	return err
}

// SyncFilesFromDisk 立即执行一次磁盘对账并返回增删改数量（网盘的「从磁盘同步」按钮）。
func (s *fileService) SyncFilesFromDisk() (stats SyncStats, err error) {
	s.syncExecMu.Lock()
	defer s.syncExecMu.Unlock()

	logrus.Info("开始从磁盘同步文件到数据库")
	// Announced only after the lock is taken, so the feed reflects work that
	// is actually running rather than work that is queued behind another sync.
	if s.events != nil {
		s.events.SyncStarted()
		defer func() { s.events.SyncFinished(err) }()
	}

	// 存储根目录不可访问（外接盘没挂上、路径被删）时 Walk 会得到空列表，
	// 对账会把整张索引当成「磁盘上已删除」清空，所以必须先拦下。
	if info, statErr := os.Stat(s.filePath); statErr != nil || !info.IsDir() {
		return stats, fmt.Errorf("存储目录不可访问: %s", s.filePath)
	}

	// 磁盘遍历放在事务外：Walk 是纯 IO，与索引读写无关，
	// 没必要依赖 SQLite deferred BEGIN 的"写语句才拿锁"行为。
	entries, err := s.scanDiskEntries()
	if err != nil {
		return stats, fmt.Errorf("扫描磁盘失败: %w", err)
	}

	ctx := context.Background()
	// 对账逐条增删索引，包在事务里：中途失败整体回滚，避免留下半同步状态。
	if err := s.repo.Transaction(ctx, func(repo fileRepository) error {
		stats = SyncStats{}
		return s.reconcileIndexWithDisk(ctx, repo, entries, &stats)
	}); err != nil {
		return SyncStats{}, err
	}

	// 确保固定目录存在（磁盘上没有时补建）
	if err := s.EnsureProtectedDirectories(ctx); err != nil {
		logrus.Warnf("同步文件时创建固定目录失败: %v", err)
	}

	logrus.Infof("磁盘文件同步完成: 新增 %d, 移除 %d, 更新 %d", stats.Added, stats.Removed, stats.Updated)
	return stats, nil
}

// reconcileIndexWithDisk 把磁盘扫描结果与索引逐条对账：
// 磁盘新增 → 建记录；磁盘删除 → 物理删记录；都在 → 保留 ID 只更新元数据。
// repo 由调用方在事务内创建，保证对账要么整体成功要么整体回滚。
func (s *fileService) reconcileIndexWithDisk(ctx context.Context, repo fileRepository, entries []diskEntry, stats *SyncStats) error {
	diskDirs := make(map[string]diskEntry)
	diskFiles := make(map[string]diskEntry)
	for _, entry := range entries {
		if entry.isDir {
			diskDirs[entry.relPath] = entry
		} else {
			diskFiles[entry.relPath] = entry
		}
	}

	existing, err := repo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("读取现有文件索引失败: %w", err)
	}

	// 软删残留与磁盘上已消失的记录一律物理删除；
	// 目录被整体删除时，其下所有子记录也会因为磁盘上不存在而一并清理。
	// 同名路径从目录变文件（或反之）时旧记录类型错位，同样物理删除后按磁盘现状重建。
	dirIDByPath := map[string]string{"": ""}
	existingByPath := make(map[string]*File)
	for _, file := range existing {
		_, dirOnDisk := diskDirs[file.StoragePath]
		_, fileOnDisk := diskFiles[file.StoragePath]
		typeMismatch := (dirOnDisk && !file.IsFolder) || (fileOnDisk && file.IsFolder)
		if file.DeletedAt.Valid || (!dirOnDisk && !fileOnDisk) || typeMismatch {
			if err := repo.HardDelete(ctx, file.ID); err != nil {
				return fmt.Errorf("清理失效记录 %d 失败: %w", file.ID, err)
			}
			// 软删残留本来就不在网盘列表里，不算作用户可见的「移除」
			if !file.DeletedAt.Valid {
				stats.Removed++
			}
			continue
		}
		existingByPath[file.StoragePath] = file
		if file.IsFolder && dirOnDisk {
			dirIDByPath[file.StoragePath] = fmt.Sprintf("%d", file.ID)
		}
	}

	// 目录按深度排序，保证父目录 ID 在子目录之前就绪。
	var dirPaths []string
	for relPath := range diskDirs {
		if _, ok := existingByPath[relPath]; ok {
			continue
		}
		dirPaths = append(dirPaths, relPath)
	}
	sort.Slice(dirPaths, func(i, j int) bool {
		depthI := strings.Count(dirPaths[i], string(os.PathSeparator))
		depthJ := strings.Count(dirPaths[j], string(os.PathSeparator))
		if depthI != depthJ {
			return depthI < depthJ
		}
		return dirPaths[i] < dirPaths[j]
	})

	for _, relPath := range dirPaths {
		parentPath := filepath.Dir(relPath)
		if parentPath == "." {
			parentPath = ""
		}
		parentID, ok := dirIDByPath[parentPath]
		if !ok {
			logrus.Warnf("同步时父目录未建立索引: %s", relPath)
			continue
		}
		folder := &File{
			UserID:      1,
			ParentID:    parentID,
			Name:        filepath.Base(relPath),
			IsFolder:    true,
			Size:        0,
			StoragePath: relPath,
		}
		if err := repo.Create(ctx, folder); err != nil {
			return fmt.Errorf("添加目录记录 %s: %w", relPath, err)
		}
		stats.Added++
		dirIDByPath[relPath] = fmt.Sprintf("%d", folder.ID)
	}

	for relPath, entry := range diskFiles {
		if file, ok := existingByPath[relPath]; ok {
			// 记录与磁盘都在：保留 ID，刷新大小与 MIME。
			// MIME 也要刷：历史分片上传写进过 application/octet-stream，
			// 光修合并逻辑救不了存量数据。
			mimeType := getMimeType(relPath)
			if file.Size != entry.size || file.MimeType != mimeType {
				file.Size = entry.size
				file.MimeType = mimeType
				if err := repo.Update(ctx, file); err != nil {
					return fmt.Errorf("更新文件记录 %s: %w", relPath, err)
				}
				stats.Updated++
			}
			continue
		}

		parentPath := filepath.Dir(relPath)
		if parentPath == "." {
			parentPath = ""
		}
		parentID, ok := dirIDByPath[parentPath]
		if !ok {
			logrus.Warnf("同步时父目录未建立索引: %s", relPath)
			continue
		}
		file := &File{
			UserID:      1,
			ParentID:    parentID,
			Name:        filepath.Base(relPath),
			IsFolder:    false,
			Size:        entry.size,
			StoragePath: relPath,
			MimeType:    getMimeType(relPath),
		}
		if err := repo.Create(ctx, file); err != nil {
			return fmt.Errorf("添加文件记录 %s: %w", relPath, err)
		}
		stats.Added++
	}

	return nil
}
