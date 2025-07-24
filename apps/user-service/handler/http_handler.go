package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	rest "websocket-server/api/rest"
	"websocket-server/apps/user-service/service"
	"websocket-server/pkg/logger"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	service *service.Service
	logger  logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(service *service.Service, logger logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/users")
	{
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)
		api.POST("/get", h.GetUserByID)
	}
}

// Register 用户注册
func (h *HTTPHandler) Register(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid register request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "User registration attempt", logger.F("username", req.Username))

	user, err := h.service.Register(ctx, &req)
	if err != nil {
		h.logger.Error(ctx, "User registration failed", logger.F("username", req.Username), logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "User registration successful", logger.F("user_id", user.User.Id), logger.F("username", user.User.Username))
	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
		"data":    user,
	})
}

// Login 用户登录
func (h *HTTPHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid login request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "User login attempt", logger.F("username", req.Username))

	response, err := h.service.Login(ctx, &req)
	if err != nil {
		h.logger.Error(ctx, "User login failed", logger.F("username", req.Username), logger.F("error", err.Error()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "User login successful", logger.F("user_id", response.User.Id), logger.F("username", response.User.Username))
	c.JSON(http.StatusOK, gin.H{
		"message": "登录成功",
		"data":    response,
	})
}

// GetUserByID 根据ID获取用户
func (h *HTTPHandler) GetUserByID(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info(ctx, "Get user by ID", logger.F("user_id", req.UserID))

	// 这里简化处理，实际应该解析userID为int64
	// userIDInt, err := strconv.ParseInt(req.UserID, 10, 64)
	// if err != nil {
	//     c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
	//     return
	// }

	// 临时返回模拟数据
	c.JSON(http.StatusOK, gin.H{
		"message": "获取用户信息成功",
		"data": gin.H{
			"id":       req.UserID,
			"username": "test_user",
			"email":    "test@example.com",
			"nickname": "测试用户",
		},
	})
}
