package discovery

import (
	"fmt"
	"math/rand"
	"time"
)

// Discovery 服务发现接口
type Discovery interface {
	// GetServiceInstance 获取服务实例（负载均衡）
	GetServiceInstance(serviceName string) (*ServiceInstance, error)

	// GetAllServiceInstances 获取所有服务实例
	GetAllServiceInstances(serviceName string) ([]*ServiceInstance, error)

	// GetAllServices 获取所有服务
	GetAllServices() map[string][]*ServiceInstance

	// Stop 停止服务发现
	Stop()
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	// Select 选择一个服务实例
	Select(instances []*ServiceInstance) (*ServiceInstance, error)
}

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	counters map[string]int
}

// NewRoundRobinLoadBalancer 创建轮询负载均衡器
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		counters: make(map[string]int),
	}
}

// Select 轮询选择服务实例
func (lb *RoundRobinLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances available")
	}

	// 过滤健康的实例
	var healthyInstances []*ServiceInstance
	for _, instance := range instances {
		if instance.Healthy {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return nil, fmt.Errorf("no healthy instances available")
	}

	// 轮询选择
	serviceName := healthyInstances[0].ServiceName
	counter := lb.counters[serviceName]
	selected := healthyInstances[counter%len(healthyInstances)]
	lb.counters[serviceName] = counter + 1

	return selected, nil
}

// RandomLoadBalancer 随机负载均衡器
type RandomLoadBalancer struct{}

// NewRandomLoadBalancer 创建随机负载均衡器
func NewRandomLoadBalancer() *RandomLoadBalancer {
	return &RandomLoadBalancer{}
}

// Select 随机选择服务实例
func (lb *RandomLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances available")
	}

	// 过滤健康的实例
	var healthyInstances []*ServiceInstance
	for _, instance := range instances {
		if instance.Healthy {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return nil, fmt.Errorf("no healthy instances available")
	}

	// 随机选择
	rand.Seed(time.Now().UnixNano())
	selected := healthyInstances[rand.Intn(len(healthyInstances))]

	return selected, nil
}
