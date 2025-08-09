package dao

import (
	"context"
	"time"

	"goim-social/apps/content-service/model"
	"goim-social/pkg/database"

	"gorm.io/gorm"
)

// contentDAO 内容数据访问实现
type contentDAO struct {
	db *database.PostgreSQL
}

// NewContentDAO 创建内容DAO实例
func NewContentDAO(db *database.PostgreSQL) ContentDAO {
	return &contentDAO{db: db}
}

// CreateContent 创建内容
func (d *contentDAO) CreateContent(ctx context.Context, content *model.Content) error {
	return d.db.WithContext(ctx).Create(content).Error
}

// GetContent 获取内容基本信息
func (d *contentDAO) GetContent(ctx context.Context, contentID int64) (*model.Content, error) {
	var content model.Content
	err := d.db.WithContext(ctx).Where("id = ?", contentID).First(&content).Error
	if err != nil {
		return nil, err
	}
	return &content, nil
}

// GetContentWithRelations 获取内容及其关联数据
func (d *contentDAO) GetContentWithRelations(ctx context.Context, contentID int64) (*model.Content, error) {
	var content model.Content
	err := d.db.WithContext(ctx).
		Preload("MediaFiles").
		Preload("Tags").
		Preload("Topics").
		Where("id = ?", contentID).
		First(&content).Error
	if err != nil {
		return nil, err
	}
	return &content, nil
}

// UpdateContent 更新内容
func (d *contentDAO) UpdateContent(ctx context.Context, content *model.Content) error {
	return d.db.WithContext(ctx).Save(content).Error
}

// DeleteContent 删除内容
func (d *contentDAO) DeleteContent(ctx context.Context, contentID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除媒体文件
		if err := tx.Where("content_id = ?", contentID).Delete(&model.ContentMediaFile{}).Error; err != nil {
			return err
		}

		// 删除标签关联
		if err := tx.Where("content_id = ?", contentID).Delete(&model.ContentTagRelation{}).Error; err != nil {
			return err
		}

		// 删除话题关联
		if err := tx.Where("content_id = ?", contentID).Delete(&model.ContentTopicRelation{}).Error; err != nil {
			return err
		}

		// 删除状态日志
		if err := tx.Where("content_id = ?", contentID).Delete(&model.ContentStatusLog{}).Error; err != nil {
			return err
		}

		// 删除内容
		return tx.Where("id = ?", contentID).Delete(&model.Content{}).Error
	})
}

// GetUserContents 获取用户内容列表
func (d *contentDAO) GetUserContents(ctx context.Context, authorID int64, status string, page, pageSize int32) ([]*model.Content, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.Content{}).Where("author_id = ?", authorID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页和排序
	offset := (page - 1) * pageSize
	query = query.Order("created_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Preload("MediaFiles").
		Preload("Tags").
		Preload("Topics")

	var contents []*model.Content
	err := query.Find(&contents).Error
	return contents, total, err
}

// GetContentsByStatus 根据状态获取内容列表
func (d *contentDAO) GetContentsByStatus(ctx context.Context, status string, page, pageSize int32) ([]*model.Content, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.Content{}).Where("status = ?", status)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页和排序
	offset := (page - 1) * pageSize
	query = query.Order("created_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Preload("MediaFiles").
		Preload("Tags").
		Preload("Topics")

	var contents []*model.Content
	err := query.Find(&contents).Error
	return contents, total, err
}

// GetContentStats 获取内容统计
func (d *contentDAO) GetContentStats(ctx context.Context, authorID int64) (*model.ContentStats, error) {
	var stats model.ContentStats

	query := d.db.WithContext(ctx).Model(&model.Content{})
	if authorID > 0 {
		query = query.Where("author_id = ?", authorID)
	}

	// 总内容数
	if err := query.Count(&stats.TotalContents).Error; err != nil {
		return nil, err
	}

	// 各状态内容数
	statusCounts := []struct {
		Status string
		Count  int64
	}{}

	statusQuery := query.Select("status, COUNT(*) as count").Group("status")
	if err := statusQuery.Scan(&statusCounts).Error; err != nil {
		return nil, err
	}

	for _, sc := range statusCounts {
		switch sc.Status {
		case model.ContentStatusPublished:
			stats.PublishedContents = sc.Count
		case model.ContentStatusDraft:
			stats.DraftContents = sc.Count
		case model.ContentStatusPending:
			stats.PendingContents = sc.Count
		}
	}

	// 总浏览数和点赞数
	var sums struct {
		TotalViews int64
		TotalLikes int64
	}

	sumQuery := query.Select("SUM(view_count) as total_views, SUM(like_count) as total_likes")
	if err := sumQuery.Scan(&sums).Error; err != nil {
		return nil, err
	}

	stats.TotalViews = sums.TotalViews
	stats.TotalLikes = sums.TotalLikes

	return &stats, nil
}

// IncrementViewCount 增加浏览次数
func (d *contentDAO) IncrementViewCount(ctx context.Context, contentID int64) error {
	return d.db.WithContext(ctx).Model(&model.Content{}).
		Where("id = ?", contentID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// CreateMediaFile 创建媒体文件
func (d *contentDAO) CreateMediaFile(ctx context.Context, mediaFile *model.ContentMediaFile) error {
	return d.db.WithContext(ctx).Create(mediaFile).Error
}

// GetMediaFiles 获取内容的媒体文件列表
func (d *contentDAO) GetMediaFiles(ctx context.Context, contentID int64) ([]*model.ContentMediaFile, error) {
	var mediaFiles []*model.ContentMediaFile
	err := d.db.WithContext(ctx).
		Where("content_id = ?", contentID).
		Order("sort_order ASC, created_at ASC").
		Find(&mediaFiles).Error
	return mediaFiles, err
}

// DeleteMediaFiles 删除内容的所有媒体文件
func (d *contentDAO) DeleteMediaFiles(ctx context.Context, contentID int64) error {
	return d.db.WithContext(ctx).Where("content_id = ?", contentID).Delete(&model.ContentMediaFile{}).Error
}

// CreateTag 创建标签
func (d *contentDAO) CreateTag(ctx context.Context, tag *model.ContentTag) error {
	return d.db.WithContext(ctx).Create(tag).Error
}

// GetTag 获取标签
func (d *contentDAO) GetTag(ctx context.Context, tagID int64) (*model.ContentTag, error) {
	var tag model.ContentTag
	err := d.db.WithContext(ctx).Where("id = ?", tagID).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetTagByName 根据名称获取标签
func (d *contentDAO) GetTagByName(ctx context.Context, name string) (*model.ContentTag, error) {
	var tag model.ContentTag
	err := d.db.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetTags 获取标签列表
func (d *contentDAO) GetTags(ctx context.Context, keyword string, page, pageSize int32) ([]*model.ContentTag, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.ContentTag{})

	if keyword != "" {
		keyword = "%" + keyword + "%"
		query = query.Where("name ILIKE ?", keyword)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页和排序
	offset := (page - 1) * pageSize
	query = query.Order("usage_count DESC, created_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize))

	var tags []*model.ContentTag
	err := query.Find(&tags).Error
	return tags, total, err
}

// UpdateTagUsageCount 更新标签使用次数
func (d *contentDAO) UpdateTagUsageCount(ctx context.Context, tagID int64, delta int64) error {
	if delta > 0 {
		return d.db.WithContext(ctx).Model(&model.ContentTag{}).
			Where("id = ?", tagID).
			UpdateColumn("usage_count", gorm.Expr("usage_count + ?", delta)).Error
	} else if delta < 0 {
		return d.db.WithContext(ctx).Model(&model.ContentTag{}).
			Where("id = ? AND usage_count >= ?", tagID, -delta).
			UpdateColumn("usage_count", gorm.Expr("usage_count + ?", delta)).Error
	}
	return nil
}

// CreateTopic 创建话题
func (d *contentDAO) CreateTopic(ctx context.Context, topic *model.ContentTopic) error {
	return d.db.WithContext(ctx).Create(topic).Error
}

// GetTopic 获取话题
func (d *contentDAO) GetTopic(ctx context.Context, topicID int64) (*model.ContentTopic, error) {
	var topic model.ContentTopic
	err := d.db.WithContext(ctx).Where("id = ?", topicID).First(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

// GetTopicByName 根据名称获取话题
func (d *contentDAO) GetTopicByName(ctx context.Context, name string) (*model.ContentTopic, error) {
	var topic model.ContentTopic
	err := d.db.WithContext(ctx).Where("name = ?", name).First(&topic).Error
	if err != nil {
		return nil, err
	}
	return &topic, nil
}

// GetTopics 获取话题列表
func (d *contentDAO) GetTopics(ctx context.Context, keyword string, hotOnly bool, page, pageSize int32) ([]*model.ContentTopic, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.ContentTopic{})

	if keyword != "" {
		keyword = "%" + keyword + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", keyword, keyword)
	}

	if hotOnly {
		query = query.Where("is_hot = ?", true)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页和排序
	offset := (page - 1) * pageSize
	query = query.Order("content_count DESC, created_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize))

	var topics []*model.ContentTopic
	err := query.Find(&topics).Error
	return topics, total, err
}

// UpdateTopicContentCount 更新话题内容数量
func (d *contentDAO) UpdateTopicContentCount(ctx context.Context, topicID int64, delta int64) error {
	if delta > 0 {
		return d.db.WithContext(ctx).Model(&model.ContentTopic{}).
			Where("id = ?", topicID).
			UpdateColumn("content_count", gorm.Expr("content_count + ?", delta)).Error
	} else if delta < 0 {
		return d.db.WithContext(ctx).Model(&model.ContentTopic{}).
			Where("id = ? AND content_count >= ?", topicID, -delta).
			UpdateColumn("content_count", gorm.Expr("content_count + ?", delta)).Error
	}
	return nil
}

// AddContentTags 添加内容标签关联
func (d *contentDAO) AddContentTags(ctx context.Context, contentID int64, tagIDs []int64) error {
	if len(tagIDs) == 0 {
		return nil
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除现有关联
		if err := tx.Where("content_id = ?", contentID).Delete(&model.ContentTagRelation{}).Error; err != nil {
			return err
		}

		// 添加新关联
		for _, tagID := range tagIDs {
			relation := &model.ContentTagRelation{
				ContentID: contentID,
				TagID:     tagID,
				CreatedAt: time.Now(),
			}
			if err := tx.Create(relation).Error; err != nil {
				return err
			}

			// 更新标签使用次数
			if err := d.UpdateTagUsageCount(ctx, tagID, 1); err != nil {
				return err
			}
		}

		return nil
	})
}

// RemoveContentTags 移除内容标签关联
func (d *contentDAO) RemoveContentTags(ctx context.Context, contentID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取现有标签关联
		var relations []model.ContentTagRelation
		if err := tx.Where("content_id = ?", contentID).Find(&relations).Error; err != nil {
			return err
		}

		// 删除关联
		if err := tx.Where("content_id = ?", contentID).Delete(&model.ContentTagRelation{}).Error; err != nil {
			return err
		}

		// 更新标签使用次数
		for _, relation := range relations {
			if err := d.UpdateTagUsageCount(ctx, relation.TagID, -1); err != nil {
				return err
			}
		}

		return nil
	})
}

// AddContentTopics 添加内容话题关联
func (d *contentDAO) AddContentTopics(ctx context.Context, contentID int64, topicIDs []int64) error {
	if len(topicIDs) == 0 {
		return nil
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除现有关联
		if err := tx.Where("content_id = ?", contentID).Delete(&model.ContentTopicRelation{}).Error; err != nil {
			return err
		}

		// 添加新关联
		for _, topicID := range topicIDs {
			relation := &model.ContentTopicRelation{
				ContentID: contentID,
				TopicID:   topicID,
				CreatedAt: time.Now(),
			}
			if err := tx.Create(relation).Error; err != nil {
				return err
			}

			// 更新话题内容数量
			if err := d.UpdateTopicContentCount(ctx, topicID, 1); err != nil {
				return err
			}
		}

		return nil
	})
}

// RemoveContentTopics 移除内容话题关联
func (d *contentDAO) RemoveContentTopics(ctx context.Context, contentID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取现有话题关联
		var relations []model.ContentTopicRelation
		if err := tx.Where("content_id = ?", contentID).Find(&relations).Error; err != nil {
			return err
		}

		// 删除关联
		if err := tx.Where("content_id = ?", contentID).Delete(&model.ContentTopicRelation{}).Error; err != nil {
			return err
		}

		// 更新话题内容数量
		for _, relation := range relations {
			if err := d.UpdateTopicContentCount(ctx, relation.TopicID, -1); err != nil {
				return err
			}
		}

		return nil
	})
}

// CreateStatusLog 创建状态变更日志
func (d *contentDAO) CreateStatusLog(ctx context.Context, log *model.ContentStatusLog) error {
	return d.db.WithContext(ctx).Create(log).Error
}

// GetStatusLogs 获取内容状态变更日志
func (d *contentDAO) GetStatusLogs(ctx context.Context, contentID int64) ([]*model.ContentStatusLog, error) {
	var logs []*model.ContentStatusLog
	err := d.db.WithContext(ctx).
		Where("content_id = ?", contentID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}
