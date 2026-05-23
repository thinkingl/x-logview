#!/bin/bash

# x-logview 完整启动脚本（后端 + 前端 + Electron）

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}       x-logview 完整启动${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 清理函数
cleanup() {
    echo ""
    echo -e "${YELLOW}正在停止所有服务...${NC}"
    [ ! -z "$BACKEND_PID" ] && kill $BACKEND_PID 2>/dev/null
    [ ! -z "$VITE_PID" ] && kill $VITE_PID 2>/dev/null
    [ ! -z "$ELECTRON_PID" ] && kill $ELECTRON_PID 2>/dev/null
    pkill -f "x-logview-server" 2>/dev/null || true
    pkill -f "electron ." 2>/dev/null || true
    pkill -f "vite" 2>/dev/null || true
    echo -e "${GREEN}已停止${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# 1. 构建后端
echo -e "${YELLOW}[1/4] 构建后端...${NC}"
cd "$SCRIPT_DIR"
if [ ! -f "bin/x-logview-server" ]; then
    go build -o bin/x-logview-server ./cmd/server
    echo -e "${GREEN}后端构建完成${NC}"
else
    echo -e "${GREEN}后端已存在${NC}"
fi
echo ""

# 2. 启动后端
echo -e "${YELLOW}[2/4] 启动后端服务...${NC}"
cd "$SCRIPT_DIR"
./bin/x-logview-server -port 8090 > /tmp/x-logview-backend.log 2>&1 &
BACKEND_PID=$!
sleep 2

if curl -s http://localhost:8090/health > /dev/null 2>&1; then
    echo -e "${GREEN}后端已启动: http://localhost:8090${NC}"
else
    echo -e "${RED}后端启动失败，查看日志: /tmp/x-logview-backend.log${NC}"
    cat /tmp/x-logview-backend.log
    exit 1
fi
echo ""

# 3. 启动前端
echo -e "${YELLOW}[3/4] 启动前端...${NC}"
cd "$SCRIPT_DIR/client"
npm run build:main > /dev/null 2>&1
npm run build:preload > /dev/null 2>&1
npm run dev:renderer > /tmp/x-logview-vite.log 2>&1 &
VITE_PID=$!
sleep 3

if curl -s http://localhost:5173 > /dev/null 2>&1; then
    echo -e "${GREEN}Vite 已启动: http://localhost:5173${NC}"
else
    echo -e "${RED}Vite 启动失败，查看日志: /tmp/x-logview-vite.log${NC}"
    cat /tmp/x-logview-vite.log
    exit 1
fi
echo ""

# 4. 启动 Electron
echo -e "${YELLOW}[4/4] 启动 Electron...${NC}"
NODE_ENV=development npx electron . > /tmp/x-logview-electron.log 2>&1 &
ELECTRON_PID=$!
sleep 2
echo -e "${GREEN}Electron 已启动${NC}"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}       所有服务已启动${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "后端:     ${YELLOW}http://localhost:8090${NC}"
echo -e "前端:     ${YELLOW}http://localhost:5173${NC}"
echo -e "Electron: ${YELLOW}PID $ELECTRON_PID${NC}"
echo ""
echo -e "按 ${RED}Ctrl+C${NC} 停止所有服务"
echo ""

wait
