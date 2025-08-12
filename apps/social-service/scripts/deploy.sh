#!/bin/bash

# 社交服务部署脚本
# 用于部署 social-service 并停止原有的 friend-service 和 group-service

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    log_info "依赖检查完成"
}

# 停止原有服务
stop_old_services() {
    log_info "停止原有的 friend-service 和 group-service..."
    
    # 停止 friend-service
    if docker ps | grep -q friend-service; then
        log_info "停止 friend-service..."
        docker stop friend-service || true
        docker rm friend-service || true
    fi
    
    # 停止 group-service
    if docker ps | grep -q group-service; then
        log_info "停止 group-service..."
        docker stop group-service || true
        docker rm group-service || true
    fi
    
    log_info "原有服务已停止"
}

# 备份数据
backup_data() {
    log_info "备份原有数据..."
    
    BACKUP_DIR="./backup/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$BACKUP_DIR"
    
    # 备份 friend-service 数据
    if docker ps -a | grep -q postgres; then
        log_info "备份好友服务数据..."
        docker exec postgres pg_dump -U postgres -d friend_db > "$BACKUP_DIR/friend_service_backup.sql" || true
    fi
    
    # 备份 group-service 数据
    if docker ps -a | grep -q postgres; then
        log_info "备份群组服务数据..."
        docker exec postgres pg_dump -U postgres -d group_db > "$BACKUP_DIR/group_service_backup.sql" || true
    fi
    
    log_info "数据备份完成，备份目录: $BACKUP_DIR"
}

# 构建新服务
build_social_service() {
    log_info "构建 social-service..."
    
    # 进入项目根目录
    cd "$(dirname "$0")/../.."
    
    # 构建 Docker 镜像
    docker build -t social-service:latest -f apps/social-service/Dockerfile .
    
    log_info "social-service 构建完成"
}

# 启动新服务
start_social_service() {
    log_info "启动 social-service..."
    
    cd apps/social-service
    
    # 启动服务
    docker-compose up -d
    
    # 等待服务启动
    log_info "等待服务启动..."
    sleep 30
    
    # 检查服务状态
    if curl -f http://localhost:22001/health > /dev/null 2>&1; then
        log_info "social-service 启动成功"
    else
        log_error "social-service 启动失败"
        docker-compose logs social-service
        exit 1
    fi
}

# 数据迁移
migrate_data() {
    log_info "执行数据迁移..."
    
    # 等待数据库启动
    log_info "等待数据库启动..."
    sleep 10
    
    # 执行迁移脚本
    if [ -f "./scripts/migrate.sql" ]; then
        log_info "执行数据库迁移脚本..."
        docker exec -i social-service_postgres_1 psql -U postgres -d goim_social < ./scripts/migrate.sql
        log_info "数据迁移完成"
    else
        log_warn "未找到迁移脚本，跳过数据迁移"
    fi
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    # 检查服务状态
    if curl -f http://localhost:22001/health > /dev/null 2>&1; then
        log_info "✅ social-service 健康检查通过"
    else
        log_error "❌ social-service 健康检查失败"
        return 1
    fi
    
    # 检查数据库连接
    if docker exec social-service_postgres_1 pg_isready -U postgres > /dev/null 2>&1; then
        log_info "✅ PostgreSQL 连接正常"
    else
        log_error "❌ PostgreSQL 连接失败"
        return 1
    fi
    
    # 检查 Redis 连接
    if docker exec social-service_redis_1 redis-cli ping | grep -q PONG; then
        log_info "✅ Redis 连接正常"
    else
        log_error "❌ Redis 连接失败"
        return 1
    fi
    
    log_info "所有健康检查通过"
}

# 更新配置
update_config() {
    log_info "更新相关配置..."
    
    # 更新 API Gateway 配置（如果存在）
    if [ -f "../../api-gateway-service/config/config.yaml" ]; then
        log_info "更新 API Gateway 配置..."
        # 这里可以添加具体的配置更新逻辑
        log_warn "请手动更新 API Gateway 配置，将 friend-service 和 group-service 的路由指向 social-service"
    fi
    
    # 更新 Logic Service 配置（如果存在）
    if [ -f "../../logic-service/config/config.yaml" ]; then
        log_info "更新 Logic Service 配置..."
        # 这里可以添加具体的配置更新逻辑
        log_warn "请手动更新 Logic Service 配置，将好友和群组相关的 gRPC 调用指向 social-service"
    fi
}

# 清理旧资源
cleanup_old_resources() {
    log_info "清理旧资源..."
    
    # 删除旧的 Docker 镜像
    docker rmi friend-service:latest || true
    docker rmi group-service:latest || true
    
    # 清理未使用的 Docker 资源
    docker system prune -f
    
    log_info "旧资源清理完成"
}

# 显示部署信息
show_deployment_info() {
    log_info "部署完成！"
    echo ""
    echo "=========================================="
    echo "         Social Service 部署信息"
    echo "=========================================="
    echo "服务地址: http://localhost:21002"
    echo "健康检查: http://localhost:21002/health"
    echo "Jaeger 追踪: http://localhost:16686"
    echo "PostgreSQL: localhost:5432"
    echo "Redis: localhost:6379"
    echo "Kafka: localhost:9092"
    echo ""
    echo "API 接口:"
    echo "  好友管理: /api/v1/friend/*"
    echo "  群组管理: /api/v1/group/*"
    echo "  社交验证: /api/v1/social/*"
    echo ""
    echo "管理命令:"
    echo "  查看日志: docker-compose logs -f social-service"
    echo "  重启服务: docker-compose restart social-service"
    echo "  停止服务: docker-compose down"
    echo "=========================================="
}

# 主函数
main() {
    log_info "开始部署 social-service..."
    
    # 检查参数
    SKIP_BACKUP=false
    SKIP_MIGRATION=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-backup)
                SKIP_BACKUP=true
                shift
                ;;
            --skip-migration)
                SKIP_MIGRATION=true
                shift
                ;;
            --help)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --skip-backup     跳过数据备份"
                echo "  --skip-migration  跳过数据迁移"
                echo "  --help           显示帮助信息"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 执行部署步骤
    check_dependencies
    
    if [ "$SKIP_BACKUP" = false ]; then
        backup_data
    else
        log_warn "跳过数据备份"
    fi
    
    stop_old_services
    build_social_service
    start_social_service
    
    if [ "$SKIP_MIGRATION" = false ]; then
        migrate_data
    else
        log_warn "跳过数据迁移"
    fi
    
    health_check
    update_config
    cleanup_old_resources
    show_deployment_info
    
    log_info "social-service 部署完成！"
}

# 错误处理
trap 'log_error "部署过程中发生错误，请检查日志"; exit 1' ERR

# 执行主函数
main "$@"
