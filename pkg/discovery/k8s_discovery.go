package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"goim-social/pkg/logger"
)

// ServiceInstance 服务实例信息
type ServiceInstance struct {
	ServiceName string            `json:"service_name"`
	Host        string            `json:"host"`
	Port        int32             `json:"port"`
	GRPCPort    int32             `json:"grpc_port"`
	Healthy     bool              `json:"healthy"`
	Metadata    map[string]string `json:"metadata"`
	LastCheck   int64             `json:"last_check"`
}

// K8sDiscovery Kubernetes 服务发现客户端
type K8sDiscovery struct {
	clientset   *kubernetes.Clientset
	namespace   string
	logger      logger.Logger
	services    map[string][]*ServiceInstance
	mutex       sync.RWMutex
	stopCh      chan struct{}
	watchers    map[string]watch.Interface
	watchersMux sync.RWMutex
}

// NewK8sDiscovery 创建 Kubernetes 服务发现客户端
func NewK8sDiscovery(namespace string, logger logger.Logger) (*K8sDiscovery, error) {
	// 获取 Kubernetes 配置
	config, err := getK8sConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s config: %w", err)
	}

	// 创建 Kubernetes 客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	discovery := &K8sDiscovery{
		clientset: clientset,
		namespace: namespace,
		logger:    logger,
		services:  make(map[string][]*ServiceInstance),
		stopCh:    make(chan struct{}),
		watchers:  make(map[string]watch.Interface),
	}

	// 启动服务发现
	if err := discovery.start(); err != nil {
		return nil, fmt.Errorf("failed to start service discovery: %w", err)
	}

	return discovery, nil
}

// getK8sConfig 获取 Kubernetes 配置
func getK8sConfig() (*rest.Config, error) {
	// 首先尝试集群内配置
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	// 如果不在集群内，尝试使用 kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home := homeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	return nil, fmt.Errorf("unable to load k8s config")
}

// homeDir 获取用户主目录
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}

// start 启动服务发现
func (k *K8sDiscovery) start() error {
	ctx := context.Background()

	// 获取所有服务
	services, err := k.clientset.CoreV1().Services(k.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=instant-messaging",
	})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	// 初始化服务实例
	for _, svc := range services.Items {
		k.updateServiceInstances(&svc)
	}

	// 启动服务监听
	go k.watchServices()

	k.logger.Info(context.Background(), "Kubernetes service discovery started",
		logger.F("namespace", k.namespace),
		logger.F("services_count", len(services.Items)))

	return nil
}

// watchServices 监听服务变化
func (k *K8sDiscovery) watchServices() {
	for {
		select {
		case <-k.stopCh:
			return
		default:
			k.startServiceWatcher()
			time.Sleep(5 * time.Second) // 重连间隔
		}
	}
}

// startServiceWatcher 启动服务监听器
func (k *K8sDiscovery) startServiceWatcher() {
	ctx := context.Background()
	
	watcher, err := k.clientset.CoreV1().Services(k.namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/part-of=instant-messaging",
	})
	if err != nil {
		k.logger.Error(ctx, "Failed to start service watcher", logger.F("error", err.Error()))
		return
	}

	k.watchersMux.Lock()
	k.watchers["services"] = watcher
	k.watchersMux.Unlock()

	defer func() {
		k.watchersMux.Lock()
		delete(k.watchers, "services")
		k.watchersMux.Unlock()
		watcher.Stop()
	}()

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Added, watch.Modified:
			if svc, ok := event.Object.(*v1.Service); ok {
				k.updateServiceInstances(svc)
			}
		case watch.Deleted:
			if svc, ok := event.Object.(*v1.Service); ok {
				k.removeServiceInstances(svc)
			}
		case watch.Error:
			k.logger.Error(ctx, "Service watcher error", logger.F("event", event))
			return
		}
	}
}

// updateServiceInstances 更新服务实例
func (k *K8sDiscovery) updateServiceInstances(svc *v1.Service) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	serviceName := svc.Name
	var instances []*ServiceInstance

	// 获取服务的端点
	ctx := context.Background()
	endpoints, err := k.clientset.CoreV1().Endpoints(k.namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		k.logger.Error(ctx, "Failed to get endpoints", 
			logger.F("service", serviceName), 
			logger.F("error", err.Error()))
		return
	}

	// 解析端点信息
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			var httpPort, grpcPort int32

			// 查找 HTTP 和 gRPC 端口
			for _, port := range subset.Ports {
				switch port.Name {
				case "http":
					httpPort = port.Port
				case "grpc":
					grpcPort = port.Port
				}
			}

			if httpPort > 0 {
				instance := &ServiceInstance{
					ServiceName: serviceName,
					Host:        addr.IP,
					Port:        httpPort,
					GRPCPort:    grpcPort,
					Healthy:     true,
					Metadata:    svc.Labels,
					LastCheck:   time.Now().Unix(),
				}
				instances = append(instances, instance)
			}
		}
	}

	k.services[serviceName] = instances
	
	k.logger.Info(ctx, "Updated service instances",
		logger.F("service", serviceName),
		logger.F("instances_count", len(instances)))
}

// removeServiceInstances 移除服务实例
func (k *K8sDiscovery) removeServiceInstances(svc *v1.Service) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	serviceName := svc.Name
	delete(k.services, serviceName)

	k.logger.Info(context.Background(), "Removed service instances",
		logger.F("service", serviceName))
}

// GetServiceInstance 获取服务实例（负载均衡）
func (k *K8sDiscovery) GetServiceInstance(serviceName string) (*ServiceInstance, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	instances, exists := k.services[serviceName]
	if !exists || len(instances) == 0 {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// 简单的轮询负载均衡（可以优化为更复杂的算法）
	for _, instance := range instances {
		if instance.Healthy {
			return instance, nil
		}
	}

	return nil, fmt.Errorf("no healthy instance found for service %s", serviceName)
}

// GetAllServiceInstances 获取所有服务实例
func (k *K8sDiscovery) GetAllServiceInstances(serviceName string) ([]*ServiceInstance, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	instances, exists := k.services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// 返回健康的实例
	var healthyInstances []*ServiceInstance
	for _, instance := range instances {
		if instance.Healthy {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	return healthyInstances, nil
}

// GetAllServices 获取所有服务
func (k *K8sDiscovery) GetAllServices() map[string][]*ServiceInstance {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	// 创建副本以避免并发问题
	result := make(map[string][]*ServiceInstance)
	for name, instances := range k.services {
		result[name] = make([]*ServiceInstance, len(instances))
		copy(result[name], instances)
	}

	return result
}

// Stop 停止服务发现
func (k *K8sDiscovery) Stop() {
	close(k.stopCh)

	k.watchersMux.Lock()
	defer k.watchersMux.Unlock()

	for _, watcher := range k.watchers {
		watcher.Stop()
	}

	k.logger.Info(context.Background(), "Kubernetes service discovery stopped")
}
