#!/bin/bash

# x-logview Electron 启动脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/client"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}       x-logview Electron 启动${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 编译主进程
echo -e "${YELLOW}编译 Electron 主进程...${NC}"
npm run build:main
echo -e "${GREEN}编译完成${NC}"
echo ""

# 启动 Vite 开发服务器
echo -e "${YELLOW}启动 Vite 开发服务器...${NC}"
npm run dev:renderer > /tmp/vite.log 2>&1 &
VITE_PID=$!
sleep 3

if curl -s http://localhost:5173 > /dev/null 2>&1; then
    echo -e "${GREEN}Vite 服务器已启动: http://localhost:5173${NC}"
else
    echo -e "${RED}Vite 启动失败${NC}"
    kill $VITE_PID 2>/dev/null
    exit 1
fi
echo ""

# 启动 Electron
echo -e "${YELLOW}启动 Electron 应用...${NC}"
NODE_ENV=development npx electron . > /tmp/electron.log 2>&1 &
ELECTRON_PID=$!
sleep 2

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}       应用已启动${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "Vite:      ${YELLOW}http://localhost:5173${NC}"
echo -e "Electron:  ${YELLOW}PID $ELECTRON_PID${NC}"
echo ""
echo -e "按 ${RED}Ctrl+C${NC} 停止所有服务"
echo ""

# 清理函数
cleanup() {
    echo ""
    echo -e "${YELLOW}正在停止...${NC}"
    kill $ELECTRON_PID 2>/dev/null || true
    kill $VITE_PID 2>/dev/null || true
    pkill -f "electron ." 2>/dev/null || true
    pkill -f "vite" 2>/dev/null || true
    echo -e "${GREEN}已停止${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM SIGHUP

# 监控 Electron 进程
while kill -0 $ELECTRON_PID 2>/dev/null; do
    sleep 1
done

# Electron 退出后清理
cleanup
