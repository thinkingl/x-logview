#!/bin/bash

# x-logview 启动脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}       x-logview 启动脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查依赖
check_deps() {
    echo -e "${YELLOW}检查依赖...${NC}"

    if ! command -v go &> /dev/null; then
        echo -e "${RED}错误: 未找到 go${NC}"
        echo "请先安装: brew install go"
        exit 1
    fi

    if ! command -v node &> /dev/null; then
        echo -e "${RED}错误: 未找到 node${NC}"
        echo "请先安装: brew install node"
        exit 1
    fi

    echo -e "${GREEN}依赖检查通过${NC}"
    echo ""
}

# 构建后端
build_backend() {
    echo -e "${YELLOW}构建后端...${NC}"

    if [ ! -f "bin/x-logview-server" ]; then
        go build -o bin/x-logview-server ./cmd/server
        echo -e "${GREEN}后端构建完成${NC}"
    else
        echo -e "${GREEN}后端已存在，跳过构建${NC}"
    fi
    echo ""
}

# 安装前端依赖
install_frontend() {
    echo -e "${YELLOW}检查前端依赖...${NC}"

    if [ ! -d "client/node_modules" ]; then
        echo "安装前端依赖..."
        cd client && npm install && cd ..
        echo -e "${GREEN}前端依赖安装完成${NC}"
    else
        echo -e "${GREEN}前端依赖已存在${NC}"
    fi
    echo ""
}

# 启动后端
start_backend() {
    echo -e "${YELLOW}启动后端服务...${NC}"
    ./bin/x-logview-server -port 8090 &
    BACKEND_PID=$!
    echo -e "${GREEN}后端已启动 (PID: $BACKEND_PID)${NC}"
    echo ""

    # 等待后端启动
    sleep 2

    # 检查后端是否正常
    if curl -s http://localhost:8090/health > /dev/null 2>&1; then
        echo -e "${GREEN}后端健康检查通过${NC}"
    else
        echo -e "${RED}后端启动失败${NC}"
        exit 1
    fi
    echo ""
}

# 启动前端
start_frontend() {
    echo -e "${YELLOW}启动前端开发服务器...${NC}"
    cd client && npm run dev &
    FRONTEND_PID=$!
    cd ..
    echo -e "${GREEN}前端已启动 (PID: $FRONTEND_PID)${NC}"
    echo ""
}

# 显示访问信息
show_info() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}       服务已启动${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "前端地址: ${YELLOW}http://localhost:5173${NC}"
    echo -e "后端地址: ${YELLOW}http://localhost:8090${NC}"
    echo -e "WebSocket: ${YELLOW}ws://localhost:8090/ws${NC}"
    echo ""
    echo -e "按 ${RED}Ctrl+C${NC} 停止所有服务"
    echo ""
}

# 清理函数
cleanup() {
    echo ""
    echo -e "${YELLOW}正在停止服务...${NC}"

    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
        echo -e "${GREEN}后端已停止${NC}"
    fi

    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
        echo -e "${GREEN}前端已停止${NC}"
    fi

    echo -e "${GREEN}所有服务已停止${NC}"
    exit 0
}

# 设置信号处理
trap cleanup SIGINT SIGTERM

# 主流程
check_deps
build_backend
install_frontend
start_backend
start_frontend
show_info

# 等待用户中断
wait
