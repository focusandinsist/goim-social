package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// SendFriendRequest 发送好友申请
func (h *HTTPHandler) SendFriendRequest(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.ApplyFriendRequest
		resp *rest.ApplyFriendResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid send friend request", logger.F("error", err.Error()))
		resp = &rest.ApplyFriendResponse{Success: false, Message: "Invalid request format"}
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err = h.svc.SendFriendRequest(ctx, req.UserId, req.FriendId, req.Remark)
	if err != nil {
		h.logger.Error(ctx, "Send friend request failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		resp = &rest.ApplyFriendResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Send friend request successful",
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		resp = &rest.ApplyFriendResponse{Success: true, Message: "好友申请发送成功"}
	}

	httpx.WriteObject(c, resp, err)
}

// AcceptFriendRequest 接受好友申请
func (h *HTTPHandler) AcceptFriendRequest(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.RespondFriendApplyRequest
		resp *rest.RespondFriendApplyResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid accept friend request", logger.F("error", err.Error()))
		resp = &rest.RespondFriendApplyResponse{Success: false, Message: "Invalid request format"}
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err = h.svc.AcceptFriendRequest(ctx, req.UserId, req.ApplicantId, "")
	if err != nil {
		h.logger.Error(ctx, "Accept friend request failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("applicantID", req.ApplicantId))
		resp = &rest.RespondFriendApplyResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Accept friend request successful",
			logger.F("userID", req.UserId),
			logger.F("applicantID", req.ApplicantId))
		resp = &rest.RespondFriendApplyResponse{Success: true, Message: "好友申请接受成功"}
	}

	httpx.WriteObject(c, resp, err)
}

// RejectFriendRequest 拒绝好友申请
func (h *HTTPHandler) RejectFriendRequest(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.RespondFriendApplyRequest
		resp *rest.RespondFriendApplyResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid reject friend request", logger.F("error", err.Error()))
		resp = &rest.RespondFriendApplyResponse{Success: false, Message: "Invalid request format"}
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err = h.svc.RejectFriendRequest(ctx, req.UserId, req.ApplicantId, "")
	if err != nil {
		h.logger.Error(ctx, "Reject friend request failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("applicantID", req.ApplicantId))
		resp = &rest.RespondFriendApplyResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Reject friend request successful",
			logger.F("userID", req.UserId),
			logger.F("applicantID", req.ApplicantId))
		resp = &rest.RespondFriendApplyResponse{Success: true, Message: "好友申请拒绝成功"}
	}

	httpx.WriteObject(c, resp, err)
}

// DeleteFriend 删除好友
func (h *HTTPHandler) DeleteFriend(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.DeleteFriendRequest
		resp *rest.DeleteFriendResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete friend request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorDeleteFriendResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err = h.svc.DeleteFriend(ctx, req.UserId, req.FriendId)
	if err != nil {
		h.logger.Error(ctx, "Delete friend failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		resp = h.converter.BuildErrorDeleteFriendResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Delete friend successful",
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		resp = h.converter.BuildDeleteFriendResponse(true, "删除好友成功")
	}

	httpx.WriteObject(c, resp, err)
}

// GetFriendList 获取好友列表
func (h *HTTPHandler) GetFriendList(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.ListFriendsRequest
		resp *rest.ListFriendsResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get friend list request", logger.F("error", err.Error()))
		resp = &rest.ListFriendsResponse{Success: false, Message: "Invalid request format"}
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	friends, err := h.svc.GetFriendList(ctx, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "Get friend list failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		resp = &rest.ListFriendsResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Get friend list successful",
			logger.F("userID", req.UserId),
			logger.F("count", len(friends)))

		// 转换好友信息
		friendInfos := make([]*rest.FriendInfo, len(friends))
		for i, friend := range friends {
			friendInfos[i] = &rest.FriendInfo{
				UserId:    friend.UserID,
				FriendId:  friend.FriendID,
				Remark:    friend.Remark,
				CreatedAt: friend.CreatedAt.Unix(),
			}
		}

		resp = &rest.ListFriendsResponse{
			Success: true,
			Message: "获取好友列表成功",
			Friends: friendInfos,
			Total:   int32(len(friends)),
		}
	}

	httpx.WriteObject(c, resp, err)
}

// GetFriendApplyList 获取好友申请列表
func (h *HTTPHandler) GetFriendApplyList(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.ListFriendApplyRequest
		resp *rest.ListFriendApplyResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid get friend apply list request", logger.F("error", err.Error()))
		resp = &rest.ListFriendApplyResponse{}
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	applies, err := h.svc.GetFriendApplyList(ctx, req.UserId)
	if err != nil {
		h.logger.Error(ctx, "Get friend apply list failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		resp = &rest.ListFriendApplyResponse{}
	} else {
		h.logger.Info(ctx, "Get friend apply list successful",
			logger.F("userID", req.UserId),
			logger.F("count", len(applies)))

		// 转换申请信息
		applyInfos := make([]*rest.FriendApplyInfo, len(applies))
		for i, apply := range applies {
			applyInfos[i] = &rest.FriendApplyInfo{
				ApplicantId: apply.ApplicantID,
				Remark:      apply.Remark,
				Timestamp:   apply.CreatedAt.Unix(),
				Status:      apply.Status,
			}
		}

		resp = &rest.ListFriendApplyResponse{Applies: applyInfos}
	}

	httpx.WriteObject(c, resp, err)
}
