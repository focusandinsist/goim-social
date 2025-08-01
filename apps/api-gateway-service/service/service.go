package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"goim-social/api/rest"
	"goim-social/apps/api-gateway-service/model"
	"goim-social/pkg/config"
	"goim-social/pkg/database"
	"goim-social/pkg/kafka"
	"goim-social/pkg/redis"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ServiceInstance 服务实例信息
type ServiceInstance struct {
	ServiceName string `json:"service_name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Healthy     bool   `json:"healthy"`
	LastCheck   int64  `json:"last_check"`
}

// ServiceRegistry 服务注册表
type ServiceRegistry struct {
	services map[string][]*ServiceInstance
	mutex    sync.RWMutex
}

// Service API网关服务
type Service struct {
	db              *database.MongoDB
	redis           *redis.RedisClient
	kafka           *kafka.Producer
	config          *config.Config
	registry        *ServiceRegistry
	imGatewayConn   *grpc.ClientConn
	imGatewayClient rest.ConnectServiceClient
}

// NewService 创建API网关服务实例
func NewService(db *database.MongoDB, redis *redis.RedisClient, kafka *kafka.Producer, cfg *config.Config) *Service {
	registry := &ServiceRegistry{
		services: make(map[string][]*ServiceInstance),
	}

	// 初始化IM Gateway gRPC客户端
	imGatewayAddr := "localhost:22006" // IM Gateway的gRPC端口
	conn, err := grpc.Dial(imGatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to IM Gateway: %v", err)
		// 在实际项目中，这里可能需要重试机制
	}

	service := &Service{
		db:              db,
		redis:           redis,
		kafka:           kafka,
		config:          cfg,
		registry:        registry,
		imGatewayConn:   conn,
		imGatewayClient: rest.NewConnectServiceClient(conn),
	}

	// 启动服务发现
	go service.startServiceDiscovery()

	return service
}

// startServiceDiscovery 启动服务发现（模拟K8s服务发现）
func (s *Service) startServiceDiscovery() {
	// TODO:
	// 这里模拟从k8sAPI服务列表
	// prod中，这里会调用k8sAPI
	// 或者nacos?

	// 模拟注册一些服务实例
	s.registerService("user-service", "localhost", 21001)
	s.registerService("group-service", "localhost", 21002)
	s.registerService("friend-service", "localhost", 21003)
	s.registerService("message-service", "localhost", 21004)
	s.registerService("logic-service", "localhost", 21005)
	s.registerService("content-service", "localhost", 21008)
	s.registerService("interaction-service", "localhost", 21009)
	s.registerService("comment-service", "localhost", 21010)
	s.registerService("history-service", "localhost", 21011)

	log.Println("API Gateway: Service discovery started")
}

// registerService 注册服务实例
func (s *Service) registerService(serviceName, host string, port int) {
	s.registry.mutex.Lock()
	defer s.registry.mutex.Unlock()

	instance := &ServiceInstance{
		ServiceName: serviceName,
		Host:        host,
		Port:        port,
		Healthy:     true,
		LastCheck:   0, // TODO
	}

	s.registry.services[serviceName] = append(s.registry.services[serviceName], instance)
	log.Printf("API Gateway: Registered service %s at %s:%d", serviceName, host, port)
}

// GetServiceInstance 获取服务实例（负载均衡）
func (s *Service) GetServiceInstance(serviceName string) (*ServiceInstance, error) {
	s.registry.mutex.RLock()
	defer s.registry.mutex.RUnlock()

	instances, exists := s.registry.services[serviceName]
	if !exists || len(instances) == 0 {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// TODO:轮询不太好，待优化
	for _, instance := range instances {
		if instance.Healthy {
			return instance, nil
		}
	}

	return nil, fmt.Errorf("no healthy instance found for service %s", serviceName)
}

// ProxyRequest 动态路由代理请求
func (s *Service) ProxyRequest(w http.ResponseWriter, r *http.Request) error {
	// 解析URL路径，提取服务名
	// 期望的URL格式: /api/v1/{service-name}/{remaining-path}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL format. Expected: /api/v1/{service-name}/{path}", http.StatusBadRequest)
		return fmt.Errorf("invalid URL format: %s", r.URL.Path)
	}

	serviceName := pathParts[2] // 第三部分是服务名

	// 获取服务实例
	instance, err := s.GetServiceInstance(serviceName)
	if err != nil {
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

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 修改请求
	r.URL.Host = targetURL.Host
	r.URL.Scheme = targetURL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Header.Set("X-Origin-Host", targetURL.Host)

	// 执行代理
	proxy.ServeHTTP(w, r)

	log.Printf("API Gateway: Proxied %s %s to %s", r.Method, r.URL.Path, targetURL.String())
	return nil
}

// GetAllServices 获取所有注册的服务
func (s *Service) GetAllServices() map[string][]*ServiceInstance {
	s.registry.mutex.RLock()
	defer s.registry.mutex.RUnlock()

	// 创建副本以避免并发问题
	result := make(map[string][]*ServiceInstance)
	for serviceName, instances := range s.registry.services {
		result[serviceName] = make([]*ServiceInstance, len(instances))
		copy(result[serviceName], instances)
	}

	return result
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
