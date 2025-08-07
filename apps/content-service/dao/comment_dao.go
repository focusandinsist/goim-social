package dao

import (
	"context"
	"fmt"

	"goim-social/apps/content-service/model"
)

// ==================== 评论相关方法实现 ====================

// CreateComment 创建评论
func (d *contentDAO) CreateComment(ctx context.Context, comment *model.Comment) error {
	return d.db.GetDB().WithContext(ctx).Create(comment).Error
}

// GetComment 获取评论
func (d *contentDAO) GetComment(ctx context.Context, commentID int64) (*model.Comment, error) {
	var comment model.Comment
	err := d.db.GetDB().WithContext(ctx).First(&comment, commentID).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// UpdateComment 更新评论内容
func (d *contentDAO) UpdateComment(ctx context.Context, commentID int64, content string) error {
	return d.db.GetDB().WithContext(ctx).Model(&model.Comment{}).
		Where("id = ?", commentID).
		Update("content", content).Error
}

// DeleteComment 删除评论
func (d *contentDAO) DeleteComment(ctx context.Context, commentID int64) error {
	return d.db.GetDB().WithContext(ctx).Delete(&model.Comment{}, commentID).Error
}

// GetComments 获取评论列表
func (d *contentDAO) GetComments(ctx context.Context, targetID int64, targetType string, parentID int64, sortBy, sortOrder string, page, pageSize int32) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := d.db.GetDB().WithContext(ctx).Model(&model.Comment{}).
		Where("target_id = ? AND target_type = ? AND parent_id = ?", targetID, targetType, parentID)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	orderClause := "created_at DESC" // 默认排序
	if sortBy != "" {
		switch sortBy {
		case "time":
			orderClause = fmt.Sprintf("created_at %s", sortOrder)
		case "hot":
			orderClause = fmt.Sprintf("like_count %s, created_at DESC", sortOrder)
		case "like":
			orderClause = fmt.Sprintf("like_count %s", sortOrder)
		}
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order(orderClause).
		Offset(int(offset)).
		Limit(int(pageSize)).
		Find(&comments).Error

	return comments, total, err
}

// GetCommentReplies 获取评论回复
func (d *contentDAO) GetCommentReplies(ctx context.Context, commentID int64, sortBy, sortOrder string, page, pageSize int32) ([]*model.Comment, int64, error) {
	var replies []*model.Comment
	var total int64

	query := d.db.GetDB().WithContext(ctx).Model(&model.Comment{}).
		Where("parent_id = ?", commentID)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	orderClause := "created_at ASC" // 回复默认按时间升序
	if sortBy != "" {
		switch sortBy {
		case "time":
			orderClause = fmt.Sprintf("created_at %s", sortOrder)
		case "like":
			orderClause = fmt.Sprintf("like_count %s", sortOrder)
		}
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order(orderClause).
		Offset(int(offset)).
		Limit(int(pageSize)).
		Find(&replies).Error

	return replies, total, err
}

// GetCommentsByUser 获取用户的评论列表
func (d *contentDAO) GetCommentsByUser(ctx context.Context, userID int64, page, pageSize int32) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := d.db.GetDB().WithContext(ctx).Model(&model.Comment{}).
		Where("user_id = ?", userID)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Find(&comments).Error

	return comments, total, err
}

// UpdateCommentReplyCount 更新评论回复数
func (d *contentDAO) UpdateCommentReplyCount(ctx context.Context, commentID int64, delta int32) error {
	return d.db.GetDB().WithContext(ctx).Model(&model.Comment{}).
		Where("id = ?", commentID).
		UpdateColumn("reply_count", d.db.GetDB().Raw("reply_count + ?", delta)).Error
}

// UpdateTargetCommentCount 更新目标对象的评论数
func (d *contentDAO) UpdateTargetCommentCount(ctx context.Context, targetID int64, targetType string, delta int64) error {
	switch targetType {
	case model.TargetTypeContent:
		return d.db.GetDB().WithContext(ctx).Model(&model.Content{}).
			Where("id = ?", targetID).
			UpdateColumn("comment_count", d.db.GetDB().Raw("comment_count + ?", delta)).Error
	case model.TargetTypeComment:
		return d.db.GetDB().WithContext(ctx).Model(&model.Comment{}).
			Where("id = ?", targetID).
			UpdateColumn("reply_count", d.db.GetDB().Raw("reply_count + ?", delta)).Error
	default:
		return fmt.Errorf("unsupported target type: %s", targetType)
	}
}
