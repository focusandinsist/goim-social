#!/bin/bash

# Kubernetes 部署脚本
# 使用 Kubernetes 原生服务发现

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 kubectl 是否可用
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl 未安装或不在 PATH 中"
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "无法连接到 Kubernetes 集群"
        exit 1
    fi
    
    log_success "Kubernetes 集群连接正常"
}

# 创建命名空间
create_namespace() {
    log_info "创建命名空间 im-system..."
    kubectl create namespace im-system --dry-run=client -o yaml | kubectl apply -f -
    log_success "命名空间 im-system 已创建或已存在"
}

# 部署基础配置
deploy_base() {
    log_info "部署基础配置..."
    
    # 部署 RBAC
    log_info "部署 RBAC 配置..."
    kubectl apply -f k8s/base/rbac.yaml
    
    # 部署 ConfigMap
    log_info "部署 ConfigMap..."
    kubectl apply -f k8s/base/configmap.yaml
    
    # 部署 Secrets（如果存在）
    if [ -f "k8s/base/secrets.yaml" ]; then
        log_info "部署 Secrets..."
        kubectl apply -f k8s/base/secrets.yaml
    else
        log_warning "未找到 secrets.yaml，请确保手动创建必要的 Secret"
    fi
    
    log_success "基础配置部署完成"
}

# 部署服务
deploy_services() {
    log_info "部署微服务..."
    
    # 部署顺序：基础服务 -> 业务服务 -> 网关服务
    services=(
        "user-service"
        "social-service"
        "content-service"
        "message-service"
    )
    
    for service in "${services[@]}"; do
        if [ -f "k8s/services/${service}.yaml" ]; then
            log_info "部署 ${service}..."
            kubectl apply -f "k8s/services/${service}.yaml"
            log_success "${service} 部署完成"
        else
            log_warning "未找到 ${service}.yaml"
        fi
    done
    
    # 部署网关服务（如果存在）
    if [ -f "k8s/services/api-gateway-service.yaml" ]; then
        log_info "部署 API Gateway..."
        kubectl apply -f k8s/services/api-gateway-service.yaml
        log_success "API Gateway 部署完成"
    fi
    
    if [ -f "k8s/services/im-gateway-service.yaml" ]; then
        log_info "部署 IM Gateway..."
        kubectl apply -f k8s/services/im-gateway-service.yaml
        log_success "IM Gateway 部署完成"
    fi
}

# 等待服务就绪
wait_for_services() {
    log_info "等待服务就绪..."
    
    services=(
        "user-service"
        "social-service"
        "content-service"
        "message-service"
    )
    
    for service in "${services[@]}"; do
        log_info "等待 ${service} 就绪..."
        kubectl wait --for=condition=available --timeout=300s deployment/${service} -n im-system
        log_success "${service} 已就绪"
    done
}

# 检查服务状态
check_services() {
    log_info "检查服务状态..."
    
    echo ""
    log_info "=== Pods 状态 ==="
    kubectl get pods -n im-system -o wide
    
    echo ""
    log_info "=== Services 状态 ==="
    kubectl get services -n im-system
    
    echo ""
    log_info "=== Deployments 状态 ==="
    kubectl get deployments -n im-system
    
    echo ""
    log_info "=== 服务发现测试 ==="
    kubectl get endpoints -n im-system
}

# 显示访问信息
show_access_info() {
    log_info "获取访问信息..."
    
    # 获取 API Gateway 访问地址
    api_gateway_ip=$(kubectl get service api-gateway-service -n im-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    api_gateway_port=$(kubectl get service api-gateway-service -n im-system -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || echo "")
    
    if [ -n "$api_gateway_ip" ] && [ -n "$api_gateway_port" ]; then
        echo ""
        log_success "=== 访问信息 ==="
        echo "API Gateway: http://${api_gateway_ip}:${api_gateway_port}"
        echo "健康检查: http://${api_gateway_ip}:${api_gateway_port}/health"
        echo "服务发现: http://${api_gateway_ip}:${api_gateway_port}/api/v1/gateway/services"
    else
        echo ""
        log_info "=== 本地访问 ==="
        echo "使用 kubectl port-forward 进行本地访问："
        echo "kubectl port-forward -n im-system service/api-gateway-service 8080:21007"
        echo "然后访问: http://localhost:8080"
    fi
}

# 清理函数
cleanup() {
    log_warning "正在清理资源..."
    kubectl delete namespace im-system --ignore-not-found=true
    log_success "清理完成"
}

# 主函数
main() {
    case "${1:-deploy}" in
        "deploy")
            log_info "开始部署 IM 系统到 Kubernetes..."
            check_kubectl
            create_namespace
            deploy_base
            deploy_services
            wait_for_services
            check_services
            show_access_info
            log_success "部署完成！"
            ;;
        "status")
            log_info "检查服务状态..."
            check_kubectl
            check_services
            show_access_info
            ;;
        "cleanup")
            log_warning "这将删除整个 im-system 命名空间及其所有资源！"
            read -p "确认继续？(y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                cleanup
            else
                log_info "取消清理操作"
            fi
            ;;
        "help"|"-h"|"--help")
            echo "用法: $0 [命令]"
            echo ""
            echo "命令:"
            echo "  deploy   部署所有服务（默认）"
            echo "  status   检查服务状态"
            echo "  cleanup  清理所有资源"
            echo "  help     显示帮助信息"
            echo ""
            echo "示例:"
            echo "  $0 deploy   # 部署系统"
            echo "  $0 status   # 检查状态"
            echo "  $0 cleanup  # 清理资源"
            ;;
        *)
            log_error "未知命令: $1"
            echo "使用 '$0 help' 查看帮助信息"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
