package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	rest "goim-social/api/rest"
	"goim-social/apps/user-service/converter"
	"goim-social/apps/user-service/service"
	"goim-social/pkg/logger"
)

// HTTPHandler HTTP处理器
type HTTPHandler struct {
	service   *service.Service
	converter *converter.Converter
	logger    logger.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(service *service.Service, logger logger.Logger) *HTTPHandler {
	return &HTTPHandler{
		service:   service,
		converter: converter.NewConverter(),
		logger:    logger,
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
		resp := h.converter.BuildErrorRegisterResponse("Invalid request format")
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	h.logger.Info(ctx, "User registration attempt", logger.F("username", req.Username))

	user, err := h.service.Register(ctx, &req)
	if err != nil {
		h.logger.Error(ctx, "User registration failed", logger.F("username", req.Username), logger.F("error", err.Error()))
		resp := h.converter.BuildErrorRegisterResponse(err.Error())
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	h.logger.Info(ctx, "User registration successful", logger.F("user_id", user.User.Id), logger.F("username", user.User.Username))
	c.JSON(http.StatusOK, user)
}

// Login 用户登录
func (h *HTTPHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid login request", logger.F("error", err.Error()))
		resp := h.converter.BuildErrorLoginResponse("Invalid request format")
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	h.logger.Info(ctx, "User login attempt", logger.F("username", req.Username))

	response, err := h.service.Login(ctx, &req)
	if err != nil {
		h.logger.Error(ctx, "User login failed", logger.F("username", req.Username), logger.F("error", err.Error()))
		resp := h.converter.BuildErrorLoginResponse(err.Error())
		c.JSON(http.StatusUnauthorized, resp)
		return
	}

	h.logger.Info(ctx, "User login successful", logger.F("user_id", response.User.Id), logger.F("username", response.User.Username))
	c.JSON(http.StatusOK, response)
}

// GetUserByID 根据ID获取用户
func (h *HTTPHandler) GetUserByID(c *gin.Context) {
	ctx := c.Request.Context()

	var req rest.GetUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid get user request", logger.F("error", err.Error()))
		resp := h.converter.BuildErrorGetUserResponse("Invalid request format")
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	h.logger.Info(ctx, "Get user by ID", logger.F("user_id", req.UserId))

	// TODO: 实现实际的用户查询逻辑
	// user, err := h.service.GetUserByID(ctx, req.UserId)
	// if err != nil {
	//     h.logger.Error(ctx, "Get user failed", logger.F("user_id", req.UserId), logger.F("error", err.Error()))
	//     resp := h.converter.BuildErrorGetUserResponse(err.Error())
	//     c.JSON(http.StatusInternalServerError, resp)
	//     return
	// }

	// 临时返回成功响应
	resp := h.converter.BuildSuccessResponse("获取用户信息成功")
	c.JSON(http.StatusOK, resp)
}
