# x-logview 设计文档

## 1. 系统架构

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      客户端层                                │
├─────────────────────────────────────────────────────────────┤
│  Electron App    │    Web App     │    Mobile App           │
│  (React/Vue)     │    (React/Vue) │    (React Native)       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    通信层 (WebSocket/gRPC)                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      后端服务层                              │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  文件服务    │  │  编码服务    │  │  搜索服务    │         │
│  │  (File)     │  │  (Encoding) │  │  (Search)   │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  格式化服务  │  │  会话服务    │  │  远程服务    │         │
│  │  (Format)   │  │  (Session)  │  │  (Remote)   │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      存储层                                  │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  文件系统    │  │  缓存系统    │  │  配置系统    │         │
│  │  (FS)       │  │  (Cache)    │  │  (Config)   │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 架构特点

- **前后端分离**：前端通过 WebSocket/gRPC 与后端通信
- **异步处理**：所有 I/O 操作异步执行，不阻塞 UI
- **流式处理**：文件内容流式加载，支持超大文件
- **模块化设计**：各服务独立，便于扩展和维护

---

## 2. 技术选型

### 2.1 后端技术栈

| 技术 | 用途 | 说明 |
|------|------|------|
| Go 1.21+ | 主语言 | 高性能、并发友好 |
| gorilla/websocket | WebSocket | 实时通信 |
| gRPC | RPC 框架 | 高效远程调用 |
| mmap | 内存映射 | 大文件处理 |
| chardet | 编码检测 | 自动识别文件编码 |

### 2.2 前端技术栈

| 技术 | 用途 | 说明 |
|------|------|------|
| Electron | 桌面应用框架 | 跨平台桌面应用 |
| React 18+ | UI 框架 | 组件化开发 |
| TypeScript | 类型安全 | 提高代码质量 |
| Monaco Editor | 代码编辑器 | VS Code 核心编辑器 |
| Tailwind CSS | 样式框架 | 快速 UI 开发 |

### 2.3 通信协议

- **WebSocket**：实时双向通信，用于文件内容推送、状态同步
- **gRPC**：高性能 RPC，用于文件操作、搜索等复杂请求
- **Protocol Buffers**：数据序列化，高效传输

---

## 3. 模块设计

### 3.1 后端模块

```
x-logview-server/
├── cmd/
│   └── server/
│       └── main.go              # 入口文件
├── internal/
│   ├── file/                    # 文件服务
│   │   ├── reader.go            # 流式读取
│   │   ├── watcher.go           # 文件监控
│   │   └── detector.go          # 类型检测
│   ├── encoding/                # 编码服务
│   │   ├── detector.go          # 编码检测
│   │   ├── converter.go         # 编码转换
│   │   └── encoding.go          # 编码定义
│   ├── search/                  # 搜索服务
│   │   ├── search.go            # 搜索引擎
│   │   ├── replace.go           # 替换功能
│   │   └── highlight.go         # 高亮匹配
│   ├── format/                  # 格式化服务
│   │   ├── xml.go               # XML 处理
│   │   ├── json.go              # JSON 处理
│   │   └── formatter.go         # 格式化接口
│   ├── session/                 # 会话服务
│   │   ├── session.go           # 会话管理
│   │   ├── state.go             # 状态保存
│   │   └── recovery.go          # 状态恢复
│   ├── remote/                  # 远程服务
│   │   ├── ssh.go               # SSH 连接
│   │   ├── wsl.go               # WSL 连接
│   │   └── remote.go            # 远程接口
│   └── ws/                      # WebSocket 服务
│       ├── handler.go           # 消息处理
│       ├── hub.go               # 连接管理
│       └── message.go           # 消息定义
├── pkg/                         # 公共包
│   ├── buffer/                  # 缓冲区管理
│   ├── cache/                   # 缓存系统
│   └── config/                  # 配置管理
├── api/                         # API 定义
│   ├── proto/                   # Protocol Buffers
│   └── websocket/               # WebSocket 消息
├── go.mod
└── go.sum
```

### 3.2 前端模块

```
x-logview-client/
├── src/
│   ├── main/                    # Electron 主进程
│   │   ├── main.ts              # 入口文件
│   │   ├── ipc.ts               # IPC 通信
│   │   └── window.ts            # 窗口管理
│   ├── renderer/                # 渲染进程
│   │   ├── components/          # React 组件
│   │   │   ├── Editor/          # 编辑器组件
│   │   │   ├── Sidebar/         # 侧边栏组件
│   │   │   ├── StatusBar/       # 状态栏组件
│   │   │   ├── RightPanel/      # 右侧面板
│   │   │   └── Common/          # 通用组件
│   │   ├── hooks/               # 自定义 Hooks
│   │   ├── stores/              # 状态管理
│   │   ├── services/            # 服务层
│   │   │   ├── websocket.ts     # WebSocket 客户端
│   │   │   ├── file.ts          # 文件服务
│   │   │   └── search.ts        # 搜索服务
│   │   ├── utils/               # 工具函数
│   │   ├── types/               # 类型定义
│   │   ├── App.tsx              # 根组件
│   │   └── index.tsx            # 入口文件
│   ├── preload/                 # 预加载脚本
│   │   └── preload.ts
│   └── shared/                  # 共享代码
│       ├── constants.ts         # 常量定义
│       └── types.ts             # 类型定义
├── package.json
└── tsconfig.json
```

---

## 4. 接口设计

### 4.1 WebSocket 消息协议

```typescript
// 消息类型定义
interface WebSocketMessage {
  id: string;           // 消息 ID
  type: string;         // 消息类型
  payload: any;         // 消息内容
  timestamp: number;    // 时间戳
}

// 消息类型枚举
enum MessageType {
  // 文件操作
  FILE_OPEN = 'file:open',
  FILE_CLOSE = 'file:close',
  FILE_CONTENT = 'file:content',
  FILE_UPDATE = 'file:update',
  
  // 搜索操作
  SEARCH_START = 'search:start',
  SEARCH_RESULT = 'search:result',
  SEARCH_CANCEL = 'search:cancel',
  
  // 编码操作
  ENCODING_DETECT = 'encoding:detect',
  ENCODING_CONVERT = 'encoding:convert',
  
  // 格式化操作
  FORMAT_XML = 'format:xml',
  FORMAT_JSON = 'format:json',
  
  // 会话操作
  SESSION_SAVE = 'session:save',
  SESSION_RESTORE = 'session:restore',
  
  // 状态同步
  STATE_SYNC = 'state:sync',
  CURSOR_UPDATE = 'cursor:update'
}
```

### 4.2 核心 API 设计

#### 文件服务 API

```protobuf
service FileService {
  // 打开文件
  rpc OpenFile(OpenFileRequest) returns (OpenFileResponse);
  
  // 读取文件内容（流式）
  rpc ReadContent(ReadContentRequest) returns (stream ContentChunk);
  
  // 关闭文件
  rpc CloseFile(CloseFileRequest) returns (CloseFileResponse);
  
  // 监控文件变化
  rpc WatchFile(WatchFileRequest) returns (stream FileEvent);
}

message OpenFileRequest {
  string path = 1;
  string encoding = 2;
  int64 buffer_size = 3;
}

message ContentChunk {
  int64 offset = 1;
  bytes data = 2;
  int32 line_start = 3;
  int32 line_end = 4;
}
```

#### 编码服务 API

```protobuf
service EncodingService {
  // 检测编码
  rpc DetectEncoding(DetectRequest) returns (DetectResponse);
  
  // 转换编码
  rpc ConvertEncoding(ConvertRequest) returns (ConvertResponse);
}

message DetectRequest {
  bytes sample_data = 1;
}

message DetectResponse {
  string encoding = 1;
  float confidence = 2;
}
```

#### 搜索服务 API

```protobuf
service SearchService {
  // 搜索
  rpc Search(SearchRequest) returns (stream SearchResult);
  
  // 替换
  rpc Replace(ReplaceRequest) returns (ReplaceResponse);
  
  // 取消搜索
  rpc CancelSearch(CancelRequest) returns (CancelResponse);
}

message SearchRequest {
  string pattern = 1;
  bool is_regex = 2;
  bool case_sensitive = 3;
  int64 start_offset = 4;
  int64 end_offset = 5;
}
```

---

## 5. 数据流设计

### 5.1 文件打开流程

```
客户端                    服务端                    文件系统
  │                         │                         │
  │──OpenFile Request──────▶│                         │
  │                         │──Read Sample Data──────▶│
  │                         │◀──Sample Data───────────│
  │                         │                         │
  │                         │──Detect File Type───────│
  │                         │──Detect Encoding────────│
  │                         │                         │
  │◀──OpenFile Response────│                         │
  │   (file info, encoding) │                         │
  │                         │                         │
  │──ReadContent Request───▶│                         │
  │                         │──Stream Read────────────▶│
  │◀──Content Chunk────────│◀──File Data─────────────│
  │◀──Content Chunk────────│◀──File Data─────────────│
  │◀──Content Chunk────────│◀──File Data─────────────│
  │         ...             │         ...             │
```

### 5.2 流式加载流程

```
┌─────────────────────────────────────────────────────────┐
│                    流式加载状态机                         │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐             │
│  │  Idle   │───▶│ Loading │───▶│ Loaded  │             │
│  └─────────┘    └─────────┘    └─────────┘             │
│       │              │              │                   │
│       │              ▼              │                   │
│       │         ┌─────────┐        │                   │
│       │         │ Error   │        │                   │
│       │         └─────────┘        │                   │
│       │              │              │                   │
│       ▼              ▼              ▼                   │
│  ┌─────────────────────────────────────────┐           │
│  │              Buffer Manager             │           │
│  │  ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐   │           │
│  │  │Chunk│  │Chunk│  │Chunk│  │Chunk│   │           │
│  │  │  1  │  │  2  │  │  3  │  │  4  │   │           │
│  │  └─────┘  └─────┘  └─────┘  └─────┘   │           │
│  └─────────────────────────────────────────┘           │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 5.3 搜索流程

```
客户端                    服务端
  │                         │
  │──Search Request────────▶│
  │                         │──Start Search Goroutine
  │                         │
  │◀──Search Progress──────│   (每找到一个结果推送)
  │◀──Search Progress──────│
  │◀──Search Progress──────│
  │         ...             │
  │                         │
  │──Cancel Search────────▶│   (用户中断)
  │                         │──Stop Goroutine
  │◀──Search Cancelled─────│
  │                         │
  │◀──Search Complete──────│   (搜索完成)
```

---

## 6. 状态管理

### 6.1 会话状态结构

```typescript
interface SessionState {
  // 文件信息
  file: {
    path: string;
    encoding: string;
    size: number;
    lastModified: number;
  };
  
  // 编辑器状态
  editor: {
    cursorPosition: {
      line: number;
      column: number;
    };
    scrollPosition: {
      top: number;
      left: number;
    };
    viewport: {
      width: number;
      height: number;
    };
  };
  
  // 未保存的修改
  changes: {
    content: string;
    timestamp: number;
  }[];
  
  // 临时文件路径
  tempFile: string;
}
```

### 6.2 状态保存与恢复

```
┌─────────────────────────────────────────────────────────┐
│                    状态管理流程                           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  自动保存触发条件：                                      │
│  ├── 定时保存（每 30 秒）                                │
│  ├── 内容变化时（防抖 5 秒）                             │
│  ├── 窗口失去焦点时                                      │
│  └── 应用关闭前                                          │
│                                                         │
│  保存内容：                                              │
│  ├── 会话状态 JSON → ~/.x-logview/sessions/             │
│  ├── 未保存内容 → ~/.x-logview/cache/                   │
│  └── 临时文件 → ~/.x-logview/temp/                      │
│                                                         │
│  恢复流程：                                              │
│  ├── 1. 读取会话状态                                     │
│  ├── 2. 检查临时文件完整性                               │
│  ├── 3. 恢复未保存的修改                                 │
│  └── 4. 恢复光标和滚动位置                               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## 7. 性能优化

### 7.1 缓冲区管理

```go
// 缓冲区配置
type BufferConfig struct {
    InitialSize int64  // 初始缓冲区大小
    MaxSize     int64  // 最大缓冲区大小
    ChunkSize   int64  // 每个块的大小
    MaxChunks   int    // 最大缓存块数
}

// 默认配置
DefaultBufferConfig = BufferConfig{
    InitialSize: 64 * 1024,      // 64KB
    MaxSize:     256 * 1024 * 1024, // 256MB
    ChunkSize:   4 * 1024,       // 4KB
    MaxChunks:   1000,
}
```

### 7.2 内存管理

- **LRU 缓存**：最近使用的块保留在内存中
- **预加载**：根据滚动方向预加载相邻块
- **释放策略**：超过最大缓存数时释放最久未使用的块
- **内存监控**：实时监控内存使用，动态调整缓存大小

### 7.3 渲染优化

- **虚拟滚动**：只渲染可见区域的内容
- **增量更新**：只更新变化的部分
- **Web Worker**：在独立线程处理复杂计算
- **RequestAnimationFrame**：使用浏览器原生动画帧

---

## 8. 部署架构

### 8.1 桌面应用部署

```
┌─────────────────────────────────────────────────────────┐
│                    Electron 应用                         │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────┐   │
│  │                 渲染进程 (React)                 │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐        │   │
│  │  │  Editor  │  │ Sidebar │  │ StatusBar│        │   │
│  │  └─────────┘  └─────────┘  └─────────┘        │   │
│  └─────────────────────────────────────────────────┘   │
│                          │ IPC                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │                 主进程 (Node.js)                 │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐        │   │
│  │  │WebSocket│  │  gRPC   │  │  File   │        │   │
│  │  │ Client  │  │ Client  │  │ System  │        │   │
│  │  └─────────┘  └─────────┘  └─────────┘        │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    Go 后端服务                           │
│  (可以是本地服务或远程服务)                               │
└─────────────────────────────────────────────────────────┘
```

### 8.2 远程访问架构

```
┌─────────────┐     SSH/WSL      ┌─────────────┐
│   本地客户端  │ ◀──────────────▶ │   远程机器    │
│  (Electron)  │                  │             │
└─────────────┘                  └─────────────┘
       │                                │
       ▼                                ▼
┌─────────────┐                  ┌─────────────┐
│  本地后端    │                  │  远程后端    │
│  (Go)       │                  │  (Go)       │
└─────────────┘                  └─────────────┘
```

---

## 9. 安全设计

### 9.1 文件访问控制

- **路径验证**：防止路径遍历攻击
- **权限检查**：检查文件读取权限
- **沙箱环境**：Electron 应用使用沙箱隔离

### 9.2 远程连接安全

- **SSH 密钥认证**：支持密钥文件认证
- **加密传输**：所有数据通过加密通道传输
- **会话超时**：自动断开空闲连接

### 9.3 本地数据安全

- **敏感信息加密**：配置文件中的敏感信息加密存储
- **临时文件清理**：应用退出时清理临时文件
- **日志脱敏**：日志中不记录敏感内容

---

## 10. 扩展性设计

### 10.1 插件系统

```typescript
// 插件接口
interface Plugin {
  id: string;
  name: string;
  version: string;
  
  // 生命周期
  activate(): void;
  deactivate(): void;
  
  // 扩展点
  onFileOpen?(file: FileInfo): void;
  onContentChange?(change: ContentChange): void;
  onSearch?(query: string): void;
}

// 插件管理器
class PluginManager {
  register(plugin: Plugin): void;
  unregister(pluginId: string): void;
  getPlugin(pluginId: string): Plugin;
}
```

### 10.2 主题系统

```typescript
// 主题定义
interface Theme {
  id: string;
  name: string;
  colors: {
    background: string;
    foreground: string;
    selection: string;
    // ... 更多颜色
  };
  fonts: {
    editor: string;
    ui: string;
  };
}

// 主题管理器
class ThemeManager {
  setTheme(themeId: string): void;
  getTheme(): Theme;
  registerTheme(theme: Theme): void;
}
```

---

## 11. 测试策略

### 11.1 单元测试

- **后端**：使用 Go 标准 testing 包
- **前端**：使用 Jest + React Testing Library
- **覆盖率目标**：> 80%

### 11.2 集成测试

- **API 测试**：使用 gRPC 测试工具
- **E2E 测试**：使用 Playwright
- **性能测试**：使用 Go benchmark

### 11.3 测试数据

- **小文件**：< 1MB
- **中等文件**：1MB - 100MB
- **大文件**：100MB - 10GB
- **超大文件**：> 10GB

---

## 12. 开发计划

### 12.1 第一阶段（4 周）

- [ ] 后端基础框架搭建
- [ ] 文件流式读取实现
- [ ] WebSocket 通信实现
- [ ] 前端基础框架搭建
- [ ] 基本编辑器集成

### 12.2 第二阶段（4 周）

- [ ] 编码检测与转换
- [ ] 二进制文件显示
- [ ] 查找替换功能
- [ ] 会话状态管理

### 12.3 第三阶段（4 周）

- [ ] XML/JSON 处理
- [ ] 远程连接支持
- [ ] 性能优化
- [ ] 测试与修复

### 12.4 第四阶段（2 周）

- [ ] 文档编写
- [ ] 打包发布
- [ ] 用户反馈收集
