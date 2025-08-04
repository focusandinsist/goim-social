package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// InviteToGroup 邀请加入群组
func (h *HTTPHandler) InviteToGroup(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.InviteToGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid invite to group request", logger.F("error", err.Error()))
		resp := h.converter.BuildInviteToGroupResponse(false, "Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	err := h.svc.InviteToGroup(ctx, req.GroupId, req.InviterId, req.UserId, "welcome")

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Invite to group failed", logger.F("error", err.Error()))
	} else {
		message = "邀请发送成功"
	}

	resp := h.converter.BuildInviteToGroupResponse(err == nil, message)
	httpx.WriteObject(c, resp, err)
}

// PublishAnnouncement 发布群公告
func (h *HTTPHandler) PublishAnnouncement(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.PublishAnnouncementRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid publish announcement request", logger.F("error", err.Error()))
		resp := h.converter.BuildPublishAnnouncementResponse(false, "Invalid request format")
		httpx.WriteObject(c, resp, err)
		return
	}

	err := h.svc.PublishAnnouncement(ctx, req.GroupId, req.UserId, req.Content)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Publish announcement failed", logger.F("error", err.Error()))
	} else {
		message = "发布群公告成功"
	}

	resp := h.converter.BuildPublishAnnouncementResponse(err == nil, message)
	httpx.WriteObject(c, resp, err)
}

// GetUserGroups 获取用户群组列表
func (h *HTTPHandler) GetUserGroups(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUserGroupsRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid get user groups request", logger.F("error", err.Error()))
		resp := h.converter.BuildGetUserGroupsResponse(false, "Invalid request format", nil, 0, 0, 0)
		httpx.WriteObject(c, resp, err)
		return
	}

	groups, total, err := h.svc.GetUserGroups(ctx, req.UserId, req.Page, req.PageSize)

	var message string
	if err != nil {
		message = err.Error()
		h.log.Error(ctx, "Get user groups failed", logger.F("error", err.Error()))
	} else {
		message = "获取用户群组列表成功"
	}

	resp := h.converter.BuildGetUserGroupsResponse(err == nil, message, groups, total, req.Page, req.PageSize)
	httpx.WriteObject(c, resp, err)
}
