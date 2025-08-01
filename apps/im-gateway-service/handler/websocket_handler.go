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

	"goim-social/api/rest"
	"goim-social/apps/im-gateway-service/service"
	"goim-social/pkg/logger"
)

// WSHandler WebSocket协议处理器
type WSHandler struct {
	svc      *service.Service
	log      logger.Logger
	upgrader websocket.Upgrader
}

// NewWSHandler 创建WebSocket处理器
func NewWSHandler(svc *service.Service, log logger.Logger) *WSHandler {
	return &WSHandler{
		svc:      svc,
		log:      log,
		upgrader: websocket.Upgrader{},
	}
}

// RegisterRoutes 注册WebSocket路由
func (ws *WSHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/connect")
	{
		api.GET("/ws", ws.HandleConnection) // WebSocket长连接
	}
}

// HandleConnection 处理WebSocket连接
func (ws *WSHandler) HandleConnection(c *gin.Context) {
	// 从 header 获取 token
	token := c.GetHeader("Authorization")
	if token == "" {
		ws.log.Error(c.Request.Context(), "Missing authorization token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证 token"})
		return
	}

	// 从headers中获取userID
	userIDStr := c.GetHeader("User-ID")
	if userIDStr == "" {
		ws.log.Error(c.Request.Context(), "Missing User-ID header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少User-ID header"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ws.log.Error(c.Request.Context(), "Invalid User-ID format", logger.F("userID", userIDStr), logger.F("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的User-ID格式"})
		return
	}

	// 验证token
	ws.log.Info(c.Request.Context(), "Validating token", logger.F("token", token), logger.F("userID", userID))
	if !ws.svc.ValidateToken(token) {
		ws.log.Error(c.Request.Context(), "Invalid token", logger.F("token", token), logger.F("userID", userID))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效认证 token"})
		return
	}
	ws.log.Info(c.Request.Context(), "Token validation successful", logger.F("userID", userID))

	// 升级到WebSocket连接
	conn, err := ws.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		ws.log.Error(c.Request.Context(), "WebSocket upgrade failed", logger.F("error", err.Error()))
		return
	}
	defer conn.Close()

	// 将userID存储到gin.Context中，供后续使用
	c.Set("user_id", userID)

	// 1. 建立连接记录到Redis
	timestamp := time.Now().Unix()
	connID := fmt.Sprintf("conn-%d-%d", userID, timestamp)
	_, err = ws.svc.Connect(c.Request.Context(), userID, token, ws.svc.GetInstanceID(), "web")
	if err != nil {
		ws.log.Error(c.Request.Context(), "Failed to register connection", logger.F("error", err.Error()))
		return
	}

	// 2. 设置ping处理器
	conn.SetPingHandler(func(appData string) error {
		// 更新Redis中的心跳时间
		go func() {
			// 通过service更新心跳时间
			if err := ws.svc.Heartbeat(context.Background(), userID, connID); err != nil {
				ws.log.Error(context.Background(), "更新心跳时间失败",
					logger.F("userID", userID),
					logger.F("error", err.Error()))
			}
		}()

		// 发送pong响应
		err = conn.WriteMessage(websocket.PongMessage, []byte(appData))
		if err != nil {
			ws.log.Error(c.Request.Context(), "发送pong响应失败",
				logger.F("userID", userID),
				logger.F("error", err.Error()))
		}
		return err
	})

	// 3. 注册本地WebSocket连接
	ws.svc.AddWebSocketConnection(userID, conn)

	// 4. 确保断开时清理资源
	defer func(uid int64, cid string) {
		ws.svc.RemoveWebSocketConnection(uid)
		ws.svc.Disconnect(c.Request.Context(), uid, cid)
	}(userID, connID)

	// 认证通过，进入主循环
	ws.handleWebSocketMessages(c, conn, userID)
}

// handleWebSocketMessages 处理WebSocket消息循环
func (ws *WSHandler) handleWebSocketMessages(c *gin.Context, conn *websocket.Conn, userID int64) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			ws.log.Error(c.Request.Context(), "WebSocket read message failed", logger.F("error", err.Error()))
			ws.log.Info(c.Request.Context(), "WebSocket connection closed", logger.F("userID", userID))
			break
		}

		var wsMsg rest.WSMessage
		if err := proto.Unmarshal(msg, &wsMsg); err != nil {
			ws.log.Error(c.Request.Context(), "Invalid WebSocket proto message", logger.F("error", err.Error()))
			continue
		}

		ws.routeWebSocketMessage(c, conn, &wsMsg)
	}
}

// routeWebSocketMessage 路由WebSocket消息到对应的处理器
func (ws *WSHandler) routeWebSocketMessage(c *gin.Context, conn *websocket.Conn, wsMsg *rest.WSMessage) {
	switch wsMsg.MessageType {
	case 1: // 文本消息
		if err := ws.svc.ForwardMessageToLogicService(c.Request.Context(), wsMsg); err != nil {
			ws.log.Error(c.Request.Context(), "ForwardMessageToLogicService failed", logger.F("error", err.Error()))
		}
	case 2: // 心跳
		if err := ws.svc.HandleHeartbeat(c.Request.Context(), wsMsg, conn); err != nil {
			ws.log.Error(c.Request.Context(), "HandleHeartbeat failed", logger.F("error", err.Error()))
		}
	case 3: // 连接管理
		// 连接管理功能已简化，暂时跳过
		ws.log.Info(c.Request.Context(), "Connection management message received", logger.F("userID", wsMsg.From))
	case 4: // 消息ACK确认
		if err := ws.svc.HandleMessageACK(c.Request.Context(), wsMsg); err != nil {
			ws.log.Error(c.Request.Context(), "HandleMessageACK failed", logger.F("error", err.Error()))
		}
	case 10: // 在线状态事件推送
		// TODO:在线状态事件推送功能暂未实现(类似上线通知粉丝/订阅者)
		ws.log.Info(c.Request.Context(), "Online status event received", logger.F("userID", wsMsg.From))
	default:
		// 未知类型
		ws.log.Warn(c.Request.Context(), "Unknown message type", logger.F("type", wsMsg.MessageType))
	}
}
