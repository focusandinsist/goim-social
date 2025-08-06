#!/bin/bash

# Kubernetes健康检查脚本
# 使用方法: ./health-check.sh [namespace]

set -e

# 默认配置
NAMESPACE=${1:-"im-system"}
MONITORING_NAMESPACE="im-monitoring"

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

# 检查kubectl连接
check_kubectl() {
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    log_success "Kubernetes cluster connection verified"
}

# 检查命名空间
check_namespace() {
    if ! kubectl get namespace "${NAMESPACE}" &> /dev/null; then
        log_error "Namespace '${NAMESPACE}' does not exist"
        exit 1
    fi
    log_success "Namespace '${NAMESPACE}' exists"
}

# 检查Pod状态
check_pods() {
    log_info "Checking Pod status in namespace '${NAMESPACE}'..."
    
    local total_pods=$(kubectl get pods -n "${NAMESPACE}" --no-headers 2>/dev/null | wc -l)
    local running_pods=$(kubectl get pods -n "${NAMESPACE}" --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
    local pending_pods=$(kubectl get pods -n "${NAMESPACE}" --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l)
    local failed_pods=$(kubectl get pods -n "${NAMESPACE}" --field-selector=status.phase=Failed --no-headers 2>/dev/null | wc -l)
    
    echo "----------------------------------------"
    echo "Pod Status Summary:"
    echo "  Total Pods: ${total_pods}"
    echo "  Running: ${running_pods}"
    echo "  Pending: ${pending_pods}"
    echo "  Failed: ${failed_pods}"
    echo "----------------------------------------"
    
    if [ "$failed_pods" -gt 0 ]; then
        log_error "Found ${failed_pods} failed pods:"
        kubectl get pods -n "${NAMESPACE}" --field-selector=status.phase=Failed
        echo ""
    fi
    
    if [ "$pending_pods" -gt 0 ]; then
        log_warning "Found ${pending_pods} pending pods:"
        kubectl get pods -n "${NAMESPACE}" --field-selector=status.phase=Pending
        echo ""
    fi
    
    # 显示所有Pod状态
    kubectl get pods -n "${NAMESPACE}" -o wide
    echo ""
    
    if [ "$running_pods" -eq "$total_pods" ] && [ "$total_pods" -gt 0 ]; then
        log_success "All pods are running"
    elif [ "$total_pods" -eq 0 ]; then
        log_warning "No pods found in namespace"
    else
        log_warning "Not all pods are running"
    fi
}

# 检查服务状态
check_services() {
    log_info "Checking Service status in namespace '${NAMESPACE}'..."
    
    local services=$(kubectl get svc -n "${NAMESPACE}" --no-headers 2>/dev/null | wc -l)
    
    if [ "$services" -gt 0 ]; then
        kubectl get svc -n "${NAMESPACE}" -o wide
        log_success "Found ${services} services"
    else
        log_warning "No services found in namespace"
    fi
    echo ""
}

# 检查Ingress状态
check_ingress() {
    log_info "Checking Ingress status in namespace '${NAMESPACE}'..."
    
    local ingresses=$(kubectl get ingress -n "${NAMESPACE}" --no-headers 2>/dev/null | wc -l)
    
    if [ "$ingresses" -gt 0 ]; then
        kubectl get ingress -n "${NAMESPACE}" -o wide
        log_success "Found ${ingresses} ingresses"
    else
        log_warning "No ingresses found in namespace"
    fi
    echo ""
}

# 检查PVC状态
check_pvc() {
    log_info "Checking PVC status in namespace '${NAMESPACE}'..."
    
    local pvcs=$(kubectl get pvc -n "${NAMESPACE}" --no-headers 2>/dev/null | wc -l)
    local bound_pvcs=$(kubectl get pvc -n "${NAMESPACE}" --field-selector=status.phase=Bound --no-headers 2>/dev/null | wc -l)
    
    if [ "$pvcs" -gt 0 ]; then
        kubectl get pvc -n "${NAMESPACE}" -o wide
        echo ""
        echo "PVC Status: ${bound_pvcs}/${pvcs} bound"
        
        if [ "$bound_pvcs" -eq "$pvcs" ]; then
            log_success "All PVCs are bound"
        else
            log_warning "Not all PVCs are bound"
        fi
    else
        log_warning "No PVCs found in namespace"
    fi
    echo ""
}

# 检查资源使用情况
check_resource_usage() {
    log_info "Checking resource usage in namespace '${NAMESPACE}'..."
    
    # 检查Pod资源使用
    if kubectl top pods -n "${NAMESPACE}" &> /dev/null; then
        kubectl top pods -n "${NAMESPACE}"
        echo ""
    else
        log_warning "Metrics server not available, cannot show resource usage"
    fi
    
    # 检查节点资源使用
    if kubectl top nodes &> /dev/null; then
        log_info "Node resource usage:"
        kubectl top nodes
        echo ""
    fi
}

# 检查事件
check_events() {
    log_info "Checking recent events in namespace '${NAMESPACE}'..."
    
    # 获取最近的事件
    local warning_events=$(kubectl get events -n "${NAMESPACE}" --field-selector type=Warning --no-headers 2>/dev/null | wc -l)
    local normal_events=$(kubectl get events -n "${NAMESPACE}" --field-selector type=Normal --no-headers 2>/dev/null | wc -l)
    
    echo "Recent Events: ${normal_events} Normal, ${warning_events} Warning"
    
    if [ "$warning_events" -gt 0 ]; then
        log_warning "Warning events found:"
        kubectl get events -n "${NAMESPACE}" --field-selector type=Warning --sort-by='.lastTimestamp' | tail -10
        echo ""
    fi
    
    # 显示最近的事件
    log_info "Last 5 events:"
    kubectl get events -n "${NAMESPACE}" --sort-by='.lastTimestamp' | tail -5
    echo ""
}

# 检查特定服务健康状态
check_service_health() {
    log_info "Checking service health endpoints..."
    
    # 定义服务健康检查端点
    local services=(
        "user-service:80:/health"
        "social-service:80:/health"     # 合并了 friend-service 和 group-service
        "message-service:80:/health"
        "logic-service:80:/health"
        "im-gateway-service:80:/health"
        "api-gateway-service:80:/health"
        "content-service:80:/health"
        "interaction-service:80:/health"
        "comment-service:80:/health"
        "history-service:80:/health"
    )
    
    for service_info in "${services[@]}"; do
        IFS=':' read -r service_name port path <<< "$service_info"
        
        # 检查服务是否存在
        if kubectl get svc "${service_name}" -n "${NAMESPACE}" &> /dev/null; then
            # 使用kubectl port-forward进行健康检查
            local pod=$(kubectl get pods -n "${NAMESPACE}" -l app="${service_name}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
            
            if [ -n "$pod" ]; then
                # 直接在Pod内执行健康检查
                if kubectl exec -n "${NAMESPACE}" "$pod" -- wget --spider --timeout=5 "http://localhost:${port#80}${path}" &> /dev/null; then
                    log_success "${service_name} health check passed"
                else
                    log_warning "${service_name} health check failed"
                fi
            else
                log_warning "${service_name} no pods found"
            fi
        else
            log_warning "${service_name} service not found"
        fi
    done
    echo ""
}

# 检查数据库连接
check_database_connections() {
    log_info "Checking database connections..."
    
    # 检查PostgreSQL
    local pg_pod=$(kubectl get pods -n "${NAMESPACE}" -l app=postgresql -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$pg_pod" ]; then
        if kubectl exec -n "${NAMESPACE}" "$pg_pod" -- pg_isready -U postgres &> /dev/null; then
            log_success "PostgreSQL is ready"
        else
            log_error "PostgreSQL is not ready"
        fi
    else
        log_warning "PostgreSQL pod not found"
    fi
    
    # 检查MongoDB
    local mongo_pod=$(kubectl get pods -n "${NAMESPACE}" -l app=mongodb -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$mongo_pod" ]; then
        if kubectl exec -n "${NAMESPACE}" "$mongo_pod" -- mongosh --eval "db.adminCommand('ping')" &> /dev/null; then
            log_success "MongoDB is ready"
        else
            log_error "MongoDB is not ready"
        fi
    else
        log_warning "MongoDB pod not found"
    fi
    
    # 检查Redis
    local redis_pod=$(kubectl get pods -n "${NAMESPACE}" -l app=redis -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$redis_pod" ]; then
        if kubectl exec -n "${NAMESPACE}" "$redis_pod" -- redis-cli ping &> /dev/null; then
            log_success "Redis is ready"
        else
            log_error "Redis is not ready"
        fi
    else
        log_warning "Redis pod not found"
    fi
    
    # 检查Kafka
    local kafka_pod=$(kubectl get pods -n "${NAMESPACE}" -l app=kafka -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$kafka_pod" ]; then
        if kubectl exec -n "${NAMESPACE}" "$kafka_pod" -- kafka-broker-api-versions --bootstrap-server localhost:9092 &> /dev/null; then
            log_success "Kafka is ready"
        else
            log_error "Kafka is not ready"
        fi
    else
        log_warning "Kafka pod not found"
    fi
    echo ""
}

# 生成健康报告
generate_report() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local report_file="/tmp/k8s-health-report-$(date +%Y%m%d-%H%M%S).txt"
    
    log_info "Generating health report..."
    
    {
        echo "Kubernetes Health Check Report"
        echo "Generated: ${timestamp}"
        echo "Namespace: ${NAMESPACE}"
        echo "========================================"
        echo ""
        
        echo "Pod Status:"
        kubectl get pods -n "${NAMESPACE}" -o wide
        echo ""
        
        echo "Service Status:"
        kubectl get svc -n "${NAMESPACE}" -o wide
        echo ""
        
        echo "PVC Status:"
        kubectl get pvc -n "${NAMESPACE}" -o wide
        echo ""
        
        echo "Recent Events:"
        kubectl get events -n "${NAMESPACE}" --sort-by='.lastTimestamp' | tail -10
        echo ""
        
        if kubectl top pods -n "${NAMESPACE}" &> /dev/null; then
            echo "Resource Usage:"
            kubectl top pods -n "${NAMESPACE}"
            echo ""
        fi
        
    } > "$report_file"
    
    log_success "Health report saved to: $report_file"
}

# 主函数
main() {
    log_info "Starting Kubernetes health check..."
    log_info "Namespace: ${NAMESPACE}"
    echo ""
    
    check_kubectl
    check_namespace
    
    echo "========================================"
    check_pods
    check_services
    check_ingress
    check_pvc
    check_resource_usage
    check_events
    check_service_health
    check_database_connections
    
    generate_report
    
    log_success "Health check completed!"
}

# 显示帮助信息
show_help() {
    echo "Usage: $0 [namespace]"
    echo ""
    echo "Arguments:"
    echo "  namespace   Kubernetes namespace to check (default: im-system)"
    echo ""
    echo "Examples:"
    echo "  $0                    # Check im-system namespace"
    echo "  $0 im-monitoring      # Check im-monitoring namespace"
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
