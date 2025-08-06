# 构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的包
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o social-service ./apps/social-service

# 运行阶段
FROM alpine:latest

# 安装ca-certificates和tzdata
RUN apk --no-cache add ca-certificates tzdata

# 设置工作目录
WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/social-service .

# 复制配置文件
COPY --from=builder /app/apps/social-service/config/config.yaml ./config/

# 设置时区
ENV TZ=Asia/Shanghai

# 暴露端口
EXPOSE 22001

# 运行应用
CMD ["./social-service"]
