#!/bin/bash

# Kubernetes一键部署脚本
# 使用方法: ./deploy.sh [environment] [action]

set -e

# 默认配置
ENVIRONMENT=${1:-"dev"}
ACTION=${2:-"deploy"}
NAMESPACE="im-system"
MONITORING_NAMESPACE="im-monitoring"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K8S_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# 颜色输出
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

# 检查kubectl是否可用
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    log_info "Kubernetes cluster connection verified"
}

# 检查必要的工具
check_dependencies() {
    local missing_tools=()
    
    for tool in kubectl helm; do
        if ! command -v $tool &> /dev/null; then
            missing_tools+=($tool)
        fi
    done
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Please install the missing tools and try again"
        exit 1
    fi
    
    log_info "All required tools are available"
}

# 等待资源就绪
wait_for_resource() {
    local resource_type=$1
    local resource_name=$2
    local namespace=$3
    local timeout=${4:-300}
    
    log_info "Waiting for ${resource_type}/${resource_name} to be ready..."
    
    if kubectl wait --for=condition=ready ${resource_type}/${resource_name} -n ${namespace} --timeout=${timeout}s; then
        log_success "${resource_type}/${resource_name} is ready"
        return 0
    else
        log_error "${resource_type}/${resource_name} failed to become ready within ${timeout}s"
        return 1
    fi
}

# 部署基础配置
deploy_base() {
    log_info "Deploying base configuration..."
    
    # 创建命名空间
    kubectl apply -f "${K8S_DIR}/base/namespace.yaml"
    
    # 部署ConfigMap和Secret
    kubectl apply -f "${K8S_DIR}/base/configmap.yaml"
    kubectl apply -f "${K8S_DIR}/base/secrets.yaml"
    
    # 部署存储配置
    kubectl apply -f "${K8S_DIR}/base/storage.yaml"
    
    log_success "Base configuration deployed"
}

# 部署基础设施
deploy_infrastructure() {
    log_info "Deploying infrastructure services..."
    
    # 部署PostgreSQL
    log_info "Deploying PostgreSQL..."
    kubectl apply -f "${K8S_DIR}/infrastructure/postgresql.yaml"
    wait_for_resource "pod" "-l app=postgresql" "${NAMESPACE}" 300
    
    # 部署MongoDB
    log_info "Deploying MongoDB..."
    kubectl apply -f "${K8S_DIR}/infrastructure/mongodb.yaml"
    wait_for_resource "pod" "-l app=mongodb" "${NAMESPACE}" 300
    
    # 部署Redis
    log_info "Deploying Redis..."
    kubectl apply -f "${K8S_DIR}/infrastructure/redis.yaml"
    wait_for_resource "pod" "-l app=redis" "${NAMESPACE}" 300
    
    # 部署Kafka (包括Zookeeper)
    log_info "Deploying Kafka..."
    kubectl apply -f "${K8S_DIR}/infrastructure/kafka.yaml"
    wait_for_resource "pod" "-l app=zookeeper" "${NAMESPACE}" 300
    wait_for_resource "pod" "-l app=kafka" "${NAMESPACE}" 300
    
    log_success "Infrastructure services deployed"
}

# 部署微服务
deploy_services() {
    log_info "Deploying microservices..."
    
    # 服务部署顺序 (按依赖关系)
    local services=(
        "user-service"
        "social-service"        # 合并了 friend-service 和 group-service
        "content-service"
        "interaction-service"
        "comment-service"
        "history-service"
        "message-service"
        "logic-service"
        "im-gateway-service"
        "api-gateway-service"
    )
    
    for service in "${services[@]}"; do
        if [ -f "${K8S_DIR}/services/${service}.yaml" ]; then
            log_info "Deploying ${service}..."
            kubectl apply -f "${K8S_DIR}/services/${service}.yaml"
            
            # 等待服务就绪
            sleep 10
            wait_for_resource "pod" "-l app=${service}" "${NAMESPACE}" 180
        else
            log_warning "Service configuration not found: ${service}.yaml"
        fi
    done
    
    log_success "Microservices deployed"
}

# 部署Ingress
deploy_ingress() {
    log_info "Deploying Ingress..."
    
    if [ -f "${K8S_DIR}/ingress/ingress.yaml" ]; then
        kubectl apply -f "${K8S_DIR}/ingress/"
        log_success "Ingress deployed"
    else
        log_warning "Ingress configuration not found"
    fi
}

# 部署监控
deploy_monitoring() {
    log_info "Deploying monitoring stack..."
    
    if [ -d "${K8S_DIR}/monitoring" ]; then
        kubectl apply -f "${K8S_DIR}/monitoring/"
        log_success "Monitoring stack deployed"
    else
        log_warning "Monitoring configuration not found"
    fi
}

# 健康检查
health_check() {
    log_info "Performing health check..."
    
    # 检查所有Pod状态
    log_info "Checking Pod status..."
    kubectl get pods -n "${NAMESPACE}" -o wide
    
    # 检查服务状态
    log_info "Checking Service status..."
    kubectl get svc -n "${NAMESPACE}"
    
    # 检查Ingress状态
    log_info "Checking Ingress status..."
    kubectl get ingress -n "${NAMESPACE}" 2>/dev/null || log_warning "No Ingress found"
    
    # 检查PVC状态
    log_info "Checking PVC status..."
    kubectl get pvc -n "${NAMESPACE}"
    
    # 检查失败的Pod
    local failed_pods=$(kubectl get pods -n "${NAMESPACE}" --field-selector=status.phase!=Running --no-headers 2>/dev/null | wc -l)
    if [ "$failed_pods" -gt 0 ]; then
        log_warning "Found ${failed_pods} non-running pods"
        kubectl get pods -n "${NAMESPACE}" --field-selector=status.phase!=Running
    else
        log_success "All pods are running"
    fi
}

# 清理部署
cleanup() {
    log_warning "Cleaning up deployment..."
    
    # 删除微服务
    if [ -d "${K8S_DIR}/services" ]; then
        kubectl delete -f "${K8S_DIR}/services/" --ignore-not-found=true
    fi
    
    # 删除Ingress
    if [ -d "${K8S_DIR}/ingress" ]; then
        kubectl delete -f "${K8S_DIR}/ingress/" --ignore-not-found=true
    fi
    
    # 删除基础设施
    if [ -d "${K8S_DIR}/infrastructure" ]; then
        kubectl delete -f "${K8S_DIR}/infrastructure/" --ignore-not-found=true
    fi
    
    # 删除基础配置
    if [ -d "${K8S_DIR}/base" ]; then
        kubectl delete -f "${K8S_DIR}/base/" --ignore-not-found=true
    fi
    
    # 删除监控
    if [ -d "${K8S_DIR}/monitoring" ]; then
        kubectl delete -f "${K8S_DIR}/monitoring/" --ignore-not-found=true
    fi
    
    log_success "Cleanup completed"
}

# 显示部署信息
show_info() {
    log_info "Deployment Information:"
    echo "----------------------------------------"
    echo "Environment: ${ENVIRONMENT}"
    echo "Namespace: ${NAMESPACE}"
    echo "Monitoring Namespace: ${MONITORING_NAMESPACE}"
    echo "K8s Directory: ${K8S_DIR}"
    echo "----------------------------------------"
    
    # 显示访问信息
    log_info "Access Information:"
    
    # API Gateway
    local api_gateway_ip=$(kubectl get svc api-gateway-service -n "${NAMESPACE}" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "ClusterIP")
    if [ "$api_gateway_ip" != "ClusterIP" ]; then
        echo "API Gateway: http://${api_gateway_ip}"
    else
        echo "API Gateway: kubectl port-forward svc/api-gateway-service -n ${NAMESPACE} 8080:80"
    fi
    
    # Monitoring
    echo "Grafana: kubectl port-forward svc/grafana -n ${MONITORING_NAMESPACE} 3000:3000"
    echo "Prometheus: kubectl port-forward svc/prometheus -n ${MONITORING_NAMESPACE} 9090:9090"
    
    echo "----------------------------------------"
}

# 主函数
main() {
    log_info "Starting Kubernetes deployment..."
    log_info "Environment: ${ENVIRONMENT}"
    log_info "Action: ${ACTION}"
    
    # 检查依赖
    check_dependencies
    check_kubectl
    
    case "${ACTION}" in
        "deploy")
            deploy_base
            deploy_infrastructure
            deploy_services
            deploy_ingress
            deploy_monitoring
            health_check
            show_info
            ;;
        "infrastructure")
            deploy_base
            deploy_infrastructure
            ;;
        "services")
            deploy_services
            ;;
        "monitoring")
            deploy_monitoring
            ;;
        "cleanup")
            cleanup
            ;;
        "health")
            health_check
            ;;
        "info")
            show_info
            ;;
        *)
            log_error "Unknown action: ${ACTION}"
            show_help
            exit 1
            ;;
    esac
    
    log_success "Deployment completed successfully!"
}

# 显示帮助信息
show_help() {
    echo "Usage: $0 [environment] [action]"
    echo ""
    echo "Environments:"
    echo "  dev         Development environment (default)"
    echo "  staging     Staging environment"
    echo "  prod        Production environment"
    echo ""
    echo "Actions:"
    echo "  deploy      Full deployment (default)"
    echo "  infrastructure  Deploy only infrastructure"
    echo "  services    Deploy only microservices"
    echo "  monitoring  Deploy only monitoring"
    echo "  cleanup     Remove all resources"
    echo "  health      Perform health check"
    echo "  info        Show deployment information"
    echo ""
    echo "Examples:"
    echo "  $0                    # Full deployment in dev environment"
    echo "  $0 prod deploy        # Full deployment in prod environment"
    echo "  $0 dev infrastructure # Deploy only infrastructure"
    echo "  $0 dev cleanup        # Cleanup all resources"
    echo ""
}

# 处理命令行参数
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac
