package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	"websocket-server/api/rest"
	"websocket-server/apps/connect-service/service"
	"websocket-server/pkg/logger"
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
		api.GET("/ws", h.WebSocketHandler)         // WebSocket长连接
		api.POST("/online_status", h.OnlineStatus) // 查询在线状态
	}
}

// WebSocketHandler 实现连接和心跳的长连接
func (h *Handler) WebSocketHandler(c *gin.Context) {
	// 从 header 获取 token
	token := c.GetHeader("Authorization")
	if token == "" {
		h.logger.Error(c.Request.Context(), "Missing authorization token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证 token"})
		return
	}

	// 从headers中获取userID
	userIDStr := c.GetHeader("User-ID")
	if userIDStr == "" {
		h.logger.Error(c.Request.Context(), "Missing User-ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少User-ID header"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.logger.Error(c.Request.Context(), "Invalid User-ID format", logger.F("userID", userIDStr), logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的User-ID格式"})
		return
	}

	// 验证token
	h.logger.Info(c.Request.Context(), "Validating token", logger.F("token", token), logger.F("userID", userID))
	if !h.service.ValidateToken(token) {
		h.logger.Error(c.Request.Context(), "Invalid token", logger.F("token", token), logger.F("userID", userID))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效认证 token"})
		return
	}
	h.logger.Info(c.Request.Context(), "Token validation successful", logger.F("userID", userID))

	// 升级到WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error(c.Request.Context(), "WebSocket upgrade failed", logger.F("error", err.Error()))
		return
	}
	defer conn.Close()

	// 将userID存储到gin.Context中，供后续使用
	c.Set("user_id", userID)

	// 1. 建立连接记录到Redis
	timestamp := time.Now().Unix()
	connID := fmt.Sprintf("conn-%d-%d", userID, timestamp)
	_, err = h.service.Connect(c.Request.Context(), userID, token, h.service.GetInstanceID(), "web")
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
		case 4: // 消息ACK确认
			if err := h.service.HandleMessageACK(c.Request.Context(), &wsMsg); err != nil {
				h.logger.Error(c.Request.Context(), "HandleMessageACK failed", logger.F("error", err.Error()))
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

// OnlineStatus 查询在线状态
func (h *Handler) OnlineStatus(c *gin.Context) {
	ctx := c.Request.Context()
	var req struct {
		UserIDs []int64 `json:"user_ids" binding:"required"`
	}
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

// gRPC服务端实现
type GRPCService struct {
	rest.UnimplementedConnectServiceServer
	handler *Handler
}

// NewGRPCService 创建gRPC服务
func (h *Handler) NewGRPCService() *GRPCService {
	return &GRPCService{handler: h}
}

// OnlineStatus 查询在线状态
func (g *GRPCService) OnlineStatus(ctx context.Context, req *rest.OnlineStatusRequest) (*rest.OnlineStatusResponse, error) {
	status, err := g.handler.service.OnlineStatus(ctx, req.UserIds)
	if err != nil {
		return &rest.OnlineStatusResponse{
			Status: make(map[int64]bool),
		}, err
	}
	return &rest.OnlineStatusResponse{
		Status: status,
	}, nil
}
