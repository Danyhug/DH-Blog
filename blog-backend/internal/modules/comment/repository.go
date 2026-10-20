package comment

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"dh-blog/internal/model"

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

// GetCommentsByArticleID 按顶层评论分页获取某篇文章的公开评论树。
//
// 返回三个值：当前页的评论树、顶层评论总数（分页用）、公开评论总数（「N 条评论」用）。
// 分页只切顶层，回复跟着自己的父评论整棵带出来 —— 把一条对话切到两页读者接不上。
func (r *Repository) GetCommentsByArticleID(articleID, page, pageSize int) ([]*Comment, int64, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// 父评论被隐藏时，它的公开子回复在树上没有爹，buildCommentTreeAndSort 会把它们当成顶层评论。
	// 分页要和这个口径一致，否则这几条回复在「顶层」里数不到、翻页时凭空消失。
	var hiddenIDs []int
	if err := r.db.Model(&Comment{}).
		Where("article_id = ? AND is_public = ?", articleID, 0).
		Pluck("id", &hiddenIDs).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("查询隐藏评论失败: %w", err)
	}

	rootScope := func() *gorm.DB {
		query := r.db.Model(&Comment{}).Where("article_id = ? AND is_public = ?", articleID, 1)
		if len(hiddenIDs) > 0 {
			return query.Where("(parent_id IS NULL OR parent_id = 0 OR parent_id IN ?)", hiddenIDs)
		}
		return query.Where("(parent_id IS NULL OR parent_id = 0)")
	}

	var rootTotal int64
	if err := rootScope().Count(&rootTotal).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("查询顶层评论总数失败: %w", err)
	}

	var roots []Comment
	if err := rootScope().
		Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&roots).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("查询顶层评论失败: %w", err)
	}

	// 逐层把当前页这几棵树的子孙捞出来，层数等于树深，不会扫全表。
	// seen 兜住 parent_id 被写出环的脏数据，否则这里会死循环。
	collected := append([]Comment(nil), roots...)
	seen := make(map[int]struct{}, len(roots))
	frontier := make([]int, 0, len(roots))
	for _, root := range roots {
		seen[root.ID] = struct{}{}
		frontier = append(frontier, root.ID)
	}
	for len(frontier) > 0 {
		var children []Comment
		if err := r.db.Where("is_public = ? AND parent_id IN ?", 1, frontier).
			Order("created_at desc").
			Find(&children).Error; err != nil {
			return nil, 0, 0, fmt.Errorf("查询评论回复失败: %w", err)
		}
		frontier = make([]int, 0, len(children))
		for _, child := range children {
			if _, ok := seen[child.ID]; ok {
				continue
			}
			seen[child.ID] = struct{}{}
			collected = append(collected, child)
			frontier = append(frontier, child.ID)
		}
	}

	var commentTotal int64
	if err := r.db.Model(&Comment{}).
		Where("article_id = ? AND is_public = ?", articleID, 1).
		Count(&commentTotal).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("查询文章评论总数失败: %w", err)
	}

	return buildCommentTreeAndSort(collected), rootTotal, commentTotal, nil
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

// GetCommentGroups 获取后台评论列表并按文章分页。
func (r *Repository) GetCommentGroups(page, pageSize int) ([]*ArticleCommentGroup, int64, error) {
	type articleCommentSummary struct {
		ArticleID    int
		ArticleTitle string
		CommentCount int64
	}

	var summaries []articleCommentSummary
	var total int64

	if err := r.db.Model(&Comment{}).Distinct("article_id").Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询评论文章总数失败: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := r.db.Model(&Comment{}).
		Select(`comments.article_id,
			COALESCE(articles.title, '文章已删除') AS article_title,
			COUNT(comments.id) AS comment_count`).
		Joins("LEFT JOIN articles ON articles.id = comments.article_id AND articles.deleted_at IS NULL").
		Group("comments.article_id, articles.title").
		Order("MAX(comments.created_at) DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&summaries).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章评论分组失败: %w", err)
	}

	if len(summaries) == 0 {
		return []*ArticleCommentGroup{}, total, nil
	}

	articleIDs := make([]int, 0, len(summaries))
	for _, summary := range summaries {
		articleIDs = append(articleIDs, summary.ArticleID)
	}

	var allComments []Comment
	if err := r.db.Where("article_id IN ?", articleIDs).Order("created_at desc").Find(&allComments).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文章评论失败: %w", err)
	}

	commentsByArticle := make(map[int][]Comment, len(summaries))
	for _, item := range allComments {
		commentsByArticle[item.ArticleID] = append(commentsByArticle[item.ArticleID], item)
	}

	groups := make([]*ArticleCommentGroup, 0, len(summaries))
	for _, summary := range summaries {
		articleComments := commentsByArticle[summary.ArticleID]
		latestCommentTime := model.JSONTime{}
		if len(articleComments) > 0 {
			latestCommentTime = articleComments[0].CreatedAt
		}
		groups = append(groups, &ArticleCommentGroup{
			ArticleID:         summary.ArticleID,
			ArticleTitle:      summary.ArticleTitle,
			CommentCount:      summary.CommentCount,
			LatestCommentTime: latestCommentTime,
			Children:          buildCommentTreeAndSort(articleComments),
		})
	}

	return groups, total, nil
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
		return rootComments[i].CreatedAt.After(rootComments[j].CreatedAt.Time)
	})
	var sortChildren func(comments []*Comment)
	sortChildren = func(comments []*Comment) {
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].CreatedAt.After(comments[j].CreatedAt.Time)
		})
		for _, item := range comments {
			sortChildren(item.Children)
		}
	}
	for _, root := range rootComments {
		sortChildren(root.Children)
	}
	return rootComments
}
