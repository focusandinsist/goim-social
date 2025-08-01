package handler

import (
	"goim-social/api/rest"
	"goim-social/apps/api-gateway-service/service"
	"goim-social/pkg/logger"
	"goim-social/pkg/utils"

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
	// // 网关管理API
	// gateway := r.Group("/api/v1/api-gateway")
	// {
	// 	gateway.POST("/online_status", h.OnlineStatus) // 查询在线状态(通过gRPC调用IM Gateway)
	// 	gateway.POST("/services", h.GetServices)       // 获取所有注册的服务
	// 	gateway.POST("/health", h.HealthCheck)         // 健康检查
	// }

	// 动态路由 [这是核心功能！]
	// 所有符合 /api/v1/{service-name}/* 格式的请求都会被动态路由
	api := r.Group("/api/v1")
	{
		api.Any("/*path", h.DynamicRoute) // 动态路由处理器
	}
}

// OnlineStatus 查询在线状态 - 通过gRPC调用IM Gateway
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

	// 通过gRPC调用IM Gateway服务
	status, err := h.svc.GetOnlineStatusFromIMGateway(ctx, req.UserIds)
	res := &rest.OnlineStatusResponse{
		Status: status,
	}
	if err != nil {
		h.log.Error(ctx, "Online status failed", logger.F("error", err.Error()))
	}
	utils.WriteObject(c, res, err)
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
