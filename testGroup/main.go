package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"os/signal"
	"runtime"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	"goim-social/api/rest"
)

// GroupMember 群成员信息
type GroupMember struct {
	UserID   int64  `json:"user_id"`
	GroupID  int64  `json:"group_id"`
	Role     string `json:"role"`
	Nickname string `json:"nickname"`
	JoinedAt int64  `json:"joined_at"`
	Online   bool   `json:"online"`
}

// GroupInfo 群组信息
type GroupInfo struct {
	ID          int64         `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	MemberCount int32         `json:"member_count"`
	Members     []GroupMember `json:"members"`
}

// ChatWindow 聊天窗口
type ChatWindow struct {
	Member   GroupMember
	Messages []ChatMessage
	mu       sync.Mutex
}

// ChatMessage 聊天消息
type ChatMessage struct {
	From      int64     `json:"from"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Nickname  string    `json:"nickname"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	UserID  int64  `json:"user_id"`
	GroupID int64  `json:"group_id"`
	Content string `json:"content"`
}

// GroupChatClient 群聊客户端
type GroupChatClient struct {
	groupID         int64
	groupInfo       GroupInfo
	chatWindows     map[int64]*ChatWindow
	connections     map[int64]*websocket.Conn
	defaultSender   int64          // 默认发送者ID
	processedMsgIDs map[int64]bool // 已处理的消息ID，用于去重
	mu              sync.RWMutex
}

// 全局客户端实例，用于HTTP处理器访问
var globalClient *GroupChatClient

// handleSendMessage 处理发送消息的HTTP请求
func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if globalClient == nil {
		http.Error(w, "Client not initialized", http.StatusInternalServerError)
		return
	}

	// 发送消息
	if err := globalClient.sendMessage(req.UserID, req.Content); err != nil {
		fmt.Printf("Failed to send message via HTTP: %v\n", err)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Message sent via HTTP: UserID=%d, Content=%s\n", req.UserID, req.Content)

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// startHTTPServer 启动HTTP服务器
func startHTTPServer() {
	http.HandleFunc("/send-message", handleSendMessage)

	fmt.Println("Starting HTTP server on :8080 for message handling...")
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("HTTP server failed: %v", err)
		}
	}()
}

func main() {
	fmt.Println("=== Multi-Window Group Chat Client ===")
	fmt.Println("Using auth-debug token for debugging")
	fmt.Println()

	client := &GroupChatClient{
		chatWindows:     make(map[int64]*ChatWindow),
		connections:     make(map[int64]*websocket.Conn),
		processedMsgIDs: make(map[int64]bool),
	}

	// 设置全局客户端实例
	globalClient = client

	// 启动HTTP服务器
	startHTTPServer()

	// Get group ID
	fmt.Print("Enter Group ID: ")
	groupIDStr, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	groupIDStr = strings.TrimSpace(groupIDStr)
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Fatal("Invalid Group ID:", err)
	}
	client.groupID = groupID

	// Fetch group info and members
	fmt.Println("Fetching group information...")
	if err := client.fetchGroupInfo(); err != nil {
		log.Fatal("Failed to fetch group info:", err)
	}

	fmt.Printf("Group: %s (ID: %d)\n", client.groupInfo.Name, client.groupInfo.ID)
	fmt.Printf("Members: %d\n", len(client.groupInfo.Members))

	// Check online status
	fmt.Println("Checking online status...")
	if err := client.checkOnlineStatus(); err != nil {
		log.Printf("Warning: Failed to check online status: %v", err)
	}

	// Display members
	client.displayMembers()

	// Set default sender (first member or owner)
	if len(client.groupInfo.Members) > 0 {
		// Try to find owner first
		for _, member := range client.groupInfo.Members {
			if member.Role == "owner" {
				client.defaultSender = member.UserID
				break
			}
		}
		// If no owner found, use first member
		if client.defaultSender == 0 {
			client.defaultSender = client.groupInfo.Members[0].UserID
		}

		defaultSenderName := "Unknown"
		for _, member := range client.groupInfo.Members {
			if member.UserID == client.defaultSender {
				defaultSenderName = member.Nickname
				break
			}
		}
		fmt.Printf("Default sender set to: %s (ID: %d)\n", defaultSenderName, client.defaultSender)
	}

	// Limit to 5 windows max
	maxWindows := 5
	if len(client.groupInfo.Members) > maxWindows {
		fmt.Printf("Limiting to %d chat windows (group has %d members)\n", maxWindows, len(client.groupInfo.Members))
		client.groupInfo.Members = client.groupInfo.Members[:maxWindows]
	}

	// Create chat windows for each member
	for _, member := range client.groupInfo.Members {
		client.chatWindows[member.UserID] = &ChatWindow{
			Member:   member,
			Messages: make([]ChatMessage, 0),
		}
	}

	// Connect WebSocket for each member
	fmt.Println("Connecting WebSocket for each member...")
	for _, member := range client.groupInfo.Members {
		if err := client.connectMember(member); err != nil {
			log.Printf("Failed to connect member %s (ID: %d): %v", member.Nickname, member.UserID, err)
			continue
		}
		fmt.Printf("Connected: %s (ID: %d)\n", member.Nickname, member.UserID)
	}

	// Open chat windows
	fmt.Println("Opening chat windows...")
	client.openChatWindows()

	// 设置信号处理，优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 使用 select{} 来永久阻塞主程序，保持后台服务运行
	// 这样HTTP服务器和WebSocket监听器就不会因为main函数退出而关闭
	fmt.Println("\n=== Group Chat Client is Running ===")
	fmt.Println("✅ HTTP server running on :8080")
	fmt.Println("✅ WebSocket connections established")
	fmt.Println("✅ Chat windows opened")
	fmt.Println("📝 Use the HTML windows to send messages")
	fmt.Println("🛑 Press Ctrl+C to exit and cleanup")
	fmt.Println("----------------------------------------")

	// 等待退出信号
	<-sigChan

	// 优雅退出，清理资源
	fmt.Println("\n🛑 Received exit signal, cleaning up...")

	// Close all connections and cleanup
	client.closeAllConnections()

	// Additional cleanup - force delete HTML files
	fmt.Println("Cleaning up chat windows...")
	for _, member := range client.groupInfo.Members {
		filename := fmt.Sprintf("chat_%d.html", member.UserID)
		if err := os.Remove(filename); err == nil {
			fmt.Printf("Deleted chat window file: %s\n", filename)
		}
	}

	fmt.Println("✅ Cleanup completed. Goodbye!")
}

// fetchGroupInfo fetches group information from the API
func (c *GroupChatClient) fetchGroupInfo() error {
	// 直接调用群组服务，而不是通过API网关
	url := fmt.Sprintf("http://localhost:21002/api/v1/group/info")

	reqData := map[string]interface{}{
		"group_id": c.groupID,
		"user_id":  1001, // Use a default user ID for fetching info
	}

	jsonData, _ := json.Marshal(reqData)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "auth-debug")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code first
	if resp.StatusCode != 200 {
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body[:n]))
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Group   struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			MemberCount int32  `json:"member_count"`
		} `json:"group"`
		Members []struct {
			UserID   int64  `json:"user_id"`
			GroupID  int64  `json:"group_id"`
			Role     string `json:"role"`
			Nickname string `json:"nickname"`
			JoinedAt int64  `json:"joined_at"`
		} `json:"members"`
	}

	// Read the entire response body first
	bodyBytes := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			bodyBytes = append(bodyBytes, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	fmt.Printf("API Response: %s\n", string(bodyBytes)) // Debug output

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return fmt.Errorf("JSON decode failed: %v, response: %s", err, string(bodyBytes))
	}

	if !result.Success {
		return fmt.Errorf("API error: %s", result.Message)
	}

	c.groupInfo = GroupInfo{
		ID:          result.Group.ID,
		Name:        result.Group.Name,
		Description: result.Group.Description,
		MemberCount: result.Group.MemberCount,
		Members:     make([]GroupMember, len(result.Members)),
	}

	for i, member := range result.Members {
		nickname := member.Nickname
		if nickname == "" {
			nickname = fmt.Sprintf("User%d", member.UserID)
		}
		c.groupInfo.Members[i] = GroupMember{
			UserID:   member.UserID,
			GroupID:  member.GroupID,
			Role:     member.Role,
			Nickname: nickname,
			JoinedAt: member.JoinedAt,
		}
	}

	return nil
}

// checkOnlineStatus checks online status of group members
func (c *GroupChatClient) checkOnlineStatus() error {
	if len(c.groupInfo.Members) == 0 {
		return nil
	}

	userIDs := make([]int64, len(c.groupInfo.Members))
	for i, member := range c.groupInfo.Members {
		userIDs[i] = member.UserID
	}

	// 直接调用IM网关服务检查在线状态
	url := "http://localhost:21006/api/v1/connect/online_status"
	reqData := map[string]interface{}{
		"user_ids": userIDs,
	}

	jsonData, _ := json.Marshal(reqData)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "auth-debug")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read response body for debugging
	bodyBytes := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			bodyBytes = append(bodyBytes, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	fmt.Printf("Online Status API Response: %s\n", string(bodyBytes)) // Debug output

	var result struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Status  map[string]bool `json:"status"`
		Data    map[string]bool `json:"data"` // Alternative field name
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		fmt.Printf("Failed to parse online status response: %v\n", err)
		// Set all members as offline if API fails
		for i := range c.groupInfo.Members {
			c.groupInfo.Members[i].Online = false
		}
		return nil // Don't fail the whole process
	}

	// Update online status - try both possible field names
	statusMap := result.Status
	if statusMap == nil {
		statusMap = result.Data
	}

	if statusMap != nil {
		for i := range c.groupInfo.Members {
			userIDStr := fmt.Sprintf("%d", c.groupInfo.Members[i].UserID)
			c.groupInfo.Members[i].Online = statusMap[userIDStr]
		}
	} else {
		fmt.Println("No status data found in response, setting all as offline")
		for i := range c.groupInfo.Members {
			c.groupInfo.Members[i].Online = false
		}
	}

	return nil
}

// displayMembers displays all group members with their online status
func (c *GroupChatClient) displayMembers() {
	fmt.Println("\nGroup Members:")
	for i, member := range c.groupInfo.Members {
		status := "Offline"
		if member.Online {
			status = "Online"
		}
		fmt.Printf("  %d. %s (ID: %d) [%s] - %s\n", i+1, member.Nickname, member.UserID, member.Role, status)
	}
	fmt.Println()
}

// connectMember connects WebSocket for a specific member
func (c *GroupChatClient) connectMember(member GroupMember) error {
	wsURL := "ws://localhost:21006/api/v1/connect/ws"

	headers := http.Header{}
	headers.Set("Authorization", "auth-debug")
	headers.Set("User-ID", strconv.FormatInt(member.UserID, 10))

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("WebSocket connection failed: %v", err)
	}

	c.mu.Lock()
	c.connections[member.UserID] = conn
	c.mu.Unlock()

	// Start message receiver for this connection
	go c.receiveMessages(member.UserID, conn)

	return nil
}

// receiveMessages receives messages for a specific user connection
func (c *GroupChatClient) receiveMessages(userID int64, conn *websocket.Conn) {
	fmt.Printf("Started message receiver for user %d\n", userID)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("Connection closed for user %d", userID)
			} else {
				log.Printf("Read message failed for user %d: %v", userID, err)
			}
			break
		}

		fmt.Printf("User %d received WebSocket message, size: %d bytes\n", userID, len(data))

		var wsMsg rest.WSMessage
		if err := proto.Unmarshal(data, &wsMsg); err != nil {
			log.Printf("Message unmarshal failed for user %d: %v", userID, err)
			continue
		}

		fmt.Printf("User %d parsed message: From=%d, Content=%s, GroupID=%d\n", userID, wsMsg.From, wsMsg.Content, wsMsg.GroupId)

		// Add message to all chat windows
		c.addMessageToAllWindows(&wsMsg)
	}
}

// addMessageToAllWindows adds a message to all chat windows
func (c *GroupChatClient) addMessageToAllWindows(wsMsg *rest.WSMessage) {
	// 检查消息是否已经处理过（去重）
	c.mu.Lock()
	if c.processedMsgIDs[wsMsg.MessageId] {
		c.mu.Unlock()
		fmt.Printf("Message %d already processed, skipping...\n", wsMsg.MessageId)
		return
	}
	// 标记消息为已处理
	c.processedMsgIDs[wsMsg.MessageId] = true
	c.mu.Unlock()

	senderNickname := fmt.Sprintf("User%d", wsMsg.From)

	// Find sender nickname
	for _, member := range c.groupInfo.Members {
		if member.UserID == wsMsg.From {
			senderNickname = member.Nickname
			break
		}
	}

	message := ChatMessage{
		From:      wsMsg.From,
		Content:   wsMsg.Content,
		Timestamp: time.Unix(wsMsg.Timestamp, 0),
		Nickname:  senderNickname,
	}

	fmt.Printf("Adding NEW message to windows: From=%s(%d), Content=%s, MsgID=%d\n", senderNickname, wsMsg.From, wsMsg.Content, wsMsg.MessageId)

	c.mu.Lock()
	windowCount := 0
	for _, window := range c.chatWindows {
		window.mu.Lock()
		window.Messages = append(window.Messages, message)
		windowCount++
		fmt.Printf("Added message to window %d, total messages: %d\n", windowCount, len(window.Messages))
		window.mu.Unlock()
	}
	c.mu.Unlock()

	fmt.Printf("Message added to %d windows, updating HTML files...\n", windowCount)

	// Update all chat windows
	c.updateAllChatWindows()
}

// sendMessage sends a message as a specific user
func (c *GroupChatClient) sendMessage(userID int64, content string) error {
	c.mu.RLock()
	conn, exists := c.connections[userID]
	c.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no connection for user %d", userID)
	}

	wsMsg := &rest.WSMessage{
		MessageId:   time.Now().UnixNano(),
		From:        userID,
		To:          0, // Group message
		GroupId:     c.groupID,
		Content:     content,
		Timestamp:   time.Now().Unix(),
		MessageType: 1, // Text message
		AckId:       fmt.Sprintf("ack_%d_%d", userID, time.Now().UnixNano()),
	}

	data, err := proto.Marshal(wsMsg)
	if err != nil {
		return fmt.Errorf("message marshal failed: %v", err)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return fmt.Errorf("WebSocket send failed: %v", err)
	}

	return nil
}

// openChatWindows opens chat windows for each member
func (c *GroupChatClient) openChatWindows() {
	for _, member := range c.groupInfo.Members {
		go c.openChatWindow(member)
		time.Sleep(500 * time.Millisecond) // Stagger window opening
	}
}

// openChatWindow opens a chat window for a specific member
func (c *GroupChatClient) openChatWindow(member GroupMember) {
	windowTitle := fmt.Sprintf("Chat - %s (ID: %d)", member.Nickname, member.UserID)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Create a simple HTML file for the chat window
		htmlContent := c.generateChatHTML(member)
		filename := fmt.Sprintf("chat_%d.html", member.UserID)

		if err := os.WriteFile(filename, []byte(htmlContent), 0644); err != nil {
			log.Printf("Failed to create chat window file for %s: %v", member.Nickname, err)
			return
		}

		cmd = exec.Command("cmd", "/c", "start", windowTitle, filename)
	case "darwin":
		cmd = exec.Command("osascript", "-e", fmt.Sprintf(`tell application "Terminal" to do script "echo 'Chat Window: %s'"`, windowTitle))
	case "linux":
		cmd = exec.Command("xterm", "-title", windowTitle, "-e", "bash")
	default:
		log.Printf("Unsupported OS for opening chat windows: %s", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open chat window for %s: %v", member.Nickname, err)
	}
}

// generateChatHTML generates HTML content for a chat window
func (c *GroupChatClient) generateChatHTML(member GroupMember) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Chat - %s (ID: %d)</title>
    <meta charset="UTF-8">
    <meta http-equiv="refresh" content="2">
    <style>
        body {
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
            margin: 20px;
            background-color: #1e1e1e;
            color: #d4d4d4;
            font-size: 14px;
        }
        .header {
            background-color: #252526;
            color: #cccccc;
            padding: 15px;
            border-radius: 6px;
            margin-bottom: 20px;
            border: 1px solid #3c3c3c;
        }
        .messages {
            height: 450px;
            overflow-y: auto;
            border: 1px solid #3c3c3c;
            padding: 15px;
            background-color: #252526;
            border-radius: 6px;
        }
        .message {
            margin-bottom: 12px;
            padding: 10px 12px;
            border-radius: 4px;
            max-width: 80%%;
            word-wrap: break-word;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
        }
        .message.own {
            background-color: #0e639c;
            margin-left: auto;
            border-left: 3px solid #007acc;
            color: #ffffff;
        }
        .message.other {
            background-color: #2d2d30;
            margin-right: auto;
            border-left: 3px solid #608b4e;
            color: #d4d4d4;
        }
        .message-header {
            font-weight: bold;
            font-size: 0.9em;
            margin-bottom: 4px;
            color: #9cdcfe;
        }
        .timestamp {
            font-size: 0.75em;
            color: #808080;
            margin-top: 4px;
        }
        .status {
            margin-top: 20px;
            padding: 12px;
            background-color: #2d2d30;
            border-radius: 4px;
            border-left: 4px solid #608b4e;
            color: #d4d4d4;
        }
        .input-area {
            margin-top: 20px;
            display: flex;
            gap: 10px;
        }
        .message-input {
            flex: 1;
            padding: 10px;
            background-color: #2d2d30;
            border: 1px solid #3c3c3c;
            border-radius: 4px;
            color: #d4d4d4;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
            font-size: 14px;
        }
        .message-input:focus {
            outline: none;
            border-color: #007acc;
        }
        .send-button {
            padding: 10px 20px;
            background-color: #0e639c;
            border: none;
            border-radius: 4px;
            color: white;
            cursor: pointer;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
        }
        .send-button:hover {
            background-color: #1177bb;
        }
    </style>
    <script>
        let userID = %d;
        let groupID = %d;

        function sendMessage() {
            const input = document.getElementById('messageInput');
            const message = input.value.trim();
            if (message === '') return;

            // 发送消息到Go后端
            fetch('http://localhost:8080/send-message', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    user_id: userID,
                    group_id: groupID,
                    content: message
                })
            }).then(response => {
                if (response.ok) {
                    input.value = '';
                    // 立即刷新页面显示新消息
                    setTimeout(() => location.reload(), 100);
                }
            }).catch(error => {
                console.error('发送消息失败:', error);
            });
        }

        function handleKeyPress(event) {
            if (event.key === 'Enter') {
                sendMessage();
            }
        }

        window.onload = function() {
            const input = document.getElementById('messageInput');
            input.addEventListener('keypress', handleKeyPress);
            input.focus();
        };
    </script>
</head>
<body>
    <div class="header">
        <h2>%s (ID: %d)</h2>
        <p>Group: %s (ID: %d)</p>
    </div>
    <div class="messages" id="messages">
        <div class="message other">
            <strong>System:</strong> Chat window opened for %s
            <div class="timestamp">%s</div>
        </div>
    </div>
    <div class="input-area">
        <input type="text" id="messageInput" class="message-input" placeholder="输入消息内容，按回车发送..." />
        <button onclick="sendMessage()" class="send-button">发送</button>
    </div>
    <div class="status">
        <strong>Status:</strong> Connected to group chat. Type message above and press Enter to send.
    </div>
</body>
</html>
`, // --- CORRECTED ARGUMENTS START HERE ---
		member.Nickname, member.UserID, // For <title>
		member.UserID, c.groupID, // For JavaScript variables
		member.Nickname, member.UserID, // For <h2>
		c.groupInfo.Name, c.groupInfo.ID, // For <p>
		member.Nickname,               // For system message
		time.Now().Format("15:04:05")) // For timestamp
}

// generateChatHTMLWithMessages generates HTML content with actual messages
func (c *GroupChatClient) generateChatHTMLWithMessages(member GroupMember, messages []ChatMessage) string {
	messagesHTML := ""

	// Add system message
	messagesHTML += fmt.Sprintf(`
        <div class="message other">
            <div class="message-header">System</div>
            Chat window opened for %s
            <div class="timestamp">%s</div>
        </div>
    `, member.Nickname, time.Now().Format("15:04:05"))

	// Add all messages
	for _, msg := range messages {
		messageClass := "other"
		if msg.From == member.UserID {
			messageClass = "own"
		}

		messagesHTML += fmt.Sprintf(`
        <div class="message %s">
            <div class="message-header">%s (ID: %d)</div>
            %s
            <div class="timestamp">%s</div>
        </div>
        `, messageClass, msg.Nickname, msg.From, msg.Content, msg.Timestamp.Format("15:04:05"))
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Chat - %s (ID: %d)</title>
    <meta charset="UTF-8">
    <meta http-equiv="refresh" content="2">
    <style>
        body {
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
            margin: 20px;
            background-color: #1e1e1e;
            color: #d4d4d4;
            font-size: 14px;
        }
        .header {
            background-color: #252526;
            color: #cccccc;
            padding: 15px;
            border-radius: 6px;
            margin-bottom: 20px;
            border: 1px solid #3c3c3c;
        }
        .messages {
            height: 450px;
            overflow-y: auto;
            border: 1px solid #3c3c3c;
            padding: 15px;
            background-color: #252526;
            border-radius: 6px;
        }
        .message {
            margin-bottom: 12px;
            padding: 10px 12px;
            border-radius: 4px;
            max-width: 80%%;
            word-wrap: break-word;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
        }
        .message.own {
            background-color: #0e639c;
            margin-left: auto;
            border-left: 3px solid #007acc;
            color: #ffffff;
        }
        .message.other {
            background-color: #2d2d30;
            margin-right: auto;
            border-left: 3px solid #608b4e;
            color: #d4d4d4;
        }
        .message-header {
            font-weight: bold;
            font-size: 0.9em;
            margin-bottom: 4px;
            color: #9cdcfe;
        }
        .timestamp {
            font-size: 0.75em;
            color: #808080;
            margin-top: 4px;
        }
        .status {
            margin-top: 20px;
            padding: 12px;
            background-color: #2d2d30;
            border-radius: 4px;
            border-left: 4px solid #608b4e;
            color: #d4d4d4;
        }
        .input-area {
            margin-top: 20px;
            display: flex;
            gap: 10px;
        }
        .message-input {
            flex: 1;
            padding: 10px;
            background-color: #2d2d30;
            border: 1px solid #3c3c3c;
            border-radius: 4px;
            color: #d4d4d4;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
            font-size: 14px;
        }
        .message-input:focus {
            outline: none;
            border-color: #007acc;
        }
        .send-button {
            padding: 10px 20px;
            background-color: #0e639c;
            border: none;
            border-radius: 4px;
            color: white;
            cursor: pointer;
            font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
        }
        .send-button:hover {
            background-color: #1177bb;
        }
    </style>
    <script>
        let userID = %d;
        let groupID = %d;

        function sendMessage() {
            const input = document.getElementById('messageInput');
            const message = input.value.trim();
            if (message === '') return;

            // 发送消息到Go后端
            fetch('http://localhost:8080/send-message', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    user_id: userID,
                    group_id: groupID,
                    content: message
                })
            }).then(response => {
                if (response.ok) {
                    input.value = '';
                    // 立即刷新页面显示新消息
                    setTimeout(() => location.reload(), 100);
                }
            }).catch(error => {
                console.error('发送消息失败:', error);
            });
        }

        function handleKeyPress(event) {
            if (event.key === 'Enter') {
                sendMessage();
            }
        }

        // Auto scroll to bottom and setup input
        window.onload = function() {
            var messages = document.getElementById('messages');
            messages.scrollTop = messages.scrollHeight;

            const input = document.getElementById('messageInput');
            input.addEventListener('keypress', handleKeyPress);
            input.focus();
        };
    </script>
</head>
<body>
    <div class="header">
        <h2>%s (ID: %d)</h2>
        <p>Group: %s (ID: %d) | Messages: %d</p>
    </div>
    <div class="messages" id="messages">
        %s
    </div>
    <div class="input-area">
        <input type="text" id="messageInput" class="message-input" placeholder="输入消息内容，按回车发送..." />
        <button onclick="sendMessage()" class="send-button">发送</button>
    </div>
    <div class="status">
        <strong>Status:</strong> Connected to group chat. Type message above and press Enter to send.
    </div>
</body>
</html>
`, // --- CORRECTED ARGUMENTS START HERE ---
		member.Nickname, member.UserID, // For <title>
		member.UserID, c.groupID, // For JavaScript variables
		member.Nickname, member.UserID, // For <h2>
		c.groupInfo.Name, c.groupInfo.ID, // For <p>
		len(messages), // For message count in <p>
		messagesHTML)  // For the messages div
}

// updateAllChatWindows updates all chat windows by regenerating HTML files
func (c *GroupChatClient) updateAllChatWindows() {
	fmt.Printf("[%s] New message received in group %s\n", time.Now().Format("15:04:05"), c.groupInfo.Name)

	// Update HTML files for each member
	for _, member := range c.groupInfo.Members {
		c.updateChatWindowHTML(member)
	}
}

// updateChatWindowHTML updates the HTML file for a specific member
func (c *GroupChatClient) updateChatWindowHTML(member GroupMember) {
	filename := fmt.Sprintf("chat_%d.html", member.UserID)

	c.mu.RLock()
	window, exists := c.chatWindows[member.UserID]
	c.mu.RUnlock()

	if !exists {
		return
	}

	window.mu.Lock()
	messages := make([]ChatMessage, len(window.Messages))
	copy(messages, window.Messages)
	window.mu.Unlock()

	// Generate updated HTML content
	htmlContent := c.generateChatHTMLWithMessages(member, messages)

	// Write to file
	if err := os.WriteFile(filename, []byte(htmlContent), 0644); err != nil {
		log.Printf("Failed to update chat window file for %s: %v", member.Nickname, err)
	}
}

// closeAllConnections closes all WebSocket connections
func (c *GroupChatClient) closeAllConnections() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for userID, conn := range c.connections {
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close connection for user %d: %v", userID, err)
		}
	}

	// Clean up HTML files
	for _, member := range c.groupInfo.Members {
		filename := fmt.Sprintf("chat_%d.html", member.UserID)
		if err := os.Remove(filename); err != nil {
			// Ignore errors for file cleanup
		}
	}
}
