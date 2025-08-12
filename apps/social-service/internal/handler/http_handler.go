package handler

import (
	"github.com/gin-gonic/gin"

	"goim-social/apps/social-service/internal/converter"
	"goim-social/apps/social-service/internal/service"
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
