#!/bin/bash

# Connect服务多实例部署脚本
# 支持启动多个Connect服务实例以实现负载均衡

set -e

# 配置参数
INSTANCES=3
BASE_HTTP_PORT=21003
BASE_GRPC_PORT=22003
SERVICE_NAME="connect"
LOG_DIR="./logs"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 创建日志目录
mkdir -p $LOG_DIR

echo -e "${BLUE}🚀 开始部署Connect服务多实例...${NC}"

# 停止现有实例
stop_instances() {
    echo -e "${YELLOW}🛑 停止现有实例...${NC}"
    
    for i in $(seq 0 $((INSTANCES-1))); do
        HTTP_PORT=$((BASE_HTTP_PORT + i*10))
        GRPC_PORT=$((BASE_GRPC_PORT + i*10))
        
        # 查找并停止进程
        PID=$(lsof -ti:$HTTP_PORT 2>/dev/null || echo "")
        if [ ! -z "$PID" ]; then
            echo "停止实例 $i (PID: $PID, HTTP端口: $HTTP_PORT)"
            kill -TERM $PID 2>/dev/null || true
            sleep 2
            kill -KILL $PID 2>/dev/null || true
        fi
    done
    
    sleep 3
    echo -e "${GREEN}✅ 现有实例已停止${NC}"
}

# 启动实例
start_instances() {
    echo -e "${YELLOW}🚀 启动新实例...${NC}"
    
    for i in $(seq 0 $((INSTANCES-1))); do
        HTTP_PORT=$((BASE_HTTP_PORT + i*10))
        GRPC_PORT=$((BASE_GRPC_PORT + i*10))
        LOG_FILE="$LOG_DIR/${SERVICE_NAME}-instance-$i.log"
        
        echo "启动实例 $i: HTTP=$HTTP_PORT, gRPC=$GRPC_PORT"
        
        # 设置环境变量
        export HTTP_PORT=$HTTP_PORT
        export GRPC_PORT=$GRPC_PORT
        export INSTANCE_ID="connect-instance-$i"
        
        # 启动服务实例
        cd apps/connect
        nohup go run cmd/main.go > "../../$LOG_FILE" 2>&1 &
        cd ../..
        
        # 等待服务启动
        sleep 2
        
        # 检查服务是否启动成功
        if curl -s "http://localhost:$HTTP_PORT/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✅ 实例 $i 启动成功${NC}"
        else
            echo -e "${RED}❌ 实例 $i 启动失败${NC}"
        fi
    done
}

# 检查实例状态
check_instances() {
    echo -e "${BLUE}📊 检查实例状态...${NC}"
    
    for i in $(seq 0 $((INSTANCES-1))); do
        HTTP_PORT=$((BASE_HTTP_PORT + i*10))
        
        if curl -s "http://localhost:$HTTP_PORT/health" > /dev/null 2>&1; then
            echo -e "实例 $i (端口 $HTTP_PORT): ${GREEN}运行中${NC}"
        else
            echo -e "实例 $i (端口 $HTTP_PORT): ${RED}未运行${NC}"
        fi
    done
}

# 显示Redis中的实例信息
show_redis_instances() {
    echo -e "${BLUE}📋 Redis中的实例信息:${NC}"
    
    # 使用redis-cli查看实例列表
    if command -v redis-cli &> /dev/null; then
        echo "活跃实例列表:"
        redis-cli SMEMBERS connect_instances_list 2>/dev/null || echo "无法连接到Redis"
        
        echo -e "\n实例详细信息:"
        INSTANCES_LIST=$(redis-cli SMEMBERS connect_instances_list 2>/dev/null)
        for instance in $INSTANCES_LIST; do
            echo "实例: $instance"
            redis-cli HGETALL "connect_instances:$instance" 2>/dev/null || echo "  无法获取详细信息"
            echo ""
        done
    else
        echo "redis-cli未安装，无法查看Redis信息"
    fi
}

# 主函数
main() {
    case "${1:-start}" in
        "start")
            stop_instances
            start_instances
            sleep 5
            check_instances
            show_redis_instances
            ;;
        "stop")
            stop_instances
            ;;
        "status")
            check_instances
            show_redis_instances
            ;;
        "restart")
            stop_instances
            start_instances
            sleep 5
            check_instances
            ;;
        *)
            echo "用法: $0 {start|stop|status|restart}"
            echo ""
            echo "命令说明:"
            echo "  start   - 启动所有实例"
            echo "  stop    - 停止所有实例"
            echo "  status  - 检查实例状态"
            echo "  restart - 重启所有实例"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"

echo -e "${GREEN}🎉 操作完成！${NC}"
echo ""
echo "实例端口分配:"
for i in $(seq 0 $((INSTANCES-1))); do
    HTTP_PORT=$((BASE_HTTP_PORT + i*10))
    GRPC_PORT=$((BASE_GRPC_PORT + i*10))
    echo "  实例 $i: HTTP=$HTTP_PORT, gRPC=$GRPC_PORT"
done
echo ""
echo "日志文件位置: $LOG_DIR/"
echo "Nginx配置文件: configs/nginx-loadbalancer.conf"
