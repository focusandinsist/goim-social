#!/bin/bash

# 批量构建所有微服务Docker镜像
# 使用方法: ./build-all.sh [registry] [tag]

set -e

# 默认配置
REGISTRY=${1:-"localhost:5000"}
TAG=${2:-"latest"}
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

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

# 检查Docker是否运行
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    log_info "Docker is running"
}

# 创建网络 (如果不存在)
create_network() {
    if ! docker network ls | grep -q "im-network"; then
        log_info "Creating Docker network: im-network"
        docker network create im-network
    fi
}

# 构建单个服务
build_service() {
    local service_name=$1
    local image_name="${REGISTRY}/im-${service_name}:${TAG}"
    
    log_info "Building ${service_name}..."
    
    # 复制Dockerfile模板
    cp "${PROJECT_ROOT}/k8s/docker/Dockerfile.template" "${PROJECT_ROOT}/Dockerfile.${service_name}"
    
    # 构建镜像
    if docker build \
        --build-arg SERVICE_NAME="${service_name}" \
        -t "${image_name}" \
        -f "${PROJECT_ROOT}/Dockerfile.${service_name}" \
        "${PROJECT_ROOT}"; then
        log_success "Built ${image_name}"
        
        # 清理临时Dockerfile
        rm -f "${PROJECT_ROOT}/Dockerfile.${service_name}"
        
        # 推送到registry (如果不是localhost)
        if [[ "${REGISTRY}" != "localhost:5000" ]]; then
            log_info "Pushing ${image_name}..."
            docker push "${image_name}"
            log_success "Pushed ${image_name}"
        fi
        
        return 0
    else
        log_error "Failed to build ${service_name}"
        rm -f "${PROJECT_ROOT}/Dockerfile.${service_name}"
        return 1
    fi
}

# 微服务列表 (按依赖顺序)
SERVICES=(
    "user-service"
    "group-service"
    "friend-service"
    "content-service"
    "interaction-service"
    "comment-service"
    "history-service"
    "message-service"
    "logic-service"
    "im-gateway-service"
    "api-gateway-service"
)

# 主函数
main() {
    log_info "Starting build process..."
    log_info "Registry: ${REGISTRY}"
    log_info "Tag: ${TAG}"
    log_info "Project Root: ${PROJECT_ROOT}"
    
    # 检查环境
    check_docker
    create_network
    
    # 构建计数器
    local success_count=0
    local total_count=${#SERVICES[@]}
    local failed_services=()
    
    # 构建所有服务
    for service in "${SERVICES[@]}"; do
        if build_service "${service}"; then
            ((success_count++))
        else
            failed_services+=("${service}")
        fi
        echo "----------------------------------------"
    done
    
    # 构建结果汇总
    echo ""
    log_info "Build Summary:"
    log_info "Total services: ${total_count}"
    log_success "Successful builds: ${success_count}"
    
    if [ ${#failed_services[@]} -gt 0 ]; then
        log_error "Failed builds: ${#failed_services[@]}"
        log_error "Failed services: ${failed_services[*]}"
        exit 1
    else
        log_success "All services built successfully!"
    fi
    
    # 显示镜像列表
    echo ""
    log_info "Built images:"
    docker images | grep "im-" | grep "${TAG}"
    
    # 生成docker-compose文件
    generate_compose_file
}

# 生成微服务的docker-compose文件
generate_compose_file() {
    local compose_file="${PROJECT_ROOT}/k8s/docker/docker-compose.services.yml"
    
    log_info "Generating docker-compose.services.yml..."
    
    cat > "${compose_file}" << EOF
version: '3.8'

# 微服务容器配置
services:
EOF

    # 为每个服务生成配置
    for service in "${SERVICES[@]}"; do
        local image_name="${REGISTRY}/im-${service}:${TAG}"
        local http_port=$((21000 + $(echo "${SERVICES[@]}" | tr ' ' '\n' | grep -n "^${service}$" | cut -d: -f1)))
        local grpc_port=$((22000 + $(echo "${SERVICES[@]}" | tr ' ' '\n' | grep -n "^${service}$" | cut -d: -f1)))
        
        cat >> "${compose_file}" << EOF
  ${service}:
    image: ${image_name}
    container_name: ${service}
    environment:
      - SERVICE_NAME=${service}
      - HTTP_PORT=${http_port}
      - GRPC_PORT=${grpc_port}
      - POSTGRESQL_HOST=postgresql
      - POSTGRESQL_PORT=5432
      - POSTGRESQL_DB=imdb
      - POSTGRESQL_USER=postgres
      - POSTGRESQL_PASSWORD=postgres123
      - MONGODB_URI=mongodb://admin:admin123@mongodb:27017/imdb?authSource=admin
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=redis123
      - KAFKA_BROKERS=kafka:29092
    ports:
      - "${http_port}:${http_port}"
      - "${grpc_port}:${grpc_port}"
    depends_on:
      - postgresql
      - mongodb
      - redis
      - kafka
    networks:
      - im-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:${http_port}/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

EOF
    done
    
    cat >> "${compose_file}" << EOF
# 使用外部网络
networks:
  im-network:
    external: true
EOF
    
    log_success "Generated ${compose_file}"
}

# 显示帮助信息
show_help() {
    echo "Usage: $0 [registry] [tag]"
    echo ""
    echo "Arguments:"
    echo "  registry    Docker registry (default: localhost:5000)"
    echo "  tag         Image tag (default: latest)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Build with defaults"
    echo "  $0 localhost:5000 v1.0.0            # Build with custom tag"
    echo "  $0 your-registry.com/project latest  # Build and push to registry"
    echo ""
    echo "Environment Variables:"
    echo "  DOCKER_BUILDKIT=1                    # Enable BuildKit for faster builds"
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
