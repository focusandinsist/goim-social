package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
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

	// 从headers中获取userID
	userIDStr := c.GetHeader("User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少User-ID header"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的User-ID格式"})
		return
	}

	// 将userID存储到gin.Context中，供后续使用
	c.Set("user_id", userID)

	// 1. 建立连接记录到Redis
	timestamp := time.Now().Unix()
	connID := fmt.Sprintf("conn-%d-%d", userID, timestamp)
	_, err = h.service.Connect(c.Request.Context(), userID, token, "connect-server-1", "web")
	if err != nil {
		h.logger.Error(c.Request.Context(), "Failed to register connection", logger.F("error", err.Error()))
		return
	}

	// 2. 设置ping处理器 - 自动响应客户端的ping消息
	conn.SetPingHandler(func(appData string) error {
		// 更新Redis中的心跳时间
		go func() {
			timestamp := time.Now().Unix()
			// 通过service更新心跳时间
			if err := h.service.UpdateHeartbeat(context.Background(), userID, connID, timestamp); err != nil {
				h.logger.Error(context.Background(), "更新心跳时间失败",
					logger.F("userID", userID),
					logger.F("error", err.Error()))
			}
		}()

		// 发送pong响应
		err = conn.WriteMessage(websocket.PongMessage, []byte(appData))
		if err != nil {
			h.logger.Error(c.Request.Context(), "发送pong响应失败",
				logger.F("userID", userID),
				logger.F("error", err.Error()))
		}
		return err
	})

	// 3. 注册本地WebSocket连接
	h.service.AddWebSocketConnection(userID, conn)

	// 4. 确保断开时清理资源
	defer func(uid int64, cid string) {
		h.service.RemoveWebSocketConnection(uid)
		h.service.Disconnect(c.Request.Context(), uid, cid)
	}(userID, connID)

	// 认证通过，进入主循环
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			h.logger.Error(c.Request.Context(), "WebSocket read message failed", logger.F("error", err.Error()))
			h.logger.Info(c.Request.Context(), "WebSocket connection closed,", logger.F("userID", userID))
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
