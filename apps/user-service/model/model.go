package model

import (
	"time"
)

// User 用户模型
type User struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Username  string    `json:"username" gorm:"type:varchar(50);uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"type:varchar(255);not null"`
	Email     string    `json:"email" gorm:"type:varchar(100);uniqueIndex;not null"`
	Nickname  string    `json:"nickname" gorm:"type:varchar(100);not null"`
	Avatar    string    `json:"avatar" gorm:"type:varchar(500)"`
	Status    int       `json:"status" gorm:"default:0;index"` // 0:正常 1:禁用 2:删除
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
