package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	"websocket-server/api/rest"
)

// 全局变量：已收到的消息集合（用于去重）
var receivedMessages = make(map[int64]bool)
var receivedMessagesMutex sync.Mutex

// isMessageDuplicate 检查消息是否重复
func isMessageDuplicate(messageID int64) bool {
	receivedMessagesMutex.Lock()
	defer receivedMessagesMutex.Unlock()

	if receivedMessages[messageID] {
		return true
	}

	receivedMessages[messageID] = true
	return false
}

// 用户信息结构
type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
	DeviceID string `json:"device_id"`
}

// API响应结构
type APIResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

// 登录响应数据结构
type LoginData struct {
	Token    string   `json:"token"`
	User     UserInfo `json:"user"`
	ExpireAt int64    `json:"expire_at"`
	DeviceID string   `json:"device_id"`
}

func main() {
	// 命令行参数
	var (
		userID     = flag.Int64("user", 1001, "调试模式下的用户ID")
		targetID   = flag.Int64("target", 1002, "目标用户ID")
		wsURL      = flag.String("wsurl", "ws://localhost:21005/api/v1/connect/ws", "WebSocket服务地址")
		userAPIURL = flag.String("userapi", "http://localhost:21001/api/v1/users", "用户服务API地址")
		autoMode   = flag.Bool("auto", false, "自动模式，自动发送消息")
		skipAuth   = flag.Bool("skip", false, "跳过认证，使用调试token")
	)
	flag.Parse()

	var userInfo *UserInfo

	if *skipAuth {
		// 跳过认证模式，使用调试token
		fmt.Println("🔧 调试模式：跳过认证流程")
		userInfo = &UserInfo{
			ID:       *userID,
			Username: fmt.Sprintf("debug_user_%d", *userID),
			Token:    "auth-debug",
			DeviceID: fmt.Sprintf("debug-device-%d", *userID),
		}
	} else {
		// 正常认证流程
		userInfo = authenticateUser(*userAPIURL)
		if userInfo == nil {
			log.Fatal("认证失败，程序退出")
		}
	}

	fmt.Printf("✅ 认证成功 - 用户: %s (ID: %d)\n", userInfo.Username, userInfo.ID)
	fmt.Printf("🎯 目标用户ID: %d\n", *targetID)

	// 建立WebSocket连接
	conn := connectWebSocket(*wsURL, userInfo)
	defer conn.Close()

	fmt.Println("\n📱 IM客户端已启动！")
	fmt.Println("💬 输入消息内容，按回车发送")
	fmt.Println("🚪 输入 'exit' 退出程序")
	fmt.Println("📋 输入 'help' 查看更多命令")
	fmt.Println(strings.Repeat("-", 50))

	// 获取未读消息
	go fetchUnreadMessages(userInfo.ID)

	// 启动消息接收协程
	go receiveMessages(conn, userInfo.ID)

	// 启动心跳协程
	go startHeartbeat(conn, userInfo.ID)

	if *autoMode {
		// 自动模式
		go autoSendMessages(conn, userInfo.ID, *targetID)
	}

	// 主循环处理用户输入
	handleUserInput(conn, userInfo.ID, *targetID)
}

// 用户认证流程
func authenticateUser(apiURL string) *UserInfo {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("🔐 用户认证")
	fmt.Println("1. 登录")
	fmt.Println("2. 注册")
	fmt.Print("请选择 (1/2): ")

	scanner.Scan()
	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "1":
		return loginUser(apiURL, scanner)
	case "2":
		return registerUser(apiURL, scanner)
	default:
		fmt.Println("❌ 无效选择")
		return authenticateUser(apiURL)
	}
}

// 用户登录
func loginUser(apiURL string, scanner *bufio.Scanner) *UserInfo {
	fmt.Println("\n📝 用户登录")

	fmt.Print("用户名: ")
	scanner.Scan()
	username := strings.TrimSpace(scanner.Text())

	fmt.Print("密码: ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())

	deviceID := fmt.Sprintf("client-%d", time.Now().Unix())

	// 构造登录请求
	loginReq := map[string]string{
		"username":  username,
		"password":  password,
		"device_id": deviceID,
	}

	reqBody, _ := json.Marshal(loginReq)

	// 发送登录请求
	resp, err := http.Post(apiURL+"/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("❌ 登录请求失败: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		fmt.Printf("❌ 解析响应失败: %v\n", err)
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ 登录失败: %s\n", apiResp.Error)
		return nil
	}

	// 解析登录数据
	dataBytes, _ := json.Marshal(apiResp.Data)
	var loginData LoginData
	if err := json.Unmarshal(dataBytes, &loginData); err != nil {
		fmt.Printf("❌ 解析登录数据失败: %v\n", err)
		return nil
	}

	return &UserInfo{
		ID:       loginData.User.ID,
		Username: loginData.User.Username,
		Token:    loginData.Token,
		DeviceID: loginData.DeviceID,
	}
}

// 未读消息请求结构
type UnreadRequest struct {
	UserID int64 `json:"user_id"`
}

// 未读消息响应结构
type UnreadResponse struct {
	Messages []UnreadMessage `json:"messages"`
	Total    int             `json:"total"`
}

type UnreadMessage struct {
	ID        string `json:"id"`
	From      int64  `json:"from"`
	To        int64  `json:"to"`
	GroupID   int64  `json:"group_id"`
	Content   string `json:"content"`
	MsgType   int32  `json:"msg_type"`
	AckID     string `json:"ack_id"`
	Status    int32  `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// fetchUnreadMessages 通过HTTP POST接口获取未读消息
func fetchUnreadMessages(userID int64) {
	fmt.Printf("\n� 正在获取未读消息...\n")

	// 构造POST请求体
	reqBody := UnreadRequest{
		UserID: userID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("❌ 构造请求失败: %v\n", err)
		return
	}

	// 发送POST请求
	url := "http://localhost:21004/api/v1/messages/unread"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ 获取未读消息失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ 未读消息请求失败，状态码: %d\n", resp.StatusCode)
		return
	}

	var unreadResp UnreadResponse
	if err := json.NewDecoder(resp.Body).Decode(&unreadResp); err != nil {
		fmt.Printf("❌ 解析未读消息失败: %v\n", err)
		return
	}

	if len(unreadResp.Messages) == 0 {
		fmt.Printf("📭 没有未读消息\n")
		return
	}

	fmt.Printf("� 收到 %d 条未读消息:\n", len(unreadResp.Messages))
	for _, msg := range unreadResp.Messages {
		// 解析时间
		createdAt, _ := time.Parse(time.RFC3339, msg.CreatedAt)
		timestamp := createdAt.Format("2006-01-02 15:04:05")

		// 显示消息
		fmt.Printf("[%s] � [未读消息] 来自用户%d: %s\n", timestamp, msg.From, msg.Content)
	}

	// 标记消息为已读
	go markMessagesAsRead(userID, unreadResp.Messages)

	fmt.Printf("[用户%d] 💬 ", userID)
}

// markMessagesAsRead 标记消息为已读
func markMessagesAsRead(userID int64, messages []UnreadMessage) {
	if len(messages) == 0 {
		return
	}

	// 提取消息ID
	var messageIDs []string
	for _, msg := range messages {
		messageIDs = append(messageIDs, msg.ID)
	}

	// 构造标记已读请求
	reqBody := map[string]interface{}{
		"user_id":     userID,
		"message_ids": messageIDs,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("❌ 构造标记已读请求失败: %v\n", err)
		return
	}

	// 发送POST请求标记已读
	url := "http://localhost:21004/api/v1/messages/mark-read"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ 标记消息已读失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✅ 已标记 %d 条消息为已读\n", len(messageIDs))
	}
}

// 用户注册
func registerUser(apiURL string, scanner *bufio.Scanner) *UserInfo {
	fmt.Println("\n📝 用户注册")

	fmt.Print("用户名: ")
	scanner.Scan()
	username := strings.TrimSpace(scanner.Text())

	fmt.Print("密码: ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())

	fmt.Print("邮箱: ")
	scanner.Scan()
	email := strings.TrimSpace(scanner.Text())

	fmt.Print("昵称: ")
	scanner.Scan()
	nickname := strings.TrimSpace(scanner.Text())

	// 构造注册请求
	registerReq := map[string]string{
		"username": username,
		"password": password,
		"email":    email,
		"nickname": nickname,
	}

	reqBody, _ := json.Marshal(registerReq)

	// 发送注册请求
	resp, err := http.Post(apiURL+"/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("❌ 注册请求失败: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		fmt.Printf("❌ 解析响应失败: %v\n", err)
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ 注册失败: %s\n", apiResp.Error)
		return nil
	}

	fmt.Println("✅ 注册成功！请使用新账号登录")
	return loginUser(apiURL, scanner)
}

// 建立WebSocket连接
func connectWebSocket(wsURL string, userInfo *UserInfo) *websocket.Conn {
	// 设置连接头
	headers := make(map[string][]string)
	headers["Authorization"] = []string{userInfo.Token}
	headers["User-ID"] = []string{fmt.Sprintf("%d", userInfo.ID)}

	fmt.Printf("🔌 正在连接WebSocket服务器: %s\n", wsURL)

	// 连接WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		log.Fatalf("❌ WebSocket连接失败: %v", err)
	}

	fmt.Println("✅ WebSocket连接成功")
	return conn
}

// 处理用户输入
func handleUserInput(conn *websocket.Conn, userID, targetID int64) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("\n[用户%d] 💬 ", userID)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		// 处理特殊命令
		switch input {
		case "exit", "quit", "q":
			fmt.Println("👋 再见！")
			return
		case "help", "h":
			showHelp()
			continue
		case "target":
			targetID = changeTarget(scanner, targetID)
			continue
		}

		// 检查是否是切换目标用户的命令
		if strings.HasPrefix(input, "/to ") {
			parts := strings.Split(input, " ")
			if len(parts) >= 2 {
				if newTarget, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					targetID = newTarget
					fmt.Printf("🎯 目标用户已切换为: %d\n", targetID)
					continue
				}
			}
		}

		// 发送消息
		sendMessage(conn, userID, targetID, input)
	}
}

// sendMessageACK 发送消息ACK确认
func sendMessageACK(conn *websocket.Conn, userID, messageID int64) {
	// 构造ACK消息
	ackMsg := &rest.WSMessage{
		MessageId:   messageID,
		From:        userID,
		To:          0, // ACK消息不需要To字段
		GroupId:     0,
		Content:     "",
		Timestamp:   time.Now().Unix(),
		MessageType: 4,  // 4表示ACK消息
		AckId:       "", // AckID已简化，不再需要
	}

	// 序列化消息
	msgBytes, err := proto.Marshal(ackMsg)
	if err != nil {
		log.Printf("❌ 序列化ACK消息失败: %v", err)
		return
	}

	// 发送ACK消息
	if err := conn.WriteMessage(websocket.BinaryMessage, msgBytes); err != nil {
		log.Printf("❌ 发送ACK消息失败: %v", err)
	} else {
		log.Printf("✅ 已发送ACK: MessageID=%d, UserID=%d", messageID, userID)
	}
}

// 显示帮助信息
func showHelp() {
	fmt.Println("\n📋 可用命令:")
	fmt.Println("  exit/quit/q    - 退出程序")
	fmt.Println("  help/h         - 显示帮助")
	fmt.Println("  target         - 更改目标用户")
	fmt.Println("  /to <用户ID>   - 快速切换目标用户")
	fmt.Println("  其他输入       - 发送消息")
}

// 更改目标用户
func changeTarget(scanner *bufio.Scanner, currentTarget int64) int64 {
	fmt.Printf("当前目标用户: %d\n", currentTarget)
	fmt.Print("请输入新的目标用户ID: ")

	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if newTarget, err := strconv.ParseInt(input, 10, 64); err == nil {
			fmt.Printf("🎯 目标用户已更改为: %d\n", newTarget)
			return newTarget
		} else {
			fmt.Println("❌ 无效的用户ID")
		}
	}

	return currentTarget
}

// 发送消息
func sendMessage(conn *websocket.Conn, from, to int64, content string) {
	msg := &rest.WSMessage{
		MessageType: 1,
		From:        from,
		To:          to,
		Content:     content,
		Timestamp:   time.Now().Unix(),
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("❌ 消息序列化失败: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		log.Printf("❌ 发送消息失败: %v", err)
		return
	}

	fmt.Printf("📤 [发送给%d]: %s\n", to, content)
}

// 心跳协程 - 定时发送ping消息保持连接活跃
func startHeartbeat(c *websocket.Conn, userID int64) {
	ticker := time.NewTicker(20 * time.Second) // 每20秒发送一次ping
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 发送ping消息
			if err := c.WriteMessage(websocket.PingMessage, []byte("heartbeat")); err != nil {
				log.Printf("❌ 用户 %d 发送ping失败: %v", userID, err)
				return
			}
		}
	}
}

// 接收消息的协程
func receiveMessages(c *websocket.Conn, userID int64) {
	// 设置ping处理器
	c.SetPingHandler(func(appData string) error {
		return c.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	// 设置pong处理器 - 静默处理，不记录日志
	c.SetPongHandler(func(appData string) error {
		return nil
	})

	for {
		// 移除读取超时，依赖ping/pong机制检测连接状态
		messageType, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("❌ 用户 %d 连接被关闭: %v", userID, err)
			} else {
				log.Printf("❌ 用户 %d 读取消息失败: %v", userID, err)
			}
			return
		}

		// 处理不同类型的消息
		switch messageType {
		case websocket.PingMessage:
			c.WriteMessage(websocket.PongMessage, message)
			continue
		case websocket.PongMessage:
			continue
		case websocket.BinaryMessage:
			// 处理业务消息
		default:
			log.Printf("⚠️ 用户 %d 收到未知类型消息: %d", userID, messageType)
			continue
		}

		// 解析业务消息
		var wsMsg rest.WSMessage
		if err := proto.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("❌ 用户 %d 解析消息失败: %v", userID, err)
			continue
		}

		// 显示所有相关消息（发给当前用户的或当前用户发送的）
		if wsMsg.To == userID || wsMsg.From == userID {
			// 消息去重检查
			if isMessageDuplicate(wsMsg.MessageId) {
				log.Printf("🔄 重复消息，忽略: MessageID=%d", wsMsg.MessageId)
				continue
			}

			timestamp := time.Unix(wsMsg.Timestamp, 0).Format("2006-01-02 15:04:05")

			// 判断是否是历史消息（根据时间戳判断，如果是5分钟前的消息就认为是历史消息）
			isHistoryMessage := time.Now().Unix()-wsMsg.Timestamp > 300 // 5分钟前的消息认为是历史消息

			var direction string
			if wsMsg.To == userID {
				// 收到的消息
				if isHistoryMessage {
					direction = fmt.Sprintf("📜 [历史消息] 来自用户%d", wsMsg.From)
				} else {
					direction = fmt.Sprintf("📥 来自用户%d", wsMsg.From)
					// 收到新消息时，发送ACK确认已读
					sendMessageACK(c, userID, wsMsg.MessageId)
				}
			} else {
				// 发送的消息
				if isHistoryMessage {
					direction = fmt.Sprintf("📜 [历史消息] 发送给用户%d", wsMsg.To)
				} else {
					direction = fmt.Sprintf("📤 发送给用户%d", wsMsg.To)
				}
			}

			fmt.Printf("\n[%s] %s: %s\n", timestamp, direction, wsMsg.Content)
			fmt.Printf("[用户%d] 💬 ", userID)
		}
	}
}

// 自动发送消息的协程
func autoSendMessages(c *websocket.Conn, userID, targetID int64) {
	counter := 1
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		content := fmt.Sprintf("🤖 自动消息 #%d", counter)
		sendMessage(c, userID, targetID, content)
		counter++
	}
}
