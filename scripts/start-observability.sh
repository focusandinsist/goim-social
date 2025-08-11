#!/bin/bash

# GoIM Social 可观测性栈启动脚本

set -e

echo "🚀 启动 GoIM Social 可观测性栈..."

# 检查Docker是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动 Docker"
    exit 1
fi

# 创建日志目录
mkdir -p logs

# 启动可观测性栈
echo "📊 启动 Prometheus + Grafana + Loki + Jaeger..."
docker-compose -f docker-compose.observability.yml up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 10

# 检查服务状态
echo "🔍 检查服务状态..."

services=("prometheus:9090" "grafana:3000" "loki:3100" "jaeger:16686")
for service in "${services[@]}"; do
    name=$(echo $service | cut -d: -f1)
    port=$(echo $service | cut -d: -f2)
    
    if curl -s "http://localhost:$port" > /dev/null; then
        echo "✅ $name 运行正常 (http://localhost:$port)"
    else
        echo "❌ $name 启动失败"
    fi
done

echo ""
echo "🎉 可观测性栈启动完成！"
echo ""
echo "📊 访问地址："
echo "  • Grafana:    http://localhost:3000 (admin/admin123)"
echo "  • Prometheus: http://localhost:9090"
echo "  • Jaeger:     http://localhost:16686"

echo ""
echo "🔧 接下来："
echo "  1. 启动你的微服务"
echo "  2. 在 Grafana 中查看指标和链路追踪"
echo "  3. 日志会自动输出到控制台（高性能JSON格式）"
echo "  4. 日志包含 trace_id、user_id 等业务信息"
echo ""
echo "📝 查看日志: docker-compose -f docker-compose.observability.yml logs -f"
echo "🛑 停止服务: docker-compose -f docker-compose.observability.yml down"
