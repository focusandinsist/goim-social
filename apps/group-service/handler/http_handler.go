package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/group-service/converter"
	"goim-social/apps/group-service/service"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// HTTPHandler HTTP协议处理器
type HTTPHandler struct {
	svc       *service.Service
	converter *converter.Converter
	log       logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, log logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc:       svc,
		converter: converter.NewConverter(),
		log:       log,
	}
}

// RegisterRoutes 注册HTTP路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/group")
	{
		api.POST("/search", h.SearchGroups)              // 搜索群组
		api.POST("/create", h.CreateGroup)               // 创建群组
		api.POST("/disband", h.DisbandGroup)             // 解散群组
		api.POST("/info", h.GetGroupInfo)                // 获取群组信息
		api.POST("/join", h.JoinGroup)                   // 加入群组
		api.POST("/leave", h.LeaveGroup)                 // 退出群组
		api.POST("/kick", h.KickMember)                  // 踢出成员
		api.POST("/invite", h.InviteToGroup)             // 邀请加入群组
		api.POST("/announcement", h.PublishAnnouncement) // 发布群公告
		api.POST("/user_groups", h.GetUserGroups)        // 获取用户群组列表
	}
}

// SearchGroups 搜索群组
func (h *HTTPHandler) SearchGroups(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.SearchGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid search groups request", logger.F("error", err.Error()))
		resp := h.converter.BuildSearchGroupResponse(false, "Invalid request format", nil, 0, 0, 0)
		httpx.WriteObject(c, resp, err)
		return
	}

	groups, total, err := h.svc.SearchGroups(ctx, req.Keyword, req.Page, req.PageSize)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Search groups failed", logger.F("error", err.Error()))
	} else {
		message = "搜索成功"
	}

	resp := h.converter.BuildSearchGroupResponse(err == nil, message, groups, total, req.Page, req.PageSize)
	httpx.WriteObject(c, resp, err)
}

// CreateGroup 创建群组
func (h *HTTPHandler) CreateGroup(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid create group request", logger.F("error", err.Error()))
		resp := h.converter.BuildCreateGroupResponse(false, "Invalid request format", nil)
		httpx.WriteObject(c, resp, err)
		return
	}

	group, err := h.svc.CreateGroup(ctx, req.Name, req.Description, req.Avatar, req.OwnerId, req.IsPublic, req.MaxMembers, req.MemberIds)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Create group failed", logger.F("error", err.Error()))
	} else {
		message = "创建群组成功"
	}

	resp := h.converter.BuildCreateGroupResponse(err == nil, message, group)
	httpx.WriteObject(c, resp, err)
}

// GetGroupInfo 获取群组信息
func (h *HTTPHandler) GetGroupInfo(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetGroupInfoRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid get group info request", logger.F("error", err.Error()))
		resp := h.converter.BuildGetGroupInfoResponse(false, "Invalid request format", nil, nil)
		httpx.WriteObject(c, resp, err)
		return
	}

	group, members, err := h.svc.GetGroupInfo(ctx, req.GroupId, req.UserId)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Get group info failed", logger.F("error", err.Error()))
	} else {
		message = "获取群组信息成功"
	}

	resp := h.converter.BuildGetGroupInfoResponse(err == nil, message, group, members)
	httpx.WriteObject(c, resp, err)
}

// DisbandGroup 解散群组
func (h *HTTPHandler) DisbandGroup(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DisbandGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid disband group request", logger.F("error", err.Error()))
		resp := h.converter.BuildDisbandGroupResponse(false, "Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	err := h.svc.DisbandGroup(ctx, req.GroupId, req.UserId)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Disband group failed", logger.F("error", err.Error()))
	} else {
		message = "解散群组成功"
	}

	resp := h.converter.BuildDisbandGroupResponse(err == nil, message)
	httpx.WriteObject(c, resp, err)
}

// JoinGroup 加入群组
func (h *HTTPHandler) JoinGroup(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.JoinGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid join group request", logger.F("error", err.Error()))
		resp := h.converter.BuildJoinGroupResponse(false, "Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	err := h.svc.JoinGroup(ctx, req.GroupId, req.UserId, req.Reason)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Join group failed", logger.F("error", err.Error()))
	} else {
		message = "加入群组成功"
	}

	resp := h.converter.BuildJoinGroupResponse(err == nil, message)
	httpx.WriteObject(c, resp, err)
}

// LeaveGroup 退出群组
func (h *HTTPHandler) LeaveGroup(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.LeaveGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid leave group request", logger.F("error", err.Error()))
		resp := h.converter.BuildLeaveGroupResponse(false, "Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	err := h.svc.LeaveGroup(ctx, req.GroupId, req.UserId)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Leave group failed", logger.F("error", err.Error()))
	} else {
		message = "退出群组成功"
	}

	resp := h.converter.BuildLeaveGroupResponse(err == nil, message)
	httpx.WriteObject(c, resp, err)
}

// KickMember 踢出成员
func (h *HTTPHandler) KickMember(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.KickMemberRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid kick member request", logger.F("error", err.Error()))
		resp := h.converter.BuildKickMemberResponse(false, "Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	err := h.svc.KickMember(ctx, req.GroupId, req.OperatorId, req.TargetUserId)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Kick member failed", logger.F("error", err.Error()))
	} else {
		message = "踢出成员成功"
	}

	resp := h.converter.BuildKickMemberResponse(err == nil, message)
	httpx.WriteObject(c, resp, err)
}
