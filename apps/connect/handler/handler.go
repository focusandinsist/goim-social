package handler

import (
	"net/http"
	"websocket-server/api/rest"
	"websocket-server/apps/connect/model"
	"websocket-server/apps/connect/service"
	"websocket-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	service *service.Service
	logger  logger.Logger
}

func NewHandler(service *service.Service, logger logger.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/connect")
	{
		api.GET("/ws", h.WebSocketHandler)         // 只保留 connect 的长连接
		api.POST("/disconnect", h.Disconnect)      // 断开连接
		api.POST("/online_status", h.OnlineStatus) // 查询在线状态
	}
}

// WebSocketHandler 实现连接和心跳的长连接
func (h *Handler) WebSocketHandler(c *gin.Context) {
	// 从 header 获取 token
	token := c.GetHeader("Authorization")
	if token == "" || !h.service.ValidateToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少或无效认证 token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error(c.Request.Context(), "WebSocket upgrade failed", logger.F("error", err.Error()))
		return
	}
	defer conn.Close()

	// 认证通过，进入主循环
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			h.logger.Error(c.Request.Context(), "WebSocket read message failed", logger.F("error", err.Error()))
			h.Disconnect(c)
			h.logger.Info(c.Request.Context(), "WebSocket connection closed,", logger.F("userID", c.GetInt64("user_id")))
			break
		}
		var wsMsg rest.WSMessage
		if err := proto.Unmarshal(msg, &wsMsg); err != nil {
			h.logger.Error(c.Request.Context(), "Invalid WebSocket proto message", logger.F("error", err.Error()))
			continue
		}

		switch wsMsg.MessageType {
		case 1: // 文本消息
			if err := h.service.ForwardMessageToMessageService(c.Request.Context(), &wsMsg); err != nil {
				h.logger.Error(c.Request.Context(), "ForwardMessageToMessageService failed", logger.F("error", err.Error()))
			}
		case 2: // 心跳
			if err := h.service.HandleHeartbeat(c.Request.Context(), &wsMsg, conn); err != nil {
				h.logger.Error(c.Request.Context(), "HandleHeartbeat failed", logger.F("error", err.Error()))
			}
		case 3: // 连接管理
			if err := h.service.HandleConnectionManage(c.Request.Context(), &wsMsg, conn); err != nil {
				h.logger.Error(c.Request.Context(), "HandleConnectionManage failed", logger.F("error", err.Error()))
			}
		case 10: // 在线状态事件推送
			if err := h.service.HandleOnlineStatusEvent(c.Request.Context(), &wsMsg, conn); err != nil {
				h.logger.Error(c.Request.Context(), "HandleOnlineStatusEvent failed", logger.F("error", err.Error()))
			}
		default:
			// 未知类型，可记录日志或忽略
			h.logger.Warn(c.Request.Context(), "Unknown message type", logger.F("type", wsMsg.MessageType))
		}
	}
}

func (h *Handler) Disconnect(c *gin.Context) {
	ctx := c.Request.Context()
	var req model.DisconnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid disconnect request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Disconnect(ctx, req.UserID, req.ConnID); err != nil {
		h.logger.Error(ctx, "Disconnect failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "断开成功"})
}

func (h *Handler) OnlineStatus(c *gin.Context) {
	ctx := c.Request.Context()
	var req model.OnlineStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error(ctx, "Invalid online status request", logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, err := h.service.OnlineStatus(ctx, req.UserIDs)
	if err != nil {
		h.logger.Error(ctx, "Online status failed", logger.F("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}
