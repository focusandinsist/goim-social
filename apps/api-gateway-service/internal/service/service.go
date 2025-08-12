package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"goim-social/api/rest"
	"goim-social/apps/api-gateway-service/internal/model"
	"goim-social/pkg/config"
	tracecontext "goim-social/pkg/context"
	"goim-social/pkg/database"
	"goim-social/pkg/discovery"
	"goim-social/pkg/kafka"
	"goim-social/pkg/logger"
	"goim-social/pkg/redis"
	"goim-social/pkg/telemetry"
)

// Service API网关服务
type Service struct {
	db              *database.MongoDB
	redis           *redis.RedisClient
	kafka           *kafka.Producer
	config          *config.Config
	discovery       discovery.Discovery
	logger          logger.Logger
	imGatewayConn   *grpc.ClientConn
	imGatewayClient rest.ConnectServiceClient
}

// NewService 创建API网关服务实例
func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer, cfg *config.Config, logger logger.Logger) *Service {
	// 获取 Kubernetes 命名空间
	namespace := os.Getenv("KUBERNETES_NAMESPACE")
	if namespace == "" {
		namespace = "im-system" // 默认命名空间
	}

	// 初始化 Kubernetes 服务发现
	k8sDiscovery, err := discovery.NewK8sDiscovery(namespace, logger)
	if err != nil {
		log.Printf("Failed to initialize Kubernetes service discovery: %v", err)
		// 在生产环境中，这里应该是致命错误
		panic(err)
	}

	// 初始化IM Gateway gRPC客户端（使用服务发现）
	var imGatewayConn *grpc.ClientConn
	var imGatewayClient rest.ConnectServiceClient

	if instance, err := k8sDiscovery.GetServiceInstance("im-gateway-service"); err == nil {
		imGatewayAddr := fmt.Sprintf("%s:%d", instance.Host, instance.GRPCPort)
		conn, err := grpc.NewClient(imGatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error(context.Background(), "Failed to connect to IM Gateway")
		} else {
			imGatewayConn = conn
			imGatewayClient = rest.NewConnectServiceClient(conn)
		}
	}

	service := &Service{
		db:              db,
		redis:           redis,
		kafka:           kafka,
		config:          cfg,
		discovery:       k8sDiscovery,
		logger:          logger,
		imGatewayConn:   imGatewayConn,
		imGatewayClient: imGatewayClient,
	}

	logger.Info(context.Background(), "API Gateway service initialized with Kubernetes service discovery")

	return service
}

// GetServiceInstance 获取服务实例（负载均衡）
func (s *Service) GetServiceInstance(serviceName string) (*discovery.ServiceInstance, error) {
	// 使用 Kubernetes 服务发现获取实例
	return s.discovery.GetServiceInstance(serviceName)
}

// ProxyRequest 动态路由代理请求
func (s *Service) ProxyRequest(w http.ResponseWriter, r *http.Request) error {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(r.Context(), "api-gateway.proxy.RouteRequest")
	defer span.End()

	// 解析URL路径，提取服务名
	// 期望的URL格式: /api/v1/{service-name}/{remaining-path}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		span.SetStatus(codes.Error, "invalid URL format")
		http.Error(w, "Invalid URL format. Expected: /api/v1/{service-name}/{path}", http.StatusBadRequest)
		return fmt.Errorf("invalid URL format: %s", r.URL.Path)
	}

	serviceName := pathParts[2] // 第三部分是服务名

	// 设置span属性
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
		attribute.String("gateway.target_service", serviceName),
		attribute.String("http.user_agent", r.Header.Get("User-Agent")),
	)

	// 获取服务实例
	instance, err := s.GetServiceInstance(serviceName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "service not available")
		http.Error(w, fmt.Sprintf("Service %s not available", serviceName), http.StatusServiceUnavailable)
		return err
	}

	// 构建目标URL
	targetURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", instance.Host, instance.Port),
		Path:   "/" + strings.Join(pathParts[3:], "/"), // 剩余路径
	}

	// 保留查询参数
	targetURL.RawQuery = r.URL.RawQuery

	// 设置目标URL到span属性
	span.SetAttributes(
		attribute.String("gateway.target_url", targetURL.String()),
		attribute.String("gateway.target_host", instance.Host),
		attribute.Int("gateway.target_port", int(instance.Port)),
	)

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 修改请求，将context传递下去
	r = r.WithContext(ctx)
	r.URL.Host = targetURL.Host
	r.URL.Scheme = targetURL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Header.Set("X-Origin-Host", targetURL.Host)

	// 执行代理
	proxy.ServeHTTP(w, r)

	log.Printf("API Gateway: Proxied %s %s to %s", r.Method, r.URL.Path, targetURL.String())
	span.SetStatus(codes.Ok, "request proxied successfully")
	return nil
}

// GetAllServices 获取所有注册的服务
func (s *Service) GetAllServices() map[string][]*discovery.ServiceInstance {
	return s.discovery.GetAllServices()
}

// GetOnlineStatusFromIMGateway 通过gRPC调用IM Gateway获取在线状态
func (s *Service) GetOnlineStatusFromIMGateway(ctx context.Context, userIDs []int64) (map[int64]bool, error) {
	if s.imGatewayClient == nil {
		return nil, fmt.Errorf("IM Gateway client not initialized")
	}

	req := &rest.OnlineStatusRequest{
		UserIds: userIDs,
	}

	resp, err := s.imGatewayClient.OnlineStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to call IM Gateway: %v", err)
	}

	return resp.Status, nil
}

// OnlineStatus 查询用户在线状态
func (s *Service) OnlineStatus(ctx context.Context, userIDs []int64) (map[int64]bool, error) {
	// 开始OpenTelemetry span
	ctx, span := telemetry.StartSpan(ctx, "api-gateway.service.OnlineStatus")
	defer span.End()

	// 设置span属性
	span.SetAttributes(
		attribute.Int("query.user_count", len(userIDs)),
	)

	// 将业务信息添加到context（如果有用户ID的话）
	if len(userIDs) > 0 {
		ctx = tracecontext.WithUserID(ctx, userIDs[0])
	}

	status := make(map[int64]bool)

	for _, uid := range userIDs {
		// 查询Redis中的用户连接信息
		pattern := fmt.Sprintf("conn:%d:*", uid)
		keys, err := s.redis.Keys(ctx, pattern)
		if err != nil {
			log.Printf("查询用户 %d 连接信息失败: %v", uid, err)
			status[uid] = false
			continue
		}

		// 如果有连接记录，说明用户在线
		status[uid] = len(keys) > 0
	}

	// 统计在线用户数量
	onlineCount := 0
	for _, online := range status {
		if online {
			onlineCount++
		}
	}

	span.SetAttributes(attribute.Int("result.online_count", onlineCount))
	span.SetStatus(codes.Ok, "online status queried successfully")
	return status, nil
}

// GetUserConnections 获取用户的所有连接信息
func (s *Service) GetUserConnections(ctx context.Context, userID int64) ([]*model.Connection, error) {
	pattern := fmt.Sprintf("conn:%d:*", userID)
	keys, err := s.redis.Keys(ctx, pattern)
	if err != nil {
		return nil, fmt.Errorf("查询用户连接失败: %v", err)
	}

	var connections []*model.Connection
	for _, key := range keys {
		connInfo, err := s.redis.HGetAll(ctx, key)
		if err != nil {
			log.Printf("获取连接信息失败: %v", err)
			continue
		}

		// 解析连接信息
		conn := &model.Connection{
			UserID:     userID,
			ConnID:     connInfo["connID"],
			ServerID:   connInfo["serverID"],
			ClientType: connInfo["clientType"],
			Online:     true,
		}

		connections = append(connections, conn)
	}

	return connections, nil
}
