package files

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// 全盘搜索的参数。上限存在的意义是把「关键词太宽」这件事变成一个可提示的状态，
// 而不是让前端渲染上万条结果。
const (
	searchResultLimit = 200
	// searchMaxDepth 是向上回溯父目录的深度上限，索引异常成环时用来兜底。
	searchMaxDepth = 64
)

// PathSegment 是文件所在目录链上的一段，前端据此直接跳转到该目录。
type PathSegment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SearchHit 是一条全盘搜索命中：文件本身，加上定位它所需的所在目录信息。
type SearchHit struct {
	File
	// ParentPath 是所在目录的展示路径（以 / 分隔），位于根目录时为空。
	ParentPath string `json:"parent_path"`
	// ParentSegments 是从根到所在目录的完整链路，供前端还原面包屑。
	ParentSegments []PathSegment `json:"parent_segments"`
}

// SearchResult 是一次全盘搜索的完整结果。
type SearchResult struct {
	Files []*SearchHit `json:"files"`
	// Truncated 表示命中数超过 Limit，返回的只是前 Limit 条。
	Truncated bool `json:"truncated"`
	Limit     int  `json:"limit"`
}

// SearchFiles 跨目录检索用户名下名称包含 keyword 的文件与文件夹。
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - keyword: 关键词，首尾空白会被忽略
//
// 返回:
//   - *SearchResult: 命中列表及是否被截断
//   - error: 错误信息
func (s *fileService) SearchFiles(ctx context.Context, userID uint64, keyword string) (*SearchResult, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return &SearchResult{Files: []*SearchHit{}, Limit: searchResultLimit}, nil
	}

	// 多取一条用来判断是否还有更多结果，返回前再裁掉。
	matches, err := s.repo.SearchByName(ctx, userID, keyword, searchResultLimit+1)
	if err != nil {
		logrus.Errorf("搜索文件失败: %v", err)
		return nil, fmt.Errorf("搜索文件失败")
	}

	truncated := len(matches) > searchResultLimit
	if truncated {
		matches = matches[:searchResultLimit]
	}

	folders, err := s.repo.ListFolders(ctx, userID)
	if err != nil {
		logrus.Errorf("加载目录索引失败: %v", err)
		return nil, fmt.Errorf("搜索文件失败")
	}
	foldersByID := make(map[string]*File, len(folders))
	for _, folder := range folders {
		foldersByID[strconv.Itoa(folder.ID)] = folder
	}

	hits := make([]*SearchHit, 0, len(matches))
	for _, match := range matches {
		segments := resolveParentSegments(foldersByID, match.ParentID)
		names := make([]string, 0, len(segments))
		for _, segment := range segments {
			names = append(names, segment.Name)
		}
		hits = append(hits, &SearchHit{
			File:           *match,
			ParentPath:     strings.Join(names, "/"),
			ParentSegments: segments,
		})
	}

	return &SearchResult{Files: hits, Truncated: truncated, Limit: searchResultLimit}, nil
}

// resolveParentSegments 从 parentID 向上回溯，返回从根到该目录的路径链。
// 目录已被删除、或索引意外成环时返回已经拿到的部分，搜索不该因为脏索引而整体失败。
func resolveParentSegments(foldersByID map[string]*File, parentID string) []PathSegment {
	var reversed []PathSegment

	visited := make(map[string]bool)
	for current := normalizeParentID(parentID); current != ""; {
		if visited[current] {
			logrus.Warnf("目录索引存在环，已在 %s 处停止回溯", current)
			break
		}
		visited[current] = true

		folder, ok := foldersByID[current]
		if !ok {
			break
		}
		reversed = append(reversed, PathSegment{ID: current, Name: folder.Name})
		if len(reversed) >= searchMaxDepth {
			break
		}
		current = normalizeParentID(folder.ParentID)
	}

	segments := make([]PathSegment, 0, len(reversed))
	for i := len(reversed) - 1; i >= 0; i-- {
		segments = append(segments, reversed[i])
	}
	return segments
}
