package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"websocket-server/api/rest"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

func main() {
	// 命令行参数
	var (
		userID   = flag.Int64("user", 1001, "用户ID")
		targetID = flag.Int64("target", 1002, "目标用户ID")
		wsURL    = flag.String("url", "ws://localhost:21005/api/v1/connect/ws", "WebSocket服务地址")
		token    = flag.String("token", "auth-debug", "认证token")
		autoMode = flag.Bool("auto", false, "自动模式，自动发送消息")
	)
	flag.Parse()

	// 设置连接头
	headers := make(map[string][]string)
	headers["Authorization"] = []string{*token}
	headers["User-ID"] = []string{fmt.Sprintf("%d", *userID)}

	// 连接WebSocket
	c, _, err := websocket.DefaultDialer.Dial(*wsURL, headers)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer c.Close()

	fmt.Printf("客户端启动成功 - 用户ID: %d, 目标用户: %d\n", *userID, *targetID)
	fmt.Println("输入消息内容，按回车发送。输入 'exit' 退出。")

	// 启动消息接收协程
	go receiveMessages(c, *userID)

	if *autoMode {
		// 自动模式
		go autoSendMessages(c, *userID, *targetID)
	}

	// 主循环处理用户输入
	for {
		var input string
		fmt.Printf("[用户%d] 请输入消息: ", *userID)
		fmt.Scanln(&input)

		if input == "exit" {
			fmt.Println("已退出客户端。")
			break
		}

		if input == "" {
			continue
		}

		// 发送消息
		msg := &rest.WSMessage{
			MessageType: 1,
			From:        *userID,
			To:          *targetID,
			Content:     input,
			Timestamp:   time.Now().Unix(),
		}

		data, err := proto.Marshal(msg)
		if err != nil {
			log.Printf("proto 序列化失败: %v", err)
			continue
		}

		if err := c.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Printf("发送消息失败: %v", err)
			continue
		}

		fmt.Printf("[用户%d] 消息已发送: %s\n", *userID, input)
	}
}

// 接收消息的协程
func receiveMessages(c *websocket.Conn, userID int64) {
	for {
		c.SetReadDeadline(time.Now().Add(30 * time.Second))
		_, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
				log.Printf("连接被关闭: %v", err)
			}
			return
		}

		// 解析消息
		var wsMsg rest.WSMessage
		if err := proto.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 调试日志：显示收到的所有消息
		log.Printf("🔍 [用户%d] 收到消息: From=%d, To=%d, Content=%s", userID, wsMsg.From, wsMsg.To, wsMsg.Content)

		// 只显示发给当前用户的消息
		if wsMsg.To == userID {
			fmt.Printf("\n[收到消息] 来自用户%d: %s\n", wsMsg.From, wsMsg.Content)
			fmt.Printf("[用户%d] 请输入消息: ", userID)
		} else {
			log.Printf("⚠️  [用户%d] 消息不是发给我的，忽略", userID)
		}
	}
}

// 自动发送消息的协程
func autoSendMessages(c *websocket.Conn, userID, targetID int64) {
	counter := 1
	for {
		time.Sleep(5 * time.Second) // 每5秒发送一条消息

		msg := &rest.WSMessage{
			MessageType: 1,
			From:        userID,
			To:          targetID,
			Content:     fmt.Sprintf("自动消息 #%d", counter),
			Timestamp:   time.Now().Unix(),
		}

		data, err := proto.Marshal(msg)
		if err != nil {
			log.Printf("自动发送消息失败: %v", err)
			continue
		}

		if err := c.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Printf("自动发送消息失败: %v", err)
			continue
		}

		fmt.Printf("[用户%d] 自动发送消息: 自动消息 #%d\n", userID, counter)
		counter++
	}
}
