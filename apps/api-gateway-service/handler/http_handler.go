package handler

import (
	"websocket-server/api/rest"
	"websocket-server/apps/api-gateway-service/service"
	"websocket-server/pkg/logger"
	"websocket-server/pkg/utils"

	"github.com/gin-gonic/gin"
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
	// 网关管理API
	gateway := r.Group("/api/v1/gateway")
	{
		gateway.POST("/online_status", h.OnlineStatus)       // 查询在线状态
		gateway.POST("/online_count", h.OnlineCount)         // 获取在线用户数量
		gateway.POST("/online_users", h.OnlineUsers)         // 获取在线用户列表
		gateway.POST("/user_connections", h.UserConnections) // 获取用户连接信息
		gateway.POST("/services", h.GetServices)             // 获取所有注册的服务
		gateway.POST("/health", h.HealthCheck)               // 健康检查
	}

	// 动态路由 - 这是核心功能！
	// 所有符合 /api/v1/{service-name}/* 格式的请求都会被动态路由
	api := r.Group("/api/v1")
	{
		api.Any("/*path", h.DynamicRoute) // 动态路由处理器
	}
}

// OnlineStatus 查询在线状态
func (h *HTTPHandler) OnlineStatus(c *gin.Context) {
	ctx := c.Request.Context()
	var req rest.OnlineStatusRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error(ctx, "Invalid online status request", logger.F("error", err.Error()))
		res := &rest.OnlineStatusResponse{
			Status: make(map[int64]bool),
		}
		utils.WriteObject(c, res, err)
		return
	}

	status, err := h.svc.OnlineStatus(ctx, req.UserIds)
	res := &rest.OnlineStatusResponse{
		Status: status,
	}
	if err != nil {
		h.log.Error(ctx, "Online status failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
}

// OnlineCount 获取在线用户数量
func (h *HTTPHandler) OnlineCount(c *gin.Context) {
	ctx := c.Request.Context()

	count, err := h.svc.GetOnlineUserCount(ctx)

	// 创建响应结构（这里简化处理，实际应该定义专门的protobuf消息）
	res := map[string]interface{}{
		"count": count,
	}

	if err != nil {
		h.log.Error(ctx, "Get online count failed", logger.F("error", err.Error()))
		res["error"] = err.Error()
	}

	c.JSON(200, res) // 临时使用JSON，后续可以改为protobuf
}

// DynamicRoute 动态路由处理器 - 核心功能！
func (h *HTTPHandler) DynamicRoute(c *gin.Context) {
	ctx := c.Request.Context()

	// 记录请求日志
	h.log.Info(ctx, "Dynamic route request",
		logger.F("method", c.Request.Method),
		logger.F("path", c.Request.URL.Path),
		logger.F("query", c.Request.URL.RawQuery))

	// 调用service层的动态路由功能
	err := h.svc.ProxyRequest(c.Writer, c.Request)
	if err != nil {
		h.log.Error(ctx, "Dynamic route failed", logger.F("error", err.Error()))
		// 错误已经在service层处理了，这里不需要再次响应
	}
}

// GetServices 获取所有注册的服务
func (h *HTTPHandler) GetServices(c *gin.Context) {
	ctx := c.Request.Context()

	services := h.svc.GetAllServices()

	res := map[string]interface{}{
		"services": services,
		"count":    len(services),
	}

	h.log.Info(ctx, "Get services request", logger.F("count", len(services)))
	c.JSON(200, res)
}

// HealthCheck 健康检查
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	res := map[string]interface{}{
		"status":    "healthy",
		"timestamp": "2024-01-01T00:00:00Z", // 简化处理
		"version":   "1.0.0",
	}
	c.JSON(200, res)
}

// OnlineUsers 获取在线用户列表
func (h *HTTPHandler) OnlineUsers(c *gin.Context) {
	ctx := c.Request.Context()

	users, err := h.svc.GetAllOnlineUsers(ctx)

	// 创建响应结构（这里简化处理，实际应该定义专门的protobuf消息）
	res := map[string]interface{}{
		"users": users,
	}

	if err != nil {
		h.log.Error(ctx, "Get online users failed", logger.F("error", err.Error()))
		res["error"] = err.Error()
	}

	c.JSON(200, res)
}

// UserConnections 获取用户连接信息
func (h *HTTPHandler) UserConnections(c *gin.Context) {
	ctx := c.Request.Context()

	// 从请求体获取user_id
	var req struct {
		UserID int64 `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Error(ctx, "Invalid user connections request", logger.F("error", err.Error()))
		c.JSON(400, map[string]interface{}{
			"error": "Invalid request format",
		})
		return
	}

	connections, err := h.svc.GetUserConnections(ctx, req.UserID)

	// 创建响应结构（这里简化处理，实际应该定义专门的protobuf消息）
	res := map[string]interface{}{
		"user_id":     req.UserID,
		"connections": connections,
	}

	if err != nil {
		h.log.Error(ctx, "Get user connections failed", logger.F("error", err.Error()))
		res["error"] = err.Error()
	}

	c.JSON(200, res)
}
