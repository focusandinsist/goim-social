package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
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

// GroupChatClient 群聊客户端
type GroupChatClient struct {
	groupID       int64
	groupInfo     GroupInfo
	chatWindows   map[int64]*ChatWindow
	connections   map[int64]*websocket.Conn
	defaultSender int64 // 默认发送者ID
	mu            sync.RWMutex
}

func main() {
	fmt.Println("=== Multi-Window Group Chat Client ===")
	fmt.Println("Using auth-debug token for debugging")
	fmt.Println()

	client := &GroupChatClient{
		chatWindows: make(map[int64]*ChatWindow),
		connections: make(map[int64]*websocket.Conn),
	}

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

	// Main command loop
	fmt.Println("\nCommands:")
	fmt.Println("  <message> - Send message as default user")
	fmt.Println("  @<userID> <message> - Send message as specific user")
	fmt.Println("  list - Show all members")
	fmt.Println("  quit - Exit")
	fmt.Println("----------------------------------------")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "quit" {
			fmt.Println("Exiting...")
			break
		}

		if input == "list" {
			client.displayMembers()
			continue
		}

		// Check if it's @userID message format
		if strings.HasPrefix(input, "@") {
			parts := strings.SplitN(input[1:], " ", 2)
			if len(parts) < 2 {
				fmt.Println("Use format: @<userID> <message>")
				continue
			}

			userID, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				fmt.Printf("Invalid user ID: %s\n", parts[0])
				continue
			}

			content := parts[1]

			if err := client.sendMessage(userID, content); err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
			} else {
				fmt.Printf("Message sent by user %d\n", userID)
			}
		} else {
			// Direct message using default sender
			if client.defaultSender == 0 {
				fmt.Println("No default sender available. Use @<userID> <message> format.")
				continue
			}

			if err := client.sendMessage(client.defaultSender, input); err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
			} else {
				defaultSenderName := "Unknown"
				for _, member := range client.groupInfo.Members {
					if member.UserID == client.defaultSender {
						defaultSenderName = member.Nickname
						break
					}
				}
				fmt.Printf("Message sent by %s (ID: %d)\n", defaultSenderName, client.defaultSender)
			}
		}
	}

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

	fmt.Printf("Adding message to windows: From=%s(%d), Content=%s\n", senderNickname, wsMsg.From, wsMsg.Content)

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
            max-width: 80%;
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
    </style>
    <script>
        function refreshMessages() {
            // This would be updated by the Go application
            setTimeout(refreshMessages, 1000);
        }
        window.onload = function() {
            refreshMessages();
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
    <div class="status">
        <strong>Status:</strong> Connected to group chat. Messages will appear here automatically.
    </div>
</body>
</html>
`, member.Nickname, member.UserID, c.groupInfo.Name, c.groupInfo.ID, member.Nickname, time.Now().Format("15:04:05"))
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
    </style>
    <script>
        // Auto scroll to bottom
        window.onload = function() {
            var messages = document.getElementById('messages');
            messages.scrollTop = messages.scrollHeight;
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
    <div class="status">
        <strong>Status:</strong> Connected to group chat. Auto-refresh every 2 seconds.
    </div>
</body>
</html>
`, member.Nickname, member.UserID, c.groupInfo.Name, c.groupInfo.ID, len(messages), messagesHTML)
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
