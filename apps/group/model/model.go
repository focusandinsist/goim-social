package model

import (
	"time"
)

type Group struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:varchar(255)" json:"description"`
	OwnerID     int64     `json:"owner_id"`
	MemberIDs   []int64   `gorm:"-" json:"member_ids"` // 这里假设成员ID不做持久化，仅演示
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
