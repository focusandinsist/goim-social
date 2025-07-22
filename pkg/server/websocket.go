package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	kratoslog "github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
)

// WebSocketServer WebSocket服务器接口
type WebSocketServer interface {
	RegisterHandler(path string, handler WebSocketHandler)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// WebSocketHandler WebSocket处理器接口
type WebSocketHandler interface {
	HandleConnection(conn *websocket.Conn, r *http.Request)
}

// WebSocketHandlerFunc WebSocket处理器函数类型
type WebSocketHandlerFunc func(conn *websocket.Conn, r *http.Request)

// HandleConnection WebSocketHandler接口实现
func (f WebSocketHandlerFunc) HandleConnection(conn *websocket.Conn, r *http.Request) {
	f(conn, r)
}

// WebSocketServerWrapper WebSocket服务器包装器
type WebSocketServerWrapper struct {
	engine   *gin.Engine
	upgrader websocket.Upgrader
	handlers map[string]WebSocketHandler
	logger   kratoslog.Logger
	mu       sync.RWMutex
}

// NewWebSocketServerWrapper 创建WebSocket服务器包装器
func NewWebSocketServerWrapper(engine *gin.Engine, logger kratoslog.Logger) *WebSocketServerWrapper {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	return &WebSocketServerWrapper{
		engine:   engine,
		upgrader: upgrader,
		handlers: make(map[string]WebSocketHandler),
		logger:   logger,
	}
}

// RegisterHandler 注册WebSocket处理器
func (ws *WebSocketServerWrapper) RegisterHandler(path string, handler WebSocketHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.handlers[path] = handler

	// 在Gin引擎上注册路由
	ws.engine.GET(path, func(c *gin.Context) {
		ws.handleWebSocket(c, handler)
	})
}

// handleWebSocket 处理WebSocket连接
func (ws *WebSocketServerWrapper) handleWebSocket(c *gin.Context, handler WebSocketHandler) {
	conn, err := ws.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		ws.logger.Log(kratoslog.LevelError, "msg", "WebSocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	// 调用处理器
	handler.HandleConnection(conn, c.Request)
}

// Start WebSocket服务器启动（实际上是依赖HTTP服务器，先留在这打个日志）
func (ws *WebSocketServerWrapper) Start(ctx context.Context) error {
	ws.logger.Log(kratoslog.LevelInfo, "msg", "WebSocket server ready")
	return nil
}

// Stop WebSocket服务器停止
func (ws *WebSocketServerWrapper) Stop(ctx context.Context) error {
	ws.logger.Log(kratoslog.LevelInfo, "msg", "WebSocket server stopping")
	return nil
}
