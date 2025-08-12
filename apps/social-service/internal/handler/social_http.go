package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// ValidateFriendship 验证好友关系
func (h *HTTPHandler) ValidateFriendship(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.GetFriendRequest // 复用获取好友的请求
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid validate friendship request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorValidateFriendshipResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	isFriend, err := h.svc.ValidateFriendship(ctx, req.UserId, req.FriendId)

	// 使用好友响应类型
	var res *rest.GetFriendResponse
	if err != nil {
		h.logger.Error(ctx, "Validate friendship failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		res = &rest.GetFriendResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Validate friendship successful",
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId),
			logger.F("isFriend", isFriend))
		message := "不是好友关系"
		if isFriend {
			message = "是好友关系"
		}
		res = &rest.GetFriendResponse{Success: true, Message: message}
	}

	httpx.WriteObject(c, res, err)
}

// ValidateGroupMembership 验证群成员关系
func (h *HTTPHandler) ValidateGroupMembership(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.ValidateGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid validate group membership request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorValidateGroupMembershipResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, req.GroupId)
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	isMember, err := h.svc.ValidateGroupMembership(ctx, req.UserId, req.GroupId)

	// 使用群成员验证响应类型
	var res *rest.ValidateGroupMemberResponse
	if err != nil {
		h.logger.Error(ctx, "Validate group membership failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("groupID", req.GroupId))
		res = &rest.ValidateGroupMemberResponse{Success: false, Message: err.Error(), IsMember: false}
	} else {
		h.logger.Info(ctx, "Validate group membership successful",
			logger.F("userID", req.UserId),
			logger.F("groupID", req.GroupId),
			logger.F("isMember", isMember))
		message := "不是群成员"
		if isMember {
			message = "是群成员"
		}
		res = &rest.ValidateGroupMemberResponse{Success: true, Message: message, IsMember: isMember}
	}

	httpx.WriteObject(c, res, err)
}

// GetUserSocialInfo 获取用户社交信息
func (h *HTTPHandler) GetUserSocialInfo(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.GetUserGroupsRequest // 复用获取用户群组的请求
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user social info request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetUserSocialInfoResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	socialInfo, err := h.svc.GetUserSocialInfo(ctx, req.UserId)

	// 使用用户群组响应类型
	var res *rest.GetUserGroupsResponse
	if err != nil {
		h.logger.Error(ctx, "Get user social info failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		res = &rest.GetUserGroupsResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Get user social info successful",
			logger.F("userID", req.UserId),
			logger.F("friendCount", socialInfo.FriendCount),
			logger.F("groupCount", socialInfo.GroupCount))
		res = &rest.GetUserGroupsResponse{
			Success: true,
			Message: fmt.Sprintf("用户有%d个好友，%d个群组", socialInfo.FriendCount, socialInfo.GroupCount),
			Total:   int32(socialInfo.GroupCount),
		}
	}

	httpx.WriteObject(c, res, err)
}
