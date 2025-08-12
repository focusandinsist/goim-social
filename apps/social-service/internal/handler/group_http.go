package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// CreateGroup 创建群组
func (h *HTTPHandler) CreateGroup(c *gin.Context) {
	var (
		ctx  = c.Request.Context()
		req  rest.CreateGroupRequest
		resp *rest.CreateGroupResponse
		err  error
	)

	if err = c.Bind(&req); err != nil {
		h.logger.Error(ctx, "Invalid create group request", logger.F("error", err.Error()))
		resp = h.converter.BuildErrorCreateGroupResponse("Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	// 将业务信息添加到context
	ctx = tracecontext.WithUserID(ctx, req.OwnerId)

	group, err := h.svc.CreateGroup(ctx, req.OwnerId, req.Name, req.Description, req.Avatar, req.IsPublic, req.MaxMembers, req.MemberIds)
	if err != nil {
		h.logger.Error(ctx, "Create group failed",
			logger.F("error", err.Error()),
			logger.F("ownerID", req.OwnerId),
			logger.F("name", req.Name))
		resp = h.converter.BuildErrorCreateGroupResponse(err.Error())
	} else {
		h.logger.Info(ctx, "Create group successful",
			logger.F("groupID", group.ID),
			logger.F("ownerID", req.OwnerId),
			logger.F("name", req.Name))
		resp = h.converter.BuildCreateGroupResponse(true, "创建群组成功", group)
	}

	httpx.WriteObject(c, resp, err)
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
