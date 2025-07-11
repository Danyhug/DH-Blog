package repository

import (
	"errors"
	"fmt"
	"sort"

	"dh-blog/internal/model"
	"gorm.io/gorm"
)

var (
	ErrCommentNotFound = errors.New("评论不存在")
)

type CommentRepository struct {
	DB *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{DB: db}
}

// AddComment 添加评论
func (r *CommentRepository) AddComment(comment *model.Comment) error {
	err := r.DB.Create(comment).Error
	if err != nil {
		return fmt.Errorf("添加评论失败: %w", err)
	}
	return nil
}

// GetCommentsByArticleID 根据文章 ID 获取评论列表
func (r *CommentRepository) GetCommentsByArticleID(articleID int) ([]*model.Comment, int64, error) {
	var allComments []model.Comment
	var total int64

	// 查询所有与文章相关的评论的总数
	if err := r.DB.Model(&model.Comment{}).Where("article_id = ? AND is_public = ?", articleID, 1).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章评论总数失败: %w", err)
	}

	// 查询所有与文章相关的评论，并按创建时间降序排序
	if err := r.DB.Where("article_id = ? AND is_public = ?", articleID, 1).Order("created_at desc").Find(&allComments).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章评论失败: %w", err)
	}

	// 构建评论树并排序
	rootComments := r.buildCommentTreeAndSort(allComments)

	return rootComments, total, nil
}

// DeleteComment 递归删除评论及其所有子评论
func (r *CommentRepository) DeleteComment(id int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// 查找所有子评论
		var commentsToDelete []model.Comment
		// 递归查询所有子评论的 ID
		var findChildren func(parentID int)
		findChildren = func(parentID int) {
			var children []model.Comment
			tx.Where("parent_id = ?", parentID).Find(&children)
			for _, child := range children {
				commentsToDelete = append(commentsToDelete, child)
				findChildren(child.ID)
			}
		}

		// 添加当前评论到删除列表
		var selfComment model.Comment
		if err := tx.First(&selfComment, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("删除评论失败: %w", ErrCommentNotFound)
			}
			return fmt.Errorf("查询评论失败: %w", err)
		}
		commentsToDelete = append(commentsToDelete, selfComment)

		// 查找并添加所有子评论
		findChildren(id)

		// 批量删除所有评论
		var idsToDelete []int
		for _, comment := range commentsToDelete {
			idsToDelete = append(idsToDelete, comment.ID)
		}

		if len(idsToDelete) > 0 {
			if err := tx.Delete(&model.Comment{}, idsToDelete).Error; err != nil {
				return fmt.Errorf("批量删除评论失败: %w", err)
			}
		}

		return nil
	})
}

// GetAllComments 获取所有评论列表（带分页）
func (r *CommentRepository) GetAllComments(page, pageSize int) ([]*model.Comment, int64, error) {
	var allComments []model.Comment
	var total int64

	// 查询所有评论的总数
	if err := r.DB.Model(&model.Comment{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询评论总数失败: %w", err)
	}

	// 查询所有评论，并按创建时间降序排序
	if err := r.DB.Order("created_at desc").Find(&allComments).Error; err != nil {
		return nil, 0, fmt.Errorf("查询评论失败: %w", err)
	}

	// 构建评论树并排序
	rootComments := r.buildCommentTreeAndSort(allComments)

	// 手动分页
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

// UpdateComment 更新评论
func (r *CommentRepository) UpdateComment(comment *model.Comment) error {
	err := r.DB.Save(comment).Error
	if err != nil {
		return fmt.Errorf("更新评论失败: %w", err)
	}
	return nil
}

// buildCommentTreeAndSort 辅助函数，用于构建评论树并进行排序
func (r *CommentRepository) buildCommentTreeAndSort(allComments []model.Comment) []*model.Comment {
	commentMap := make(map[int]*model.Comment)
	for i := range allComments {
		commentMap[allComments[i].ID] = &allComments[i]
	}

	var rootComments []*model.Comment
	for i := range allComments {
		comment := &allComments[i]
		if comment.ParentID != nil && *comment.ParentID != 0 {
			if parent, ok := commentMap[*comment.ParentID]; ok {
				if parent.Children == nil {
					parent.Children = make([]*model.Comment, 0)
				}
				parent.Children = append(parent.Children, comment)
			} else {
				rootComments = append(rootComments, comment)
			}
		} else {
			rootComments = append(rootComments, comment)
		}
	}

	sort.Slice(rootComments, func(i, j int) bool {
		return rootComments[i].CreatedAt.Time.After(rootComments[j].CreatedAt.Time)
	})
	for _, rc := range rootComments {
		sort.Slice(rc.Children, func(i, j int) bool {
			return rc.Children[i].CreatedAt.Time.After(rc.Children[j].CreatedAt.Time)
		})
	}
	return rootComments
}
