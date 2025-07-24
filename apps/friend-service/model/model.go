package model

import (
	"time"
)

// Friend 好友关系
type Friend struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int64     `json:"user_id" gorm:"not null;index"`    // 用户ID
	FriendID  int64     `json:"friend_id" gorm:"not null;index"`  // 好友ID
	Remark    string    `json:"remark" gorm:"type:varchar(100)"`  // 备注
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"` // 添加时间
}

// TableName 指定表名
func (Friend) TableName() string {
	return "friends"
}

// FriendApply 好友申请
type FriendApply struct {
	ID           int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       int64      `json:"user_id" gorm:"not null;index"`                    // 被申请人
	ApplicantID  int64      `json:"applicant_id" gorm:"not null;index"`               // 申请人
	Remark       string     `json:"remark" gorm:"type:text"`                          // 申请备注
	Status       string     `json:"status" gorm:"type:varchar(20);default:'pending'"` // 状态(pending/accepted/rejected)
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`                 // 申请时间
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime"`                 // 更新时间
	AgreeTime    *time.Time `json:"agree_time,omitempty" gorm:"index"`                // 同意时间
	RejectTime   *time.Time `json:"reject_time,omitempty" gorm:"index"`               // 拒绝时间
	AgreeRemark  string     `json:"agree_remark,omitempty" gorm:"type:varchar(100)"`  // 同意时备注
	RejectReason string     `json:"reject_reason,omitempty" gorm:"type:varchar(200)"` // 拒绝原因
}

// TableName 指定表名
func (FriendApply) TableName() string {
	return "friend_applies"
}
