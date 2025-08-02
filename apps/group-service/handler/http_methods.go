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
		res := &rest.InviteToGroupResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.InviteToGroup(ctx, req.GroupId, req.InviterId, req.UserId, "welcome")
	res := &rest.InviteToGroupResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "邀请发送成功"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Invite to group failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// PublishAnnouncement 发布群公告
func (h *HTTPHandler) PublishAnnouncement(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.PublishAnnouncementRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid publish announcement request", logger.F("error", err.Error()))
		res := &rest.PublishAnnouncementResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.PublishAnnouncement(ctx, req.GroupId, req.UserId, req.Content)
	res := &rest.PublishAnnouncementResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "发布群公告成功"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Publish announcement failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// GetUserGroups 获取用户群组列表
func (h *HTTPHandler) GetUserGroups(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetUserGroupsRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid get user groups request", logger.F("error", err.Error()))
		res := &rest.GetUserGroupsResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	groups, total, err := h.svc.GetUserGroups(ctx, req.UserId, req.Page, req.PageSize)
	res := &rest.GetUserGroupsResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取用户群组列表成功"
		}(),
		Groups: func() []*rest.GroupInfo {
			if err != nil {
				return []*rest.GroupInfo{}
			}
			var pbGroups []*rest.GroupInfo
			for _, group := range groups {
				pbGroups = append(pbGroups, &rest.GroupInfo{
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
				})
			}
			return pbGroups
		}(),
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	if err != nil {
		h.log.Error(ctx, "Get user groups failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}
