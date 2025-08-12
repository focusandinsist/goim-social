package model

// 默认配置
const (
	DefaultMaxMembers = 500
	DefaultPageSize   = 20
)

// 群成员角色
const (
	RoleOwner  = "owner"  // 群主
	RoleAdmin  = "admin"  // 管理员
	RoleMember = "member" // 普通成员
)

// 邀请进群状态
const (
	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
	InvitationStatusRejected = "rejected"
	InvitationStatusExpired  = "expired"
)

// 加群申请状态
const (
	JoinRequestStatusPending  = "pending"
	JoinRequestStatusApproved = "approved"
	JoinRequestStatusRejected = "rejected"
)

// 好友申请状态
const (
	FriendApplyStatusPending  = "pending"
	FriendApplyStatusAccepted = "accepted"
	FriendApplyStatusRejected = "rejected"
)
