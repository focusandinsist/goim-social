package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/api/rest"
	"goim-social/apps/group-service/service"
	"goim-social/pkg/httpx"
	"goim-social/pkg/logger"
)

// HTTPHandler HTTP协议处理器
type HTTPHandler struct {
	svc *service.Service
	log logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(svc *service.Service, log logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		svc: svc,
		log: log,
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
		res := &rest.SearchGroupResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	groups, total, err := h.svc.SearchGroups(ctx, req.Keyword, req.Page, req.PageSize)
	res := &rest.SearchGroupResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "搜索成功"
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
		h.log.Error(ctx, "Search groups failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// CreateGroup 创建群组
func (h *HTTPHandler) CreateGroup(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.CreateGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid create group request", logger.F("error", err.Error()))
		res := &rest.CreateGroupResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	group, err := h.svc.CreateGroup(ctx, req.Name, req.Description, req.Avatar, req.OwnerId, req.IsPublic, req.MaxMembers, req.MemberIds)
	res := &rest.CreateGroupResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "创建群组成功"
		}(),
		Group: func() *rest.GroupInfo {
			if group != nil {
				return &rest.GroupInfo{
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
			}
			return nil
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Create group failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// GetGroupInfo 获取群组信息
func (h *HTTPHandler) GetGroupInfo(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.GetGroupInfoRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid get group info request", logger.F("error", err.Error()))
		res := &rest.GetGroupInfoResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	group, members, err := h.svc.GetGroupInfo(ctx, req.GroupId, req.UserId)
	res := &rest.GetGroupInfoResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "获取群组信息成功"
		}(),
		Group: func() *rest.GroupInfo {
			if group != nil {
				return &rest.GroupInfo{
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
			}
			return nil
		}(),
		Members: func() []*rest.GroupMemberInfo {
			if err != nil {
				return []*rest.GroupMemberInfo{}
			}
			var pbMembers []*rest.GroupMemberInfo
			for _, member := range members {
				pbMembers = append(pbMembers, &rest.GroupMemberInfo{
					UserId:   member.UserID,
					GroupId:  member.GroupID,
					Role:     member.Role,
					Nickname: member.Nickname,
					JoinedAt: member.JoinedAt.Unix(),
				})
			}
			return pbMembers
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Get group info failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// DisbandGroup 解散群组
func (h *HTTPHandler) DisbandGroup(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.DisbandGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid disband group request", logger.F("error", err.Error()))
		res := &rest.DisbandGroupResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.DisbandGroup(ctx, req.GroupId, req.UserId)
	res := &rest.DisbandGroupResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "解散群组成功"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Disband group failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// JoinGroup 加入群组
func (h *HTTPHandler) JoinGroup(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.JoinGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid join group request", logger.F("error", err.Error()))
		res := &rest.JoinGroupResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.JoinGroup(ctx, req.GroupId, req.UserId, req.Reason)
	res := &rest.JoinGroupResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "加入群组成功"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Join group failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// LeaveGroup 退出群组
func (h *HTTPHandler) LeaveGroup(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.LeaveGroupRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid leave group request", logger.F("error", err.Error()))
		res := &rest.LeaveGroupResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.LeaveGroup(ctx, req.GroupId, req.UserId)
	res := &rest.LeaveGroupResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "退出群组成功"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Leave group failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}

// KickMember 踢出成员
func (h *HTTPHandler) KickMember(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.KickMemberRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid kick member request", logger.F("error", err.Error()))
		res := &rest.KickMemberResponse{
			Success: false,
			Message: "Invalid request format",
		}
		httpx.WriteObject(c, res, err)
		return
	}

	err := h.svc.KickMember(ctx, req.GroupId, req.OperatorId, req.TargetUserId)
	res := &rest.KickMemberResponse{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "踢出成员成功"
		}(),
	}
	if err != nil {
		h.log.Error(ctx, "Kick member failed", logger.F("error", err.Error()))
	}
	httpx.WriteObject(c, res, err)
}
