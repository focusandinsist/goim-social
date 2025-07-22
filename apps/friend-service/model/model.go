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
