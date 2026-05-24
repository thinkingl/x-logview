import React, { useState, useEffect, useCallback } from 'react';
import { Editor } from './components/Editor/Editor';
import { Sidebar } from './components/Sidebar/Sidebar';
import { RightPanel } from './components/RightPanel/RightPanel';
import { StatusBar } from './components/StatusBar/StatusBar';
import { TabBar } from './components/TabBar/TabBar';
import { Settings } from './components/Settings/Settings';
import { wsService } from './services/websocket';
import { FileInfo } from './shared/types';
import './App.css';

interface Tab {
  id: string;
  file: FileInfo;
  modified: boolean;
}

function App() {
  const [connected, setConnected] = useState(false);
  const [tabs, setTabs] = useState<Tab[]>([]);
  const [activeTabId, setActiveTabId] = useState<string | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [rightPanelOpen, setRightPanelOpen] = useState(true);
  const [selectedText, setSelectedText] = useState('');
  const [settingsOpen, setSettingsOpen] = useState(false);

  const currentFile = tabs.find(t => t.id === activeTabId)?.file || null;

  // 保存打开的文件到 localStorage
  const saveOpenFiles = useCallback(() => {
    const filePaths = tabs.map(t => t.file.path);
    localStorage.setItem('x-logview-open-files', JSON.stringify(filePaths));
  }, [tabs]);

  // 当 tabs 变化时保存
  useEffect(() => {
    if (tabs.length > 0) {
      saveOpenFiles();
    }
  }, [tabs, saveOpenFiles]);

  useEffect(() => {
    // 延迟连接，等待后端启动完成
    const connectTimer = setTimeout(() => {
      wsService.connect()
        .then(() => setConnected(true))
        .catch(console.error);
    }, 3000);

    const handleBackendRestarted = () => {
      console.log('Backend restarted, reconnecting WebSocket...');
      wsService.reconnect();
    };

    // 菜单事件处理
    const handleMenuNewFile = () => handleNewFile();
    const handleMenuOpenFile = () => handleFileOpenDialog();
    const handleMenuSave = () => handleSaveFile();
    const handleMenuSaveAs = () => handleSaveFileAs();
    const handleMenuCloseTab = () => {
      if (activeTabId) handleFileClose(activeTabId);
    };
    const handleMenuReload = () => handleFileReload();

    if (window.electronAPI) {
      window.electronAPI.onBackendRestarted?.(handleBackendRestarted);
      window.electronAPI.onMenuNewFile?.(handleMenuNewFile);
      window.electronAPI.onMenuOpenFile?.(handleMenuOpenFile);
      window.electronAPI.onMenuSave?.(handleMenuSave);
      window.electronAPI.onMenuSaveAs?.(handleMenuSaveAs);
      window.electronAPI.onMenuCloseTab?.(handleMenuCloseTab);
      window.electronAPI.onMenuReload?.(handleMenuReload);
    }

    return () => {
      clearTimeout(connectTimer);
      wsService.disconnect();
    };
  }, []);

  const handleFileOpen = useCallback(async (path: string) => {
    console.log('Opening file:', path);
    const existingTab = tabs.find(t => t.file.path === path);
    if (existingTab) {
      console.log('File already open, switching to tab');
      setActiveTabId(existingTab.id);
      return;
    }

    try {
      console.log('Sending file:open to backend');
      const response = await wsService.send('file:open', { path });
      console.log('Received response:', response);
      const fileInfo: FileInfo = response.payload;

      const newTab: Tab = {
        id: `tab-${Date.now()}`,
        file: fileInfo,
        modified: false,
      };

      console.log('Creating new tab:', newTab);
      setTabs(prev => [...prev, newTab]);
      setActiveTabId(newTab.id);
    } catch (error) {
      console.error('Failed to open file:', error);
    }
  }, [tabs]);

  const handleFileClose = useCallback((tabId: string) => {
    const tab = tabs.find(t => t.id === tabId);
    if (tab) {
      wsService.send('file:close', { path: tab.file.path }).catch(console.error);
    }

    setTabs(prev => prev.filter(t => t.id !== tabId));

    if (activeTabId === tabId) {
      const remainingTabs = tabs.filter(t => t.id !== tabId);
      setActiveTabId(remainingTabs.length > 0 ? remainingTabs[remainingTabs.length - 1].id : null);
    }
  }, [tabs, activeTabId]);

  const handleFileReload = useCallback(async () => {
    if (!currentFile) return;

    try {
      const response = await wsService.send('file:open', { path: currentFile.path });
      const fileInfo: FileInfo = response.payload;

      setTabs(prev => prev.map(t =>
        t.id === activeTabId ? { ...t, file: fileInfo } : t
      ));
    } catch (error) {
      console.error('Failed to reload file:', error);
    }
  }, [currentFile, activeTabId]);

  // 恢复打开的文件
  useEffect(() => {
    if (!connected) return;

    const saved = localStorage.getItem('x-logview-open-files');
    if (!saved) return;

    try {
      const filePaths: string[] = JSON.parse(saved);
      if (filePaths.length > 0) {
        console.log('Restoring open files:', filePaths);
        filePaths.forEach(path => handleFileOpen(path));
      }
    } catch (error) {
      console.error('Failed to restore open files:', error);
    }
  }, [connected, handleFileOpen]);

  const handleFileOpenDialog = useCallback(async () => {
    console.log('Opening file dialog');
    if (window.electronAPI) {
      console.log('electronAPI available');
      const path = await window.electronAPI.openFile();
      console.log('Selected path:', path);
      if (path) {
        handleFileOpen(path);
      }
    } else {
      console.error('electronAPI not available');
    }
  }, [handleFileOpen]);

  const handleNewFile = useCallback(() => {
    const newTab: Tab = {
      id: `tab-${Date.now()}`,
      file: {
        path: `untitled-${Date.now()}.txt`,
        size: 0,
        mod_time: new Date().toISOString(),
        file_type: 'text',
        encoding: 'utf-8',
        total_lines: 0,
        loaded: true,
      },
      modified: false,
    };
    setTabs(prev => [...prev, newTab]);
    setActiveTabId(newTab.id);
  }, []);

  const handleSaveFile = useCallback(async () => {
    if (!currentFile) return;

    if (window.electronAPI) {
      const activeTab = tabs.find(t => t.id === activeTabId);
      if (activeTab) {
        const result = await window.electronAPI.saveFile(currentFile.path, '');
        if (result?.success) {
          console.log('File saved successfully');
        }
      }
    }
  }, [currentFile, activeTabId, tabs]);

  const handleSaveFileAs = useCallback(async () => {
    if (!currentFile) return;

    if (window.electronAPI) {
      const result = await window.electronAPI.saveFileAs('');
      if (result?.success && result.path) {
        console.log('File saved to:', result.path);
        // 更新 tab 的文件路径
        if (activeTabId) {
          setTabs(prev => prev.map(t =>
            t.id === activeTabId
              ? { ...t, file: { ...t.file, path: result.path } }
              : t
          ));
        }
      }
    }
  }, [currentFile, activeTabId]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'o') {
        e.preventDefault();
        handleFileOpenDialog();
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'w') {
        e.preventDefault();
        if (activeTabId) {
          handleFileClose(activeTabId);
        }
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
        e.preventDefault();
        handleFileReload();
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault();
        handleNewFile();
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        if (e.shiftKey) {
          handleSaveFileAs();
        } else {
          handleSaveFile();
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleFileOpenDialog, activeTabId, handleFileClose, handleFileReload, handleNewFile, handleSaveFile, handleSaveFileAs]);

  const handleTextSelect = (text: string) => {
    setSelectedText(text);
  };

  return (
    <div className="app">
      <div className="app-toolbar">
        <button className="toolbar-button" onClick={handleNewFile} title="新建文件 (Ctrl+N)">
          📄 新建
        </button>
        <button className="toolbar-button" onClick={handleFileOpenDialog} title="打开文件 (Ctrl+O)">
          📂 打开
        </button>
        <button
          className="toolbar-button"
          onClick={handleSaveFile}
          disabled={!currentFile}
          title="保存文件 (Ctrl+S)"
        >
          💾 保存
        </button>
        <button
          className="toolbar-button"
          onClick={handleSaveFileAs}
          disabled={!currentFile}
          title="另存为 (Ctrl+Shift+S)"
        >
          📋 另存为
        </button>
        <div className="toolbar-separator" />
        <button
          className="toolbar-button"
          onClick={() => activeTabId && handleFileClose(activeTabId)}
          disabled={!activeTabId}
          title="关闭文件 (Ctrl+W)"
        >
          ✕ 关闭
        </button>
        <button
          className="toolbar-button"
          onClick={handleFileReload}
          disabled={!currentFile}
          title="重新加载 (Ctrl+R)"
        >
          🔄 重新加载
        </button>
        <div className="toolbar-separator" />
        <button
          className="toolbar-button"
          onClick={() => setSidebarOpen(!sidebarOpen)}
        >
          {sidebarOpen ? '◀' : '▶'} 侧边栏
        </button>
        <button
          className="toolbar-button"
          onClick={() => setRightPanelOpen(!rightPanelOpen)}
        >
          {rightPanelOpen ? '▶' : '◀'} 面板
        </button>
        <div className="toolbar-separator" />
        <button
          className="toolbar-button"
          onClick={() => setSettingsOpen(true)}
          title="设置"
        >
          ⚙ 设置
        </button>
      </div>

      <div className="app-content">
        {sidebarOpen && (
          <Sidebar
            onFileOpen={handleFileOpen}
            currentFile={currentFile}
            files={tabs.map(t => t.file)}
            onRefreshFiles={() => {}}
          />
        )}
        <div className="editor-area">
          <TabBar
            tabs={tabs}
            activeTabId={activeTabId}
            onTabSelect={setActiveTabId}
            onTabClose={handleFileClose}
          />
          <Editor
            file={currentFile}
            onTextSelect={handleTextSelect}
            onFileModified={(modified) => {
              if (activeTabId) {
                setTabs(prev => prev.map(t =>
                  t.id === activeTabId ? { ...t, modified } : t
                ));
              }
            }}
            onContentChange={(content) => {
              // 可以在这里保存内容到状态
              console.log('Content changed:', content.substring(0, 100));
            }}
          />
        </div>
        {rightPanelOpen && (
          <RightPanel
            selectedText={selectedText}
            file={currentFile}
          />
        )}
      </div>
      <StatusBar
        file={currentFile}
        connected={connected}
      />
      <Settings
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
      />
    </div>
  );
}

export default App;
