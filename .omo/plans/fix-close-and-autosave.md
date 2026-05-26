# 修复计划：mac关闭按钮异常退出 + 自动保存未保存内容

## 问题分析

### 问题1：mac关闭按钮导致程序异常退出
**原因**：`mainWindow.on('close')` 中的异步操作和 `e.preventDefault()` 配合不当

**修复方案**：
```typescript
mainWindow.on('close', (e) => {
  e.preventDefault();
  
  // 同步停止后端
  if (backendProcess && !backendProcess.killed) {
    backendProcess.kill('SIGTERM');
    
    // 使用同步轮询等待进程退出
    const startTime = Date.now();
    const checkExit = setInterval(() => {
      if (!backendProcess || backendProcess.killed || Date.now() - startTime > 2000) {
        clearInterval(checkExit);
        mainWindow?.destroy();
      }
    }, 100);
  } else {
    mainWindow?.destroy();
  }
});
```

### 问题2：新建未保存文件丢失
**原因**：新建文件只保存了路径，没有保存内容

**修复方案**：
1. 在 `App.tsx` 中添加自动保存逻辑
2. 监听编辑器内容变化，自动保存到 `localStorage`
3. 启动时恢复未保存的内容

```typescript
// 自动保存未保存的内容
useEffect(() => {
  tabs.forEach(tab => {
    if (tab.modified && tab.file.path.includes('untitled-')) {
      localStorage.setItem(`x-logview-unsaved-${tab.id}`, content);
    }
  });
}, [tabs, content]);
```

## 需要修改的文件

1. `client/src/main/main.ts` - 修复窗口关闭逻辑
2. `client/src/renderer/App.tsx` - 添加自动保存功能
3. `client/src/renderer/components/Editor/Editor.tsx` - 暴露内容变化回调

## 执行步骤

1. 修改 `main.ts` 中的窗口关闭事件处理
2. 在 `App.tsx` 中添加自动保存逻辑
3. 在 `Editor.tsx` 中添加内容变化回调
4. 测试修复效果

是否需要我创建详细的实施计划？
