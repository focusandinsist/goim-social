package handler

import (
	"github.com/gin-gonic/gin"

	"websocket-server/apps/friend-service/service"
	"websocket-server/pkg/logger"
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
	api := r.Group("/api/v1/friend")
	{
		api.POST("/list", h.ListFriends)                 // 查询好友列表
		api.POST("/profile", h.GetFriendProfile)         // 获取单个好友个人简介
		api.POST("/update_remark", h.UpdateFriendRemark) // 更新好友备注
		api.POST("/apply_list", h.ListFriendApply)       // 查询好友申请列表
		api.POST("/apply", h.ApplyFriend)                // 申请加好友
		api.POST("/respond_apply", h.RespondFriendApply) // 回应好友申请
		api.POST("/delete", h.DeleteFriend)              // 删除好友
	}
}
