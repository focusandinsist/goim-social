package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"goim-social/apps/comment-service/model"
	"goim-social/pkg/database"
)

// commentDAO 评论数据访问实现
type commentDAO struct {
	db *database.PostgreSQL
}

// NewCommentDAO 创建评论DAO实例
func NewCommentDAO(db *database.PostgreSQL) CommentDAO {
	return &commentDAO{
		db: db,
	}
}

// CreateComment 创建评论
func (d *commentDAO) CreateComment(ctx context.Context, comment *model.Comment) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建评论
		if err := tx.Create(comment).Error; err != nil {
			return err
		}

		// 如果是回复，更新父评论的回复数
		if comment.ParentID > 0 {
			if err := d.incrementReplyCountTx(tx, comment.ParentID, 1); err != nil {
				return err
			}
		}

		// 更新统计数据
		return d.incrementCommentStatsTx(tx, comment.ObjectID, comment.ObjectType, "total_count", 1)
	})
}

// GetComment 获取评论
func (d *commentDAO) GetComment(ctx context.Context, commentID int64) (*model.Comment, error) {
	var comment model.Comment
	err := d.db.WithContext(ctx).Where("id = ?", commentID).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// UpdateComment 更新评论
func (d *commentDAO) UpdateComment(ctx context.Context, comment *model.Comment) error {
	return d.db.WithContext(ctx).Save(comment).Error
}

// DeleteComment 删除评论（软删除）
func (d *commentDAO) DeleteComment(ctx context.Context, commentID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取评论信息
		var comment model.Comment
		if err := tx.Where("id = ?", commentID).First(&comment).Error; err != nil {
			return err
		}

		// 标记为已删除
		if err := tx.Model(&comment).Update("status", model.CommentStatusDeleted).Error; err != nil {
			return err
		}

		// 如果是回复，减少父评论的回复数
		if comment.ParentID > 0 {
			if err := d.incrementReplyCountTx(tx, comment.ParentID, -1); err != nil {
				return err
			}
		}

		// 更新统计数据
		return d.incrementCommentStatsTx(tx, comment.ObjectID, comment.ObjectType, "total_count", -1)
	})
}

// GetComments 获取评论列表
func (d *commentDAO) GetComments(ctx context.Context, params *model.GetCommentsParams) ([]*model.Comment, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.Comment{})

	// 构建查询条件
	if params.ObjectID > 0 {
		query = query.Where("object_id = ?", params.ObjectID)
	}
	if params.ObjectType != "" {
		query = query.Where("object_type = ?", params.ObjectType)
	}
	if params.ParentID >= 0 {
		query = query.Where("parent_id = ?", params.ParentID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	} else {
		// 默认只显示已通过的评论
		query = query.Where("status = ?", model.CommentStatusApproved)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 构建排序
	orderBy := d.buildOrderBy(params.SortBy, params.SortOrder)
	query = query.Order(orderBy)

	// 分页
	if params.Page <= 0 {
		params.Page = model.DefaultPage
	}
	if params.PageSize <= 0 {
		params.PageSize = model.DefaultPageSize
	}
	if params.PageSize > model.MaxPageSize {
		params.PageSize = model.MaxPageSize
	}

	offset := (params.Page - 1) * params.PageSize
	query = query.Offset(int(offset)).Limit(int(params.PageSize))

	var comments []*model.Comment
	if err := query.Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	// 如果需要包含回复，加载回复数据
	if params.IncludeReplies && len(comments) > 0 {
		if err := d.loadReplies(ctx, comments, params.MaxReplyCount); err != nil {
			return nil, 0, err
		}
	}

	return comments, total, nil
}

// GetUserComments 获取用户评论
func (d *commentDAO) GetUserComments(ctx context.Context, params *model.GetUserCommentsParams) ([]*model.Comment, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.Comment{}).Where("user_id = ?", params.UserID)

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	if params.Page <= 0 {
		params.Page = model.DefaultPage
	}
	if params.PageSize <= 0 {
		params.PageSize = model.DefaultPageSize
	}
	if params.PageSize > model.MaxPageSize {
		params.PageSize = model.MaxPageSize
	}

	offset := (params.Page - 1) * params.PageSize
	query = query.Order("created_at DESC").Offset(int(offset)).Limit(int(params.PageSize))

	var comments []*model.Comment
	err := query.Find(&comments).Error
	return comments, total, err
}

// GetCommentsByParent 获取父评论的回复
func (d *commentDAO) GetCommentsByParent(ctx context.Context, parentID int64, limit int32) ([]*model.Comment, error) {
	var comments []*model.Comment
	query := d.db.WithContext(ctx).Where("parent_id = ? AND status = ?", parentID, model.CommentStatusApproved).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(int(limit))
	}

	err := query.Find(&comments).Error
	return comments, err
}

// GetCommentTree 获取评论树
func (d *commentDAO) GetCommentTree(ctx context.Context, rootID int64, maxDepth int) ([]*model.Comment, error) {
	var comments []*model.Comment
	query := d.db.WithContext(ctx).Where("root_id = ? AND status = ?", rootID, model.CommentStatusApproved).
		Order("parent_id ASC, created_at ASC")

	err := query.Find(&comments).Error
	return comments, err
}

// UpdateCommentStatus 更新评论状态
func (d *commentDAO) UpdateCommentStatus(ctx context.Context, commentID int64, status string) error {
	return d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("status", status).Error
}

// BatchUpdateCommentStatus 批量更新评论状态
func (d *commentDAO) BatchUpdateCommentStatus(ctx context.Context, commentIDs []int64, status string) error {
	return d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id IN ?", commentIDs).
		Update("status", status).Error
}

// IncrementReplyCount 增加回复数
func (d *commentDAO) IncrementReplyCount(ctx context.Context, commentID int64, delta int32) error {
	return d.incrementReplyCountTx(d.db.WithContext(ctx), commentID, delta)
}

// incrementReplyCountTx 在事务中增加回复数
func (d *commentDAO) incrementReplyCountTx(tx *gorm.DB, commentID int64, delta int32) error {
	return tx.Model(&model.Comment{}).
		Where("id = ?", commentID).
		UpdateColumn("reply_count", gorm.Expr("reply_count + ?", delta)).Error
}

// IncrementLikeCount 增加点赞数
func (d *commentDAO) IncrementLikeCount(ctx context.Context, commentID int64, delta int32) error {
	return d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ?", commentID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// UpdateHotStatus 更新热门状态
func (d *commentDAO) UpdateHotStatus(ctx context.Context, commentID int64, isHot bool) error {
	return d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("is_hot", isHot).Error
}

// UpdatePinStatus 更新置顶状态
func (d *commentDAO) UpdatePinStatus(ctx context.Context, commentID int64, isPinned bool) error {
	return d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("is_pinned", isPinned).Error
}

// 辅助方法

// buildOrderBy 构建排序语句
func (d *commentDAO) buildOrderBy(sortBy, sortOrder string) string {
	if sortOrder != model.SortOrderAsc && sortOrder != model.SortOrderDesc {
		sortOrder = model.SortOrderDesc
	}

	switch sortBy {
	case model.SortByTime:
		return fmt.Sprintf("created_at %s", sortOrder)
	case model.SortByHot:
		return fmt.Sprintf("is_hot DESC, like_count %s, created_at %s", sortOrder, sortOrder)
	case model.SortByLike:
		return fmt.Sprintf("like_count %s, created_at %s", sortOrder, sortOrder)
	default:
		// 默认按时间排序，置顶评论在前
		return fmt.Sprintf("is_pinned DESC, created_at %s", sortOrder)
	}
}

// loadReplies 加载回复数据
func (d *commentDAO) loadReplies(ctx context.Context, comments []*model.Comment, maxReplyCount int32) error {
	if maxReplyCount <= 0 {
		maxReplyCount = model.DefaultReplyShow
	}
	if maxReplyCount > model.MaxReplyShow {
		maxReplyCount = model.MaxReplyShow
	}

	for _, comment := range comments {
		if comment.ReplyCount > 0 {
			replies, err := d.GetCommentsByParent(ctx, comment.ID, maxReplyCount)
			if err != nil {
				return err
			}
			comment.Replies = replies
		}
	}

	return nil
}

// GetCommentStats 获取评论统计
func (d *commentDAO) GetCommentStats(ctx context.Context, objectID int64, objectType string) (*model.CommentStats, error) {
	var stats model.CommentStats
	err := d.db.WithContext(ctx).Where("object_id = ? AND object_type = ?", objectID, objectType).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果统计记录不存在，创建一个新的
			stats = model.CommentStats{
				ObjectID:   objectID,
				ObjectType: objectType,
			}
			if createErr := d.db.WithContext(ctx).Create(&stats).Error; createErr != nil {
				return nil, createErr
			}
		} else {
			return nil, err
		}
	}
	return &stats, nil
}

// GetBatchCommentStats 批量获取评论统计
func (d *commentDAO) GetBatchCommentStats(ctx context.Context, objectIDs []int64, objectType string) ([]*model.CommentStats, error) {
	var statsList []*model.CommentStats
	err := d.db.WithContext(ctx).Where("object_id IN ? AND object_type = ?", objectIDs, objectType).Find(&statsList).Error
	return statsList, err
}

// UpdateCommentStats 更新评论统计
func (d *commentDAO) UpdateCommentStats(ctx context.Context, stats *model.CommentStats) error {
	return d.db.WithContext(ctx).Save(stats).Error
}

// IncrementCommentStats 增加评论统计
func (d *commentDAO) IncrementCommentStats(ctx context.Context, objectID int64, objectType string, field string, delta int64) error {
	return d.incrementCommentStatsTx(d.db.WithContext(ctx), objectID, objectType, field, delta)
}

// incrementCommentStatsTx 在事务中增加评论统计
func (d *commentDAO) incrementCommentStatsTx(tx *gorm.DB, objectID int64, objectType string, field string, delta int64) error {
	// 先尝试更新
	result := tx.Model(&model.CommentStats{}).
		Where("object_id = ? AND object_type = ?", objectID, objectType).
		UpdateColumn(field, gorm.Expr(field+" + ?", delta))

	if result.Error != nil {
		return result.Error
	}

	// 如果没有更新到记录，说明统计记录不存在，需要创建
	if result.RowsAffected == 0 {
		stats := &model.CommentStats{
			ObjectID:   objectID,
			ObjectType: objectType,
		}
		// 根据字段设置初始值
		switch field {
		case "total_count":
			stats.TotalCount = delta
		case "approved_count":
			stats.ApprovedCount = delta
		case "pending_count":
			stats.PendingCount = delta
		case "today_count":
			stats.TodayCount = delta
		case "hot_count":
			stats.HotCount = delta
		}
		return tx.Create(stats).Error
	}

	return nil
}

// CreateModerationLog 创建审核日志
func (d *commentDAO) CreateModerationLog(ctx context.Context, log *model.CommentModerationLog) error {
	return d.db.WithContext(ctx).Create(log).Error
}

// GetModerationLogs 获取审核日志
func (d *commentDAO) GetModerationLogs(ctx context.Context, commentID int64) ([]*model.CommentModerationLog, error) {
	var logs []*model.CommentModerationLog
	err := d.db.WithContext(ctx).Where("comment_id = ?", commentID).Order("created_at DESC").Find(&logs).Error
	return logs, err
}

// AddCommentLike 添加评论点赞
func (d *commentDAO) AddCommentLike(ctx context.Context, commentID, userID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查是否已点赞
		var count int64
		if err := tx.Model(&model.CommentLike{}).
			Where("comment_id = ? AND user_id = ?", commentID, userID).
			Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return fmt.Errorf("已经点赞过了")
		}

		// 创建点赞记录
		like := &model.CommentLike{
			CommentID: commentID,
			UserID:    userID,
		}
		if err := tx.Create(like).Error; err != nil {
			return err
		}

		// 增加点赞数
		return d.incrementLikeCountTx(tx, commentID, 1)
	})
}

// RemoveCommentLike 移除评论点赞
func (d *commentDAO) RemoveCommentLike(ctx context.Context, commentID, userID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除点赞记录
		result := tx.Where("comment_id = ? AND user_id = ?", commentID, userID).Delete(&model.CommentLike{})
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("未找到点赞记录")
		}

		// 减少点赞数
		return d.incrementLikeCountTx(tx, commentID, -1)
	})
}

// incrementLikeCountTx 在事务中增加点赞数
func (d *commentDAO) incrementLikeCountTx(tx *gorm.DB, commentID int64, delta int32) error {
	return tx.Model(&model.Comment{}).
		Where("id = ?", commentID).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", delta)).Error
}

// IsCommentLiked 检查是否已点赞
func (d *commentDAO) IsCommentLiked(ctx context.Context, commentID, userID int64) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.CommentLike{}).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetCommentLikeCount 获取评论点赞数
func (d *commentDAO) GetCommentLikeCount(ctx context.Context, commentID int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.CommentLike{}).
		Where("comment_id = ?", commentID).
		Count(&count).Error
	return count, err
}

// BatchGetComments 批量获取评论
func (d *commentDAO) BatchGetComments(ctx context.Context, commentIDs []int64) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := d.db.WithContext(ctx).Where("id IN ?", commentIDs).Find(&comments).Error
	return comments, err
}

// BatchDeleteComments 批量删除评论
func (d *commentDAO) BatchDeleteComments(ctx context.Context, commentIDs []int64) error {
	return d.db.WithContext(ctx).Model(&model.Comment{}).
		Where("id IN ?", commentIDs).
		Update("status", model.CommentStatusDeleted).Error
}

// GetPendingComments 获取待审核评论
func (d *commentDAO) GetPendingComments(ctx context.Context, page, pageSize int32) ([]*model.Comment, int64, error) {
	return d.getCommentsByStatus(ctx, model.CommentStatusPending, page, pageSize)
}

// GetCommentsByStatus 根据状态获取评论
func (d *commentDAO) GetCommentsByStatus(ctx context.Context, status string, page, pageSize int32) ([]*model.Comment, int64, error) {
	return d.getCommentsByStatus(ctx, status, page, pageSize)
}

// getCommentsByStatus 根据状态获取评论的内部实现
func (d *commentDAO) getCommentsByStatus(ctx context.Context, status string, page, pageSize int32) ([]*model.Comment, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.Comment{}).Where("status = ?", status)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	if page <= 0 {
		page = model.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = model.DefaultPageSize
	}
	if pageSize > model.MaxPageSize {
		pageSize = model.MaxPageSize
	}

	offset := (page - 1) * pageSize
	query = query.Order("created_at DESC").Offset(int(offset)).Limit(int(pageSize))

	var comments []*model.Comment
	err := query.Find(&comments).Error
	return comments, total, err
}

// CleanDeletedComments 清理已删除的评论
func (d *commentDAO) CleanDeletedComments(ctx context.Context, beforeTime time.Time) (int64, error) {
	result := d.db.WithContext(ctx).Unscoped().
		Where("status = ? AND updated_at < ?", model.CommentStatusDeleted, beforeTime).
		Delete(&model.Comment{})
	return result.RowsAffected, result.Error
}

// CleanOldModerationLogs 清理旧的审核日志
func (d *commentDAO) CleanOldModerationLogs(ctx context.Context, beforeTime time.Time) (int64, error) {
	result := d.db.WithContext(ctx).Where("created_at < ?", beforeTime).Delete(&model.CommentModerationLog{})
	return result.RowsAffected, result.Error
}
