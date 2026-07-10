package comment

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"
)

// Repository 封装评论的数据访问逻辑。
type Repository struct {
	db *gorm.DB
}

func newRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// AddComment 添加评论。
func (r *Repository) AddComment(comment *Comment) error {
	if err := r.db.Create(comment).Error; err != nil {
		return fmt.Errorf("添加评论失败: %w", err)
	}
	return nil
}

// GetCommentsByArticleID 根据文章 ID 获取公开评论列表。
func (r *Repository) GetCommentsByArticleID(articleID int) ([]*Comment, int64, error) {
	var allComments []Comment
	var total int64

	if err := r.db.Model(&Comment{}).Where("article_id = ? AND is_public = ?", articleID, 1).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章评论总数失败: %w", err)
	}
	if err := r.db.Where("article_id = ? AND is_public = ?", articleID, 1).Order("created_at desc").Find(&allComments).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章评论失败: %w", err)
	}

	return buildCommentTreeAndSort(allComments), total, nil
}

// DeleteComment 递归删除评论及其所有子评论。
func (r *Repository) DeleteComment(id int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var commentsToDelete []Comment
		var findChildren func(parentID int)
		findChildren = func(parentID int) {
			var children []Comment
			tx.Where("parent_id = ?", parentID).Find(&children)
			for _, child := range children {
				commentsToDelete = append(commentsToDelete, child)
				findChildren(child.ID)
			}
		}

		var selfComment Comment
		if err := tx.First(&selfComment, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("删除评论失败: %w", ErrCommentNotFound)
			}
			return fmt.Errorf("查询评论失败: %w", err)
		}
		commentsToDelete = append(commentsToDelete, selfComment)
		findChildren(id)

		idsToDelete := make([]int, 0, len(commentsToDelete))
		for _, item := range commentsToDelete {
			idsToDelete = append(idsToDelete, item.ID)
		}
		if len(idsToDelete) > 0 {
			if err := tx.Delete(&Comment{}, idsToDelete).Error; err != nil {
				return fmt.Errorf("批量删除评论失败: %w", err)
			}
		}
		return nil
	})
}

// GetAllComments 获取所有评论列表并按根评论分页。
func (r *Repository) GetAllComments(page, pageSize int) ([]*Comment, int64, error) {
	var allComments []Comment
	var total int64

	if err := r.db.Model(&Comment{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询评论总数失败: %w", err)
	}
	if err := r.db.Order("created_at desc").Find(&allComments).Error; err != nil {
		return nil, 0, fmt.Errorf("查询评论失败: %w", err)
	}

	rootComments := buildCommentTreeAndSort(allComments)
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > len(rootComments) {
		end = len(rootComments)
	}
	if offset > len(rootComments) {
		offset = len(rootComments)
	}

	return rootComments[offset:end], total, nil
}

// UpdateComment 更新评论。
func (r *Repository) UpdateComment(comment *Comment) error {
	if err := r.db.Save(comment).Error; err != nil {
		return fmt.Errorf("更新评论失败: %w", err)
	}
	return nil
}

// Count 获取评论总数。
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&Comment{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("查询评论总数失败: %w", err)
	}
	return count, nil
}

func buildCommentTreeAndSort(allComments []Comment) []*Comment {
	commentMap := make(map[int]*Comment, len(allComments))
	for i := range allComments {
		commentMap[allComments[i].ID] = &allComments[i]
	}

	var rootComments []*Comment
	for i := range allComments {
		current := &allComments[i]
		if current.ParentID != nil && *current.ParentID != 0 {
			if parent, ok := commentMap[*current.ParentID]; ok {
				if parent.Children == nil {
					parent.Children = make([]*Comment, 0)
				}
				parent.Children = append(parent.Children, current)
			} else {
				rootComments = append(rootComments, current)
			}
		} else {
			rootComments = append(rootComments, current)
		}
	}

	sort.Slice(rootComments, func(i, j int) bool {
		return rootComments[i].CreatedAt.Time.After(rootComments[j].CreatedAt.Time)
	})
	for _, root := range rootComments {
		sort.Slice(root.Children, func(i, j int) bool {
			return root.Children[i].CreatedAt.Time.After(root.Children[j].CreatedAt.Time)
		})
	}
	return rootComments
}
