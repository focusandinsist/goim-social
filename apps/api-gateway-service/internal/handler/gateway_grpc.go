package handler

import (
	"context"
)

// 这个文件用于未来扩展API网关特有的gRPC方法
// 目前为空，但提供了扩展的结构

// HealthCheck 健康检查（示例方法，未来可能需要）
func (h *GRPCHandler) healthCheck(ctx context.Context) error {
	h.log.Info(ctx, "gRPC health check")
	// 这里可以添加健康检查逻辑
	return nil
}

// GetServiceStatus 获取服务状态（示例方法，未来可能需要）
func (h *GRPCHandler) getServiceStatus(ctx context.Context) map[string]interface{} {
	h.log.Info(ctx, "gRPC get service status")

	// 获取所有注册的服务
	services := h.svc.GetAllServices()

	status := make(map[string]interface{})
	for serviceName, instances := range services {
		status[serviceName] = map[string]interface{}{
			"instance_count": len(instances),
			"instances":      instances,
		}
	}

	return status
}
