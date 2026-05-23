# x-logview

万能的跨平台日志查看工具，参考 VS Code 界面设计，支持超大文件查看。

## 功能特性

- **超大文件支持**：流式加载 GB 级别文件，不卡顿
- **智能编码检测**：自动识别 UTF-8、GBK、UTF-16 等编码
- **二进制查看**：十六进制 + 文本双栏显示
- **文件监控**：自动检测文件修改并刷新
- **查找替换**：支持正则表达式，可中断
- **XML/JSON 格式化**：自动格式化、压缩、语法检查
- **数字智能提示**：选中数字显示十六进制、时间戳
- **会话恢复**：自动保存状态，启动后恢复
- **Follow 模式**：实时跟踪日志尾部内容

## 技术栈

- **后端**：Go + WebSocket + gRPC
- **前端**：Electron + React + TypeScript + Vite

## 快速开始

### 安装依赖

```bash
# Go 后端
go mod download

# 前端
cd client && npm install
```

### 开发模式

```bash
# 启动 Go 后端服务
make dev-server

# 启动前端开发服务器（新终端）
make dev-client
```

### 构建

```bash
# 构建所有
make build

# 仅构建后端
make build-server

# 仅构建前端
make build-client
```

## 项目结构

```
x-logview/
├── cmd/server/          # Go 后端入口
├── internal/            # 内部模块
│   ├── file/           # 文件服务
│   ├── encoding/       # 编码检测与转换
│   ├── search/         # 搜索服务
│   ├── format/         # XML/JSON 格式化
│   ├── session/        # 会话管理
│   └── ws/             # WebSocket 服务
├── pkg/                # 公共包
│   ├── buffer/         # 缓冲区管理
│   ├── cache/          # 缓存系统
│   └── config/         # 配置管理
├── client/             # Electron + React 前端
│   ├── src/
│   │   ├── main/       # Electron 主进程
│   │   ├── renderer/   # React 渲染进程
│   │   ├── preload/    # 预加载脚本
│   │   └── shared/     # 共享类型
│   └── ...
├── REQUIREMENTS.md     # 需求文档
├── DESIGN.md           # 设计文档
└── Makefile            # 构建脚本
```

## 配置

配置文件位于 `~/.x-logview/config.json`：

```json
{
  "buffer": {
    "initial_size": 65536,
    "max_size": 268435456,
    "chunk_size": 4096,
    "max_chunks": 1000
  },
  "server": {
    "port": 8090,
    "hostname": "localhost"
  },
  "session": {
    "auto_save": true,
    "auto_save_interval": 30,
    "restore_state": true
  }
}
```

## API 接口

### WebSocket 消息类型

| 类型 | 说明 |
|------|------|
| `file:open` | 打开文件 |
| `file:close` | 关闭文件 |
| `file:content` | 读取内容 |
| `file:update` | 文件更新通知 |
| `search:start` | 开始搜索 |
| `search:cancel` | 取消搜索 |
| `encoding:detect` | 检测编码 |
| `encoding:convert` | 转换编码 |
| `format:json` | 格式化 JSON |
| `format:xml` | 格式化 XML |
| `session:save` | 保存会话 |
| `session:restore` | 恢复会话 |

### REST API

- `GET /health` - 健康检查
- `GET /api/files` - 列出打开的文件
- `GET /api/sessions` - 列出会话

## 许可证

MIT License
