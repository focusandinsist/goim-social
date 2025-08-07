#!/bin/bash

# 社交服务快速启动脚本

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}    启动 Social Service (端口: 22001)${NC}"
echo -e "${BLUE}========================================${NC}"

# 检查端口是否被占用
if lsof -Pi :22001 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}警告: 端口 22001 已被占用，正在尝试停止...${NC}"
    pkill -f "social-service" || true
    sleep 2
fi

# 进入服务目录
cd "$(dirname "$0")/.."

echo -e "${GREEN}1. 检查配置文件...${NC}"
if [ ! -f "config/config.yaml" ]; then
    echo "错误: 配置文件不存在"
    exit 1
fi

echo -e "${GREEN}2. 构建服务...${NC}"
go build -o social-service main.go

echo -e "${GREEN}3. 启动服务...${NC}"
echo "服务地址: http://localhost:22001"
echo "健康检查: http://localhost:22001/health"
echo "按 Ctrl+C 停止服务"
echo ""

# 启动服务
./social-service
