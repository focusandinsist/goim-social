package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/social-service/converter"
	"goim-social/apps/social-service/service"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	svc       *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, converter *converter.Converter, logger logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:       svc,
		converter: converter,
		logger:    logger,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(engine *gin.Engine) {
	// 好友相关路由
	friendGroup := engine.Group("/api/v1/friend")
	{
		friendGroup.POST("/send_request", h.SendFriendRequest)
		friendGroup.POST("/accept_request", h.AcceptFriendRequest)
		friendGroup.POST("/reject_request", h.RejectFriendRequest)
		friendGroup.POST("/delete", h.DeleteFriend)
		friendGroup.POST("/list", h.GetFriendList)
		friendGroup.POST("/apply_list", h.GetFriendApplyList)
	}

	// 群组相关路由
	groupGroup := engine.Group("/api/v1/group")
	{
		groupGroup.POST("/create", h.CreateGroup)
		groupGroup.POST("/info", h.GetGroup)
		groupGroup.POST("/update", h.UpdateGroup)
		groupGroup.POST("/join", h.JoinGroup)
		groupGroup.POST("/leave", h.LeaveGroup)
		groupGroup.POST("/members", h.GetGroupMembers)
	}

	// 社交关系验证路由
	socialGroup := engine.Group("/api/v1/social")
	{
		socialGroup.POST("/validate_friendship", h.ValidateFriendship)
		socialGroup.POST("/validate_membership", h.ValidateGroupMembership)
		socialGroup.POST("/user_info", h.GetUserSocialInfo)
	}
}

// ============ 好友相关处理器 ============

// SendFriendRequest 发送好友申请
func (h *HTTPHandler) SendFriendRequest(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.ApplyFriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid send friend request", logger.F("error", err.Error()))
		res := &rest.ApplyFriendResponse{Success: false, Message: "Invalid request format"}
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err := h.svc.SendFriendRequest(ctx, req.UserId, req.FriendId, req.Remark)

	var res *rest.ApplyFriendResponse
	if err != nil {
		h.logger.Error(ctx, "Send friend request failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		res = &rest.ApplyFriendResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Send friend request successful",
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		res = &rest.ApplyFriendResponse{Success: true, Message: "好友申请发送成功"}
	}

	httpx.WriteObject(c, res, err)
}

// AcceptFriendRequest 接受好友申请
func (h *HTTPHandler) AcceptFriendRequest(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.RespondFriendApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid accept friend request", logger.F("error", err.Error()))
		res := &rest.RespondFriendApplyResponse{Success: false, Message: "Invalid request format"}
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err := h.svc.AcceptFriendRequest(ctx, req.UserId, req.ApplicantId, "")

	var res *rest.RespondFriendApplyResponse
	if err != nil {
		h.logger.Error(ctx, "Accept friend request failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("applicantID", req.ApplicantId))
		res = &rest.RespondFriendApplyResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Accept friend request successful",
			logger.F("userID", req.UserId),
			logger.F("applicantID", req.ApplicantId))
		res = &rest.RespondFriendApplyResponse{Success: true, Message: "好友申请接受成功"}
	}

	httpx.WriteObject(c, res, err)
}

// RejectFriendRequest 拒绝好友申请
func (h *HTTPHandler) RejectFriendRequest(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.RespondFriendApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid reject friend request", logger.F("error", err.Error()))
		res := &rest.RespondFriendApplyResponse{Success: false, Message: "Invalid request format"}
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err := h.svc.RejectFriendRequest(ctx, req.UserId, req.ApplicantId, "")

	var res *rest.RespondFriendApplyResponse
	if err != nil {
		h.logger.Error(ctx, "Reject friend request failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("applicantID", req.ApplicantId))
		res = &rest.RespondFriendApplyResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Reject friend request successful",
			logger.F("userID", req.UserId),
			logger.F("applicantID", req.ApplicantId))
		res = &rest.RespondFriendApplyResponse{Success: true, Message: "好友申请拒绝成功"}
	}

	httpx.WriteObject(c, res, err)
}

// DeleteFriend 删除好友
func (h *HTTPHandler) DeleteFriend(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.DeleteFriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid delete friend request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorDeleteFriendResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err := h.svc.DeleteFriend(ctx, req.UserId, req.FriendId)

	var res *rest.DeleteFriendResponse
	if err != nil {
		h.logger.Error(ctx, "Delete friend failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		res = h.converter.BuildErrorDeleteFriendResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Delete friend successful",
			logger.F("userID", req.UserId),
			logger.F("friendID", req.FriendId))
		res = h.converter.BuildDeleteFriendResponse(true, "删除好友成功")
	}

	httpx.WriteObject(c, res, err)
}

// GetFriendList 获取好友列表
func (h *HTTPHandler) GetFriendList(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.ListFriendsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get friend list request", logger.F("error", err.Error()))
		res := &rest.ListFriendsResponse{Success: false, Message: "Invalid request format"}
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	friends, err := h.svc.GetFriendList(ctx, req.UserId)

	var res *rest.ListFriendsResponse
	if err != nil {
		h.logger.Error(ctx, "Get friend list failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		res = &rest.ListFriendsResponse{Success: false, Message: err.Error()}
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

		res = &rest.ListFriendsResponse{
			Success: true,
			Message: "获取好友列表成功",
			Friends: friendInfos,
			Total:   int32(len(friends)),
		}
	}

	httpx.WriteObject(c, res, err)
}

// GetFriendApplyList 获取好友申请列表
func (h *HTTPHandler) GetFriendApplyList(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.ListFriendApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get friend apply list request", logger.F("error", err.Error()))
		res := &rest.ListFriendApplyResponse{}
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	applies, err := h.svc.GetFriendApplyList(ctx, req.UserId)

	var res *rest.ListFriendApplyResponse
	if err != nil {
		h.logger.Error(ctx, "Get friend apply list failed",
			logger.F("error", err.Error()),
			logger.F("userID", req.UserId))
		res = &rest.ListFriendApplyResponse{}
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

		res = &rest.ListFriendApplyResponse{Applies: applyInfos}
	}

	httpx.WriteObject(c, res, err)
}

// ============ 群组相关处理器 ============

// CreateGroup 创建群组
func (h *HTTPHandler) CreateGroup(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid create group request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorCreateGroupResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.OwnerId)

	group, err := h.svc.CreateGroup(ctx, req.OwnerId, req.Name, req.Description, req.Avatar, req.IsPublic, req.MaxMembers, req.MemberIds)

	var res *rest.CreateGroupResponse
	if err != nil {
		h.logger.Error(ctx, "Create group failed",
			logger.F("error", err.Error()),
			logger.F("ownerID", req.OwnerId),
			logger.F("name", req.Name))
		res = h.converter.BuildErrorCreateGroupResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Create group successful",
			logger.F("groupID", group.ID),
			logger.F("ownerID", req.OwnerId),
			logger.F("name", req.Name))
		res = h.converter.BuildCreateGroupResponse(true, "创建群组成功", group)
	}

	httpx.WriteObject(c, res, err)
}

// GetGroup 获取群组信息
func (h *HTTPHandler) GetGroup(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.GetGroupInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get group request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetGroupResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, req.GroupId)

	group, err := h.svc.GetGroup(ctx, req.GroupId)

	var res *rest.GetGroupInfoResponse
	if err != nil {
		h.logger.Error(ctx, "Get group failed",
			logger.F("error", err.Error()),
			logger.F("groupID", req.GroupId))
		res = &rest.GetGroupInfoResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Get group successful",
			logger.F("groupID", req.GroupId),
			logger.F("name", group.Name))

		groupInfo := &rest.GroupInfo{
			Id:           group.ID,
			Name:         group.Name,
			Description:  group.Description,
			Avatar:       group.Avatar,
			OwnerId:      group.OwnerID,
			MemberCount:  group.MemberCount,
			MaxMembers:   group.MaxMembers,
			IsPublic:     group.IsPublic,
			Announcement: group.Announcement,
			CreatedAt:    group.CreatedAt.Unix(),
			UpdatedAt:    group.UpdatedAt.Unix(),
		}

		res = &rest.GetGroupInfoResponse{
			Success: true,
			Message: "获取群组信息成功",
			Group:   groupInfo,
		}
	}

	httpx.WriteObject(c, res, err)
}

// UpdateGroup 更新群组信息
func (h *HTTPHandler) UpdateGroup(c *gin.Context) {
	ctx := c.Request.Context()

	// 使用发布公告的请求结构，因为它包含了群组ID和用户ID
	var req rest.PublishAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid update group request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorUpdateGroupResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, req.GroupId)
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err := h.svc.UpdateGroup(ctx, req.GroupId, req.UserId, "", "", "", req.Content)

	var res *rest.PublishAnnouncementResponse
	if err != nil {
		h.logger.Error(ctx, "Update group failed",
			logger.F("error", err.Error()),
			logger.F("groupID", req.GroupId),
			logger.F("userID", req.UserId))
		res = &rest.PublishAnnouncementResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Update group successful",
			logger.F("groupID", req.GroupId),
			logger.F("userID", req.UserId))
		res = &rest.PublishAnnouncementResponse{Success: true, Message: "更新群组信息成功"}
	}

	httpx.WriteObject(c, res, err)
}

// JoinGroup 加入群组
func (h *HTTPHandler) JoinGroup(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.JoinGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid join group request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorJoinGroupResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, req.GroupId)
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err := h.svc.JoinGroup(ctx, req.GroupId, req.UserId)

	var res *rest.JoinGroupResponse
	if err != nil {
		h.logger.Error(ctx, "Join group failed",
			logger.F("error", err.Error()),
			logger.F("groupID", req.GroupId),
			logger.F("userID", req.UserId))
		res = h.converter.BuildErrorJoinGroupResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Join group successful",
			logger.F("groupID", req.GroupId),
			logger.F("userID", req.UserId))
		res = h.converter.BuildJoinGroupResponse(true, "加入群组成功")
	}

	httpx.WriteObject(c, res, err)
}

// LeaveGroup 离开群组
func (h *HTTPHandler) LeaveGroup(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.LeaveGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid leave group request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorLeaveGroupResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, req.GroupId)
	ctx = tracecontext.WithUserID(ctx, req.UserId)

	err := h.svc.LeaveGroup(ctx, req.GroupId, req.UserId)

	var res *rest.LeaveGroupResponse
	if err != nil {
		h.logger.Error(ctx, "Leave group failed",
			logger.F("error", err.Error()),
			logger.F("groupID", req.GroupId),
			logger.F("userID", req.UserId))
		res = h.converter.BuildErrorLeaveGroupResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Leave group successful",
			logger.F("groupID", req.GroupId),
			logger.F("userID", req.UserId))
		res = h.converter.BuildLeaveGroupResponse(true, "离开群组成功")
	}

	httpx.WriteObject(c, res, err)
}

// GetGroupMembers 获取群成员列表
func (h *HTTPHandler) GetGroupMembers(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.GetGroupInfoRequest // 复用获取群组信息的请求
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get group members request", logger.F("error", err.Error()))
		res := h.converter.BuildErrorGetGroupMembersResponse("Invalid request format")
		httpx.WriteObject(c, res, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithGroupID(ctx, req.GroupId)

	members, err := h.svc.GetGroupMembers(ctx, req.GroupId)

	// 使用群组信息响应，包含成员信息
	var res *rest.GetGroupInfoResponse
	if err != nil {
		h.logger.Error(ctx, "Get group members failed",
			logger.F("error", err.Error()),
			logger.F("groupID", req.GroupId))
		res = &rest.GetGroupInfoResponse{Success: false, Message: err.Error()}
	} else {
		h.logger.Info(ctx, "Get group members successful",
			logger.F("groupID", req.GroupId),
			logger.F("count", len(members)))

		// 转换成员信息
		memberInfos := make([]*rest.GroupMemberInfo, len(members))
		for i, member := range members {
			memberInfos[i] = &rest.GroupMemberInfo{
				UserId:   member.UserID,
				GroupId:  member.GroupID,
				Role:     member.Role,
				Nickname: member.Nickname,
				JoinedAt: member.JoinedAt.Unix(),
			}
		}

		res = &rest.GetGroupInfoResponse{
			Success: true,
			Message: "获取群成员列表成功",
			Members: memberInfos,
		}
	}

	httpx.WriteObject(c, res, err)
}

// ============ 社交关系验证处理器 ============

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
