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
	"time"

	"websocket-server/api/rest"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

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

	// 启动消息接收协程
	go receiveMessages(conn, userInfo.ID)

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

// 接收消息的协程
func receiveMessages(c *websocket.Conn, userID int64) {
	for {
		c.SetReadDeadline(time.Now().Add(30 * time.Second))
		_, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
				log.Printf("❌ 连接被关闭: %v", err)
			}
			return
		}

		// 解析消息
		var wsMsg rest.WSMessage
		if err := proto.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("❌ 解析消息失败: %v", err)
			continue
		}

		// 只显示发给当前用户的消息
		if wsMsg.To == userID {
			timestamp := time.Unix(wsMsg.Timestamp, 0).Format("15:04:05")
			fmt.Printf("\n📥 [%s] 来自用户%d: %s\n", timestamp, wsMsg.From, wsMsg.Content)
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
