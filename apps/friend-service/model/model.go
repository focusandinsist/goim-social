package model

type Friend struct {
	UserID    int64  `json:"user_id"`
	FriendID  int64  `json:"friend_id"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"created_at"`
}

type AddFriendRequest struct {
	UserID   int64  `json:"user_id"`
	FriendID int64  `json:"friend_id"`
	Remark   string `json:"remark"`
}

type DeleteFriendRequest struct {
	UserID   int64 `json:"user_id"`
	FriendID int64 `json:"friend_id"`
}

type ListFriendsRequest struct {
	UserID int64 `json:"user_id"`
}

type UpdateFriendRemarkRequest struct {
	UserID    int64  `json:"user_id"`
	FriendID  int64  `json:"friend_id"`
	NewRemark string `json:"new_remark"`
}

type GetFriendRequest struct {
	UserID   int64 `json:"user_id"`
	FriendID int64 `json:"friend_id"`
}

type FriendApply struct {
	UserID       int64  `json:"user_id" bson:"user_id"`                                 // 被申请人
	ApplicantID  int64  `json:"applicant_id" bson:"applicant_id"`                       // 申请人
	Remark       string `json:"remark" bson:"remark"`                                   // 申请备注
	Status       string `json:"status" bson:"status"`                                   // pending/accepted/rejected
	Timestamp    int64  `json:"timestamp" bson:"timestamp"`                             // 申请时间
	AgreeTime    int64  `json:"agree_time,omitempty" bson:"agree_time,omitempty"`       // 同意时间
	RejectTime   int64  `json:"reject_time,omitempty" bson:"reject_time,omitempty"`     // 拒绝时间
	AgreeRemark  string `json:"agree_remark,omitempty" bson:"agree_remark,omitempty"`   // 同意时备注
	RejectReason string `json:"reject_reason,omitempty" bson:"reject_reason,omitempty"` // 拒绝原因
}
