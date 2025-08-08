#!/bin/bash

# Proto代码生成脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 函数定义
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

# 检查必要的工具
check_prerequisites() {
    log_info "检查必要的工具..."
    
    if ! command -v protoc &> /dev/null; then
        log_error "protoc 未安装或不在 PATH 中"
        log_info "请安装 Protocol Buffers compiler:"
        log_info "  - Windows: https://github.com/protocolbuffers/protobuf/releases"
        log_info "  - macOS: brew install protobuf"
        log_info "  - Ubuntu: sudo apt install protobuf-compiler"
        exit 1
    fi
    
    if ! command -v protoc-gen-go &> /dev/null; then
        log_error "protoc-gen-go 未安装"
        log_info "请安装 protoc-gen-go:"
        log_info "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
        exit 1
    fi
    
    if ! command -v protoc-gen-go-grpc &> /dev/null; then
        log_error "protoc-gen-go-grpc 未安装"
        log_info "请安装 protoc-gen-go-grpc:"
        log_info "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
        exit 1
    fi
    
    log_success "所有必要工具已安装"
}

# 生成proto代码
generate_proto() {
    log_info "生成proto代码..."
    
    # 创建输出目录
    mkdir -p api/rest/pb
    
    # 生成search.proto
    if [ -f "api/rest/search.proto" ]; then
        log_info "生成 search.proto..."
        protoc \
            --proto_path=api/rest \
            --proto_path=/usr/include \
            --proto_path=/usr/local/include \
            --go_out=api/rest/pb \
            --go_opt=paths=source_relative \
            --go-grpc_out=api/rest/pb \
            --go-grpc_opt=paths=source_relative \
            api/rest/search.proto
        
        if [ $? -eq 0 ]; then
            log_success "search.proto 生成成功"
        else
            log_error "search.proto 生成失败"
            exit 1
        fi
    else
        log_warning "search.proto 文件不存在，跳过"
    fi
    
    # 生成其他proto文件（如果存在）
    for proto_file in api/rest/*.proto; do
        if [ -f "$proto_file" ] && [ "$(basename "$proto_file")" != "search.proto" ]; then
            filename=$(basename "$proto_file" .proto)
            log_info "生成 $filename.proto..."
            
            protoc \
                --proto_path=api/rest \
                --go_out=api/rest/pb \
                --go_opt=paths=source_relative \
                --go-grpc_out=api/rest/pb \
                --go-grpc_opt=paths=source_relative \
                "$proto_file"
            
            if [ $? -eq 0 ]; then
                log_success "$filename.proto 生成成功"
            else
                log_error "$filename.proto 生成失败"
                exit 1
            fi
        fi
    done
}

# 更新go.mod
update_go_mod() {
    log_info "更新 go.mod..."
    go mod tidy
    
    if [ $? -eq 0 ]; then
        log_success "go.mod 更新成功"
    else
        log_error "go.mod 更新失败"
        exit 1
    fi
}

# 显示帮助信息
show_help() {
    echo "Proto代码生成脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  generate    生成proto代码（默认）"
    echo "  check       检查工具是否安装"
    echo "  clean       清理生成的代码"
    echo "  help        显示帮助"
    echo ""
}

# 清理生成的代码
clean_generated() {
    log_info "清理生成的proto代码..."
    
    if [ -d "api/rest/pb" ]; then
        rm -rf api/rest/pb
        log_success "清理完成"
    else
        log_info "没有找到生成的代码目录"
    fi
}

# 主函数
main() {
    case "${1:-generate}" in
        "check")
            check_prerequisites
            ;;
        "generate")
            check_prerequisites
            generate_proto
            update_go_mod
            log_success "Proto代码生成完成！"
            ;;
        "clean")
            clean_generated
            ;;
        "help")
            show_help
            ;;
        *)
            log_error "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
