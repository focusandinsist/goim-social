package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// DeleteFriend 删除好友
func (h *HTTPHandler) DeleteFriend(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DeleteFriendRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid delete friend request", logger.F("error", err.Error()))
		res := h.converter.BuildDeleteFriendResponse(false, "Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.DeleteFriend(ctx, req.GetUserId(), req.GetFriendId())

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Delete friend failed", logger.F("error", err.Error()))
	} else {
		message = "删除好友成功"
	}

	res := h.converter.BuildDeleteFriendResponse(err == nil, message)
	httpx.WriteObject(c, res, err)
}

// ListFriends 查询好友列表
func (h *HTTPHandler) ListFriends(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ListFriendsRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid list friends request", logger.F("error", err.Error()))
		res := h.converter.BuildListFriendsResponse(false, "Invalid request format", nil, 0, 0)
		httpx.WriteObject(c, res, err)
		return
	}

	friends, err := h.svc.ListFriends(ctx, req.GetUserId())

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "List friends failed", logger.F("error", err.Error()))
	} else {
		message = "查询成功"
	}

	res := h.converter.BuildListFriendsResponse(err == nil, message, friends, req.GetPage(), req.GetPageSize())
	httpx.WriteObject(c, res, err)
}

// GetFriendProfile 获取好友简介
func (h *HTTPHandler) GetFriendProfile(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.FriendProfileRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid friend profile request", logger.F("error", err.Error()))
		res := h.converter.BuildFriendProfileResponse(false, "Invalid request format", "", "", "", "", 0)
		httpx.WriteObject(c, res, err)
		return
	}

	friend, err := h.svc.GetFriend(ctx, req.GetUserId(), req.GetFriendId())

	var message, alias string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Get friend failed", logger.F("error", err.Error()))
	} else {
		message = "获取好友简介成功"
		if friend != nil {
			alias = friend.Remark
		}
	}

	// TODO: 从用户服务获取用户信息
	res := h.converter.BuildFriendProfileResponse(err == nil, message, alias, "好友昵称", "头像URL", "未知", 25)
	httpx.WriteObject(c, res, err)
}

// UpdateFriendRemark 更新好友备注
func (h *HTTPHandler) UpdateFriendRemark(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.SetFriendAliasRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid set friend alias request", logger.F("error", err.Error()))
		res := h.converter.BuildSetFriendAliasResponse(false, "Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.UpdateFriendRemark(ctx, req.GetUserId(), req.GetFriendId(), req.GetAlias())

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Update friend remark failed", logger.F("error", err.Error()))
	} else {
		message = "更新好友备注成功"
	}

	res := h.converter.BuildSetFriendAliasResponse(err == nil, message)
	httpx.WriteObject(c, res, err)
}

// ListFriendApply 查询好友申请列表
func (h *HTTPHandler) ListFriendApply(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ListFriendApplyRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid list friend apply request", logger.F("error", err.Error()))
		res := h.converter.BuildListFriendApplyResponse(nil)
		httpx.WriteObject(c, res, err)
		return
	}

	applies, err := h.svc.ListFriendApply(ctx, req.UserId)
	if err != nil {
		h.log.Error(ctx, "List friend apply failed", logger.F("error", err.Error()))
		applies = nil // 确保错误时返回空列表
	}

	res := h.converter.BuildListFriendApplyResponse(applies)
	httpx.WriteObject(c, res, err)
}

// ApplyFriend 申请加好友
func (h *HTTPHandler) ApplyFriend(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.ApplyFriendRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid apply friend request", logger.F("error", err.Error()))
		res := h.converter.BuildApplyFriendResponse(false, "Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.ApplyFriend(ctx, req.UserId, req.FriendId, req.Remark)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Apply friend failed", logger.F("error", err.Error()))
	} else {
		message = "好友申请已提交，等待对方处理"
	}

	res := h.converter.BuildApplyFriendResponse(err == nil, message)
	httpx.WriteObject(c, res, err)
}

// RespondFriendApply 回应好友申请
func (h *HTTPHandler) RespondFriendApply(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.RespondFriendApplyRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid respond friend apply request", logger.F("error", err.Error()))
		res := h.converter.BuildRespondFriendApplyResponse(false, "Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.RespondFriendApply(ctx, req.UserId, req.ApplicantId, req.Agree)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Respond friend apply failed", logger.F("error", err.Error()))
	} else {
		message = "操作成功"
	}

	res := h.converter.BuildRespondFriendApplyResponse(err == nil, message)
	httpx.WriteObject(c, res, err)
}
