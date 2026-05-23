# x-logview 测试补充 TODO List

## 一、后端单元测试补充

### 1. encoding 模块 (`internal/encoding/encoding_test.go`)
- [ ] TestIsGBK - GBK 编码检测
- [ ] TestIsBig5 - Big5 编码检测
- [ ] TestIsShiftJIS - Shift-JIS 编码检测
- [ ] TestNewDecodedReader - 解码 Reader 创建（各编码类型）
- [ ] TestConvertEncodingGBK - GBK 转 UTF-8
- [ ] TestConvertEncodingUTF16 - UTF-16 转 UTF-8
- [ ] TestConvertEncodingBig5 - Big5 转 UTF-8
- [ ] TestConvertEncodingShiftJIS - Shift-JIS 转 UTF-8

### 2. file 模块 (`internal/file/reader_test.go`)
- [ ] TestFileServiceReadSample - 采样读取
- [ ] TestFileServiceReadEmptyFile - 空文件读取
- [ ] TestFileServiceReadLargeFile - 大文件读取（多次加载）
- [ ] TestFileServiceDuplicateOpen - 重复打开同一文件

### 3. session 模块 (`internal/session/session_test.go`)
- [ ] TestSessionManagerUpdateTempFile - 更新临时文件路径
- [ ] TestSessionManagerUpdateEditorState - 更新编辑器状态（光标、滚动位置）

### 4. ws 模块 (`internal/ws/handler_test.go`)
- [ ] TestHubHandleRemoteConnect - remote:connect 消息处理
- [ ] TestHubHandleRemoteDisconnect - remote:disconnect 消息处理
- [ ] TestHubHandleRemoteList - remote:list 消息处理
- [ ] TestHubHandleRemoteExec - remote:exec 消息处理
- [ ] TestHubHandleAutoSave - autosave:save 消息处理
- [ ] TestHubHandleAutoSaveRestore - autosave:restore 消息处理
- [ ] TestHubHandleAutoSaveUpdate - autosave:update 消息处理
- [ ] TestHubHandleSearchReplace - search:replace 消息处理
- [ ] TestHubHandleEncodeDetect - encoding:detect 消息处理
- [ ] TestHubHandleEncodeConvert - encoding:convert 消息处理
- [ ] TestHubHandleFormatJSON - format:json 消息处理
- [ ] TestHubHandleFormatXML - format:xml 消息处理
- [ ] TestHubHandleSessionSave - session:save 消息处理
- [ ] TestHubHandleSessionRestore - session:restore 消息处理
- [ ] TestHubHandleStateSync - state:sync 消息处理
- [ ] TestHubHandleCursorUpdate - cursor:update 消息处理

## 二、后端集成测试

### 5. HTTP API 集成测试 (`cmd/server/main_test.go`)
- [ ] TestHealthEndpoint - GET /health 返回 200 和 {"status":"ok"}
- [ ] TestHealthEndpointMethodNotAllowed - POST /health 返回 405
- [ ] TestFilesEndpoint - GET /api/files 返回文件列表
- [ ] TestFilesEndpointEmpty - 无打开文件时返回空数组
- [ ] TestSessionsEndpoint - GET /api/sessions 返回会话列表
- [ ] TestSessionsEndpointEmpty - 无会话时返回空数组
- [ ] TestCORSPreflight - OPTIONS 请求返回正确的 CORS 头
- [ ] TestCORSHeaders - 跨域请求包含 Access-Control-Allow-Origin

### 6. WebSocket 集成测试 (`cmd/server/ws_test.go`)
- [ ] TestWSFileOpenAndRead - 打开文件并读取内容完整流程
- [ ] TestWSFileOpenNotFound - 打开不存在的文件返回错误
- [ ] TestWSFileClose - 关闭文件后无法再读取
- [ ] TestWSSearchAndReplace - 搜索并替换完整流程
- [ ] TestWSSearchCancel - 取消搜索
- [ ] TestWSEncodingDetect - 检测文件编码
- [ ] TestWSEncodingConvert - 转换文件编码
- [ ] TestWSFormatJSON - 格式化 JSON
- [ ] TestWSFormatXML - 格式化 XML
- [ ] TestWSSessionSaveAndRestore - 保存并恢复会话
- [ ] TestWSAutoSave - 注册自动保存并恢复
- [ ] TestWSRemoteConnect - SSH 连接（mock）
- [ ] TestWSInvalidMessage - 发送无效消息不断开连接
- [ ] TestWSConcurrentClients - 多客户端并发连接

## 三、前端测试

### 7. WebSocket 服务测试 (`client/src/services/websocket.test.ts`)
- [ ] TestWebSocketConnect - 连接成功
- [ ] TestWebSocketDisconnect - 断开连接
- [ ] TestWebSocketSend - 发送消息并收到响应
- [ ] TestWebSocketReconnect - 断线自动重连
- [ ] TestWebSocketReconnectMaxAttempts - 超过最大重连次数停止
- [ ] TestWebSocketOnOff - 事件监听注册和移除

### 8. 文件服务测试 (`client/src/services/file.test.ts`)
- [ ] TestFileServiceOpen - 打开文件
- [ ] TestFileServiceClose - 关闭文件
- [ ] TestFileServiceReadContent - 读取内容
- [ ] TestFileServiceSearch - 搜索
- [ ] TestFileServiceReplace - 替换
- [ ] TestFileServiceFormatJSON - 格式化 JSON
- [ ] TestFileServiceFormatXML - 格式化 XML
- [ ] TestFileServiceAutoSave - 自动保存注册和恢复

## 四、执行优先级

| 优先级 | 任务 | 原因 |
|--------|------|------|
| P0 | 5. HTTP API 集成测试 | 核心接口，必须覆盖 |
| P0 | 6. WebSocket 集成测试 | 核心通信，必须覆盖 |
| P0 | 4. ws 模块新消息测试 | 新增功能未覆盖 |
| P1 | 1. encoding 补充 | 编码转换是核心功能 |
| P1 | 2. file 补充 | 文件读取是核心功能 |
| P1 | 7. WebSocket 前端测试 | 前端核心服务 |
| P2 | 3. session 补充 | 会话管理补充 |
| P2 | 8. 文件服务前端测试 | 前端服务补充 |
