package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"websocket-server/api/rest"
	"websocket-server/apps/message/consumer"
	"websocket-server/apps/message/model"
	"websocket-server/pkg/database"
	"websocket-server/pkg/kafka"
	"websocket-server/pkg/redis"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Service struct {
	db    *database.MongoDB
	redis *redis.RedisClient
	kafka *kafka.Producer
}

// 移除本地StreamManager，使用consumer包中的全局StreamManager

func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer) *Service {
	return &Service{
		db:    db,
		redis: redis,
		kafka: kafka,
	}
}

// SendMessage 发送消息
func (s *Service) SendMessage(ctx context.Context, msg *model.Message) error {
	// TODO: 持久化消息、推送到目标用户/群组
	return nil
}

// GetHistory 获取历史消息
func (s *Service) GetHistory(ctx context.Context, userID, groupID int64, page, size int) ([]*model.Message, int, error) {
	collection := s.db.GetCollection("message")

	// 构建查询条件
	var filter map[string]interface{}
	if groupID > 0 {
		// 群聊消息
		filter = map[string]interface{}{
			"group_id": groupID,
		}
	} else {
		// 私聊消息：查询与该用户相关的所有消息（发送给他的或他发送的）
		filter = map[string]interface{}{
			"$or": []map[string]interface{}{
				{"from": userID},
				{"to": userID},
			},
			"group_id": 0, // 确保是私聊消息
		}
	}

	// 计算跳过的记录数
	skip := int64((page - 1) * size)
	limit := int64(size)

	// 查询总数
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("❌ 查询历史消息总数失败: %v", err)
		return nil, 0, err
	}

	// 查询消息列表（按时间正序，最早的消息在前）
	cursor, err := collection.Find(ctx, filter, &options.FindOptions{
		Sort:  map[string]interface{}{"created_at": 1}, // 按创建时间正序
		Skip:  &skip,
		Limit: &limit,
	})
	if err != nil {
		log.Printf("❌ 查询历史消息失败: %v", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var messages []*model.Message
	for cursor.Next(ctx) {
		var msg model.Message
		if err := cursor.Decode(&msg); err != nil {
			log.Printf("❌ 解析历史消息失败: %v", err)
			continue
		}
		messages = append(messages, &msg)
	}

	log.Printf("✅ 查询历史消息成功: 用户=%d, 群组=%d, 总数=%d, 返回=%d", userID, groupID, total, len(messages))
	return messages, int(total), nil
}

// GetUnreadMessages 获取未读消息
func (s *Service) GetUnreadMessages(ctx context.Context, userID int64) ([]*model.Message, error) {
	collection := s.db.GetCollection("message")

	// 查询发给该用户的未读消息
	filter := map[string]interface{}{
		"to":     userID,
		"status": 0, // 0:未读
	}

	// 按时间正序排列（最早的消息先显示）
	cursor, err := collection.Find(ctx, filter, &options.FindOptions{
		Sort: map[string]interface{}{"created_at": 1},
	})
	if err != nil {
		log.Printf("❌ 查询未读消息失败: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*model.Message
	for cursor.Next(ctx) {
		var msg model.Message
		if err := cursor.Decode(&msg); err != nil {
			log.Printf("❌ 解析未读消息失败: %v", err)
			continue
		}
		messages = append(messages, &msg)
	}

	log.Printf("✅ 查询未读消息成功: 用户=%d, 未读消息数=%d", userID, len(messages))
	return messages, nil
}

// MarkMessagesAsRead 标记消息为已读
func (s *Service) MarkMessagesAsRead(ctx context.Context, userID int64, messageIDs []string) error {
	collection := s.db.GetCollection("message")

	// 将字符串ID转换为ObjectID
	var objectIDs []interface{}
	for _, idStr := range messageIDs {
		if objectID, err := primitive.ObjectIDFromHex(idStr); err == nil {
			objectIDs = append(objectIDs, objectID)
		} else {
			log.Printf("⚠️ 无效的消息ID: %s", idStr)
		}
	}

	if len(objectIDs) == 0 {
		log.Printf("⚠️ 没有有效的消息ID需要标记")
		return nil
	}

	// 构建更新条件
	filter := map[string]interface{}{
		"_id": map[string]interface{}{
			"$in": objectIDs,
		},
		"to": userID, // 确保只能标记发给自己的消息
	}

	// 更新状态为已读
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"status":     1, // 1:已读
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Printf("❌ 标记消息已读失败: %v", err)
		return err
	}

	log.Printf("✅ 标记消息已读成功: 用户=%d, 更新数量=%d", userID, result.ModifiedCount)
	return nil
}

// MarkMessageAsReadByID 根据消息ID标记单条消息为已读
func (s *Service) MarkMessageAsReadByID(ctx context.Context, userID int64, messageID int64) error {
	collection := s.db.GetCollection("message")

	// 构建更新条件
	filter := map[string]interface{}{
		"message_id": messageID,
		"to":         userID, // 确保只能标记发给自己的消息
	}

	// 更新状态为已读
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"status":     1, // 1:已读
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("❌ 标记消息已读失败: MessageID=%d, UserID=%d, Error=%v", messageID, userID, err)
		return err
	}

	if result.ModifiedCount == 0 {
		log.Printf("⚠️ 没有找到需要标记的消息: MessageID=%d, UserID=%d", messageID, userID)
		return fmt.Errorf("消息不存在或已经是已读状态")
	}

	log.Printf("✅ 消息已标记为已读: MessageID=%d, UserID=%d", messageID, userID)
	return nil
}

// HandleWSMessage 处理 WebSocket 消息收发并存储到 MongoDB
func (s *Service) HandleWSMessage(ctx context.Context, wsMsg *model.WSMessage, conn *websocket.Conn) error {
	// 构造消息
	msg := &model.HistoryMessage{
		ID:        wsMsg.MessageID,
		From:      wsMsg.From,
		To:        wsMsg.To,
		GroupID:   wsMsg.GroupID,
		Content:   wsMsg.Content,
		MsgType:   wsMsg.MessageType,
		AckID:     wsMsg.AckID,
		CreatedAt: time.Now(),
		Status:    0, // 默认未读
	}
	// 存储到 MongoDB 消息表（collection: message）
	_, err := s.db.GetCollection("message").InsertOne(ctx, msg)
	if err != nil {
		return err
	}
	// 回显消息（可扩展为推送给目标用户）
	resp, err := json.Marshal(wsMsg)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, resp)
}

// gRPC接口实现
func (s *Service) SendWSMessage(ctx context.Context, req *rest.SendWSMessageRequest) (*rest.SendWSMessageResponse, error) {
	msg := req.Msg
	// 持久化到MongoDB
	_, err := s.db.GetCollection("message").InsertOne(ctx, msg)
	if err != nil {
		return &rest.SendWSMessageResponse{Success: false, Message: err.Error()}, nil
	}
	// 可选: 推送到Kafka等
	return &rest.SendWSMessageResponse{Success: true, Message: "OK"}, nil
}

type GRPCService struct {
	rest.UnimplementedMessageServiceServer
	svc *Service
}

// NewGRPCService 构造函数
func (s *Service) NewGRPCService(svc *Service) *GRPCService {
	return &GRPCService{svc: svc}
}

func (g *GRPCService) SendWSMessage(ctx context.Context, req *rest.SendWSMessageRequest) (*rest.SendWSMessageResponse, error) {
	log.Printf("📥 Message服务接收消息: From=%d, To=%d, Content=%s", req.Msg.From, req.Msg.To, req.Msg.Content)

	// 1. 发布消息到Kafka（异步处理）
	messageEvent := map[string]interface{}{
		"type":      "new_message",
		"message":   req.Msg,
		"timestamp": time.Now().Unix(),
	}

	if err := g.svc.kafka.PublishMessage("message-events", messageEvent); err != nil {
		log.Printf("❌ 发布消息到Kafka失败: %v", err)
		return &rest.SendWSMessageResponse{Success: false, Message: "消息发送失败"}, err
	}

	log.Printf("✅ 消息已发布到Kafka: From=%d, To=%d", req.Msg.From, req.Msg.To)
	return &rest.SendWSMessageResponse{Success: true, Message: "消息发送成功"}, nil
}

// GetHistoryMessages gRPC接口：获取历史消息
func (g *GRPCService) GetHistoryMessages(ctx context.Context, req *rest.GetHistoryRequest) (*rest.GetHistoryResponse, error) {
	log.Printf("📜 获取历史消息请求: UserID=%d, GroupID=%d, Page=%d, Size=%d", req.UserId, req.GroupId, req.Page, req.Size)

	// 调用service层获取历史消息
	messages, total, err := g.svc.GetHistory(ctx, req.UserId, req.GroupId, int(req.Page), int(req.Size))
	if err != nil {
		log.Printf("❌ 获取历史消息失败: %v", err)
		return nil, err
	}

	// 将model.Message转换为rest.WSMessage
	var wsMessages []*rest.WSMessage
	for _, msg := range messages {
		wsMsg := &rest.WSMessage{
			MessageId:   0, // ObjectID无法直接转换为int64，暂时设为0
			From:        msg.From,
			To:          msg.To,
			GroupId:     msg.GroupID,
			Content:     msg.Content,
			Timestamp:   msg.CreatedAt.Unix(),
			MessageType: msg.MsgType,
			AckId:       msg.AckID,
		}
		wsMessages = append(wsMessages, wsMsg)
	}

	log.Printf("✅ 获取历史消息成功: 总数=%d, 返回=%d", total, len(wsMessages))
	return &rest.GetHistoryResponse{
		Messages: wsMessages,
		Total:    int32(total),
		Page:     req.Page,
		Size:     req.Size,
	}, nil
}

// MessageStream 实现双向流通信
func (g *GRPCService) MessageStream(stream rest.MessageService_MessageStreamServer) error {
	// 存储连接的Connect服务实例
	var connectServiceID string

	// 获取全局流管理器
	streamManager := consumer.GetStreamManager()

	// 在函数返回时移除连接
	defer func() {
		if connectServiceID != "" {
			streamManager.RemoveStream(connectServiceID)
		}
	}()

	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		switch reqType := req.RequestType.(type) {
		case *rest.MessageStreamRequest_Subscribe:
			// Connect服务订阅消息推送
			connectServiceID = reqType.Subscribe.ConnectServiceId
			log.Printf("Connect服务 %s 已订阅消息推送", connectServiceID)

			// 添加到连接管理器
			streamManager.AddStream(connectServiceID, stream)

		case *rest.MessageStreamRequest_Ack:
			// 处理消息确认
			ack := reqType.Ack
			log.Printf("收到消息确认: MessageID=%d, UserID=%d", ack.MessageId, ack.UserId)

			// 标记消息为已读
			err := g.svc.MarkMessageAsReadByID(stream.Context(), ack.UserId, ack.MessageId)
			if err != nil {
				log.Printf("❌ 标记消息已读失败: MessageID=%d, UserID=%d, Error=%v", ack.MessageId, ack.UserId, err)
			} else {
				log.Printf("✅ 消息已标记为已读: MessageID=%d, UserID=%d", ack.MessageId, ack.UserId)
			}

			// 发送确认回复
			stream.Send(&rest.MessageStreamResponse{
				ResponseType: &rest.MessageStreamResponse_AckConfirm{
					AckConfirm: &rest.AckConfirmEvent{
						AckId:     ack.AckId,
						MessageId: ack.MessageId,
						Confirmed: true,
					},
				},
			})

		case *rest.MessageStreamRequest_PushResult:
			// 处理推送结果反馈
			result := reqType.PushResult
			if result.Success {
				log.Printf("消息推送成功: UserID=%d", result.TargetUserId)
			} else {
				log.Printf("消息推送失败: UserID=%d, Error=%s", result.TargetUserId, result.ErrorMessage)
			}

		case *rest.MessageStreamRequest_SendMessage:
			// 处理通过双向流发送的消息
			sendReq := reqType.SendMessage
			log.Printf("📥 通过双向流接收消息: From=%d, To=%d, Content=%s", sendReq.Msg.From, sendReq.Msg.To, sendReq.Msg.Content)

			// 调用现有的SendWSMessage方法处理消息
			_, err := g.SendWSMessage(stream.Context(), sendReq)
			if err != nil {
				log.Printf("❌ 处理双向流消息失败: %v", err)
			}
		}
	}
}
