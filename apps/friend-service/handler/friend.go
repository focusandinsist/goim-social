package handler

import (
	"github.com/gin-gonic/gin"

	"websocket-server/api/rest"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/utils"
)

// DeleteFriend 删除好友
func (h *HTTPHandler) DeleteFriend(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DeleteFriendRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid delete friend request", logger.F("error", err.Error()))
		res := &rest.DeleteFriendResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	err := h.svc.DeleteFriend(ctx, req.GetUserId(), req.GetFriendId())
	res := &rest.DeleteFriendResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "删除好友成功"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Delete friend failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// ListFriends 查询好友列表
func (h *HTTPHandler) ListFriends(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ListFriendsRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid list friends request", logger.F("error", err.Error()))
		res := &rest.ListFriendsResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	friends, err := h.svc.ListFriends(ctx, req.GetUserId())
	res := &rest.ListFriendsResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "查询成功"
		}(),
		Friends: func() []*rest.FriendInfo {
			if err != nil {
				return []*rest.FriendInfo{}
			}
			var pbFriends []*rest.FriendInfo
			for _, friend := range friends {
				pbFriends = append(pbFriends, &rest.FriendInfo{
					UserId:    friend.UserID,
					FriendId:  friend.FriendID,
					Remark:    friend.Remark,
					CreatedAt: friend.CreatedAt.Unix(),
				})
			}
			return pbFriends
		}(),
		Total:    int32(len(friends)),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}
	if err != nil {
		h.log.Error(ctx, "List friends failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// GetFriendProfile 获取好友简介
func (h *HTTPHandler) GetFriendProfile(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.FriendProfileRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid friend profile request", logger.F("error", err.Error()))
		res := &rest.FriendProfileResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	friend, err := h.svc.GetFriend(ctx, req.GetUserId(), req.GetFriendId())
	res := &rest.FriendProfileResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取好友简介成功"
		}(),
		Alias: func() string {
			if friend != nil {
				return friend.Remark
			}
			return ""
		}(),
		Nickname: "好友昵称",  // TODO: 从用户服务获取
		Avatar:   "头像URL", // TODO: 从用户服务获取
		Age:      25,      // TODO: 从用户服务获取
		Gender:   "未知",    // TODO: 从用户服务获取
	}
	if err != nil {
		h.log.Error(ctx, "Get friend failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// UpdateFriendRemark 更新好友备注
func (h *HTTPHandler) UpdateFriendRemark(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.SetFriendAliasRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid set friend alias request", logger.F("error", err.Error()))
		res := &rest.SetFriendAliasResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	err := h.svc.UpdateFriendRemark(ctx, req.GetUserId(), req.GetFriendId(), req.GetAlias())
	res := &rest.SetFriendAliasResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "更新好友备注成功"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Update friend remark failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// ListFriendApply 查询好友申请列表
func (h *HTTPHandler) ListFriendApply(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ListFriendApplyRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid list friend apply request", logger.F("error", err.Error()))
		res := &rest.ListFriendApplyResponse{
			Applies: []*rest.FriendApplyInfo{},
		}
		utils.WriteObject(c, res, err)
		return
	}

	applies, err := h.svc.ListFriendApply(ctx, req.UserId)

	var pbApplies []*rest.FriendApplyInfo
	if err == nil {
		for _, a := range applies {
			pbApplies = append(pbApplies, &rest.FriendApplyInfo{
				ApplicantId: a.ApplicantID,
				Remark:      a.Remark,
				Timestamp:   a.CreatedAt.Unix(),
				Status:      a.Status,
			})
		}
	}

	res := &rest.ListFriendApplyResponse{
		Applies: pbApplies,
	}
	if err != nil {
		h.log.Error(ctx, "List friend apply failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// ApplyFriend 申请加好友
func (h *HTTPHandler) ApplyFriend(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ApplyFriendRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid apply friend request", logger.F("error", err.Error()))
		res := &rest.ApplyFriendResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	err := h.svc.ApplyFriend(ctx, req.UserId, req.FriendId, req.Remark)
	res := &rest.ApplyFriendResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "好友申请已提交，等待对方处理"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Apply friend failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// RespondFriendApply 回应好友申请
func (h *HTTPHandler) RespondFriendApply(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.RespondFriendApplyRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid respond friend apply request", logger.F("error", err.Error()))
		res := &rest.RespondFriendApplyResponse{
			Success: false,
			Message: "Invalid request format",
		}
		utils.WriteObject(c, res, err)
		return
	}

	err := h.svc.RespondFriendApply(ctx, req.UserId, req.ApplicantId, req.Agree)
	res := &rest.RespondFriendApplyResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "操作成功"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Respond friend apply failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}
