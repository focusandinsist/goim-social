package model

import "time"

// Group 群组
type Group struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string    `json:"name" gorm:"type:varchar(100);not null;index"`
	Description  string    `json:"description" gorm:"type:text"`
	Avatar       string    `json:"avatar" gorm:"type:varchar(500)"`
	OwnerID      int64     `json:"owner_id" gorm:"not null;index"`
	MemberCount  int32     `json:"member_count" gorm:"default:1"`
	MaxMembers   int32     `json:"max_members" gorm:"default:500"`
	IsPublic     bool      `json:"is_public" gorm:"default:true"`
	Announcement string    `json:"announcement" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName .
func (Group) TableName() string {
	return "groups"
}

// GroupMember 群成员
type GroupMember struct {
	ID       int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID   int64     `json:"user_id" gorm:"not null;index"`
	GroupID  int64     `json:"group_id" gorm:"not null;index"`
	Role     string    `json:"role" gorm:"type:varchar(20);default:'member'"` // owner, admin, member
	Nickname string    `json:"nickname" gorm:"type:varchar(100)"`
	JoinedAt time.Time `json:"joined_at" gorm:"autoCreateTime"`
}

// TableName .
func (GroupMember) TableName() string {
	return "group_members"
}

// GroupInvitation 群邀请
type GroupInvitation struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID   int64     `json:"group_id" gorm:"not null;index"`
	InviterID int64     `json:"inviter_id" gorm:"not null"`
	InviteeID int64     `json:"invitee_id" gorm:"not null;index"`
	Status    string    `json:"status" gorm:"type:varchar(20);default:'pending'"` // pending, accepted, rejected, expired
	Message   string    `json:"message" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	ExpiredAt time.Time `json:"expired_at"`
}

// TableName .
func (GroupInvitation) TableName() string {
	return "group_invitations"
}

// GroupJoinRequest 加群申请
type GroupJoinRequest struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID   int64     `json:"group_id" gorm:"not null;index"`
	UserID    int64     `json:"user_id" gorm:"not null;index"`
	Status    string    `json:"status" gorm:"type:varchar(20);default:'pending'"` // pending, approved, rejected
	Reason    string    `json:"reason" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName .
func (GroupJoinRequest) TableName() string {
	return "group_join_requests"
}

// GroupAnnouncement 群公告
type GroupAnnouncement struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	GroupID   int64     `json:"group_id" gorm:"not null;index"`
	UserID    int64     `json:"user_id" gorm:"not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName .
func (GroupAnnouncement) TableName() string {
	return "group_announcements"
}
