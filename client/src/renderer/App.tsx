import React, { useState, useEffect, useCallback } from 'react';
import { Editor } from './components/Editor/Editor';
import { Sidebar } from './components/Sidebar/Sidebar';
import { RightPanel } from './components/RightPanel/RightPanel';
import { StatusBar } from './components/StatusBar/StatusBar';
import { TabBar } from './components/TabBar/TabBar';
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

  const currentFile = tabs.find(t => t.id === activeTabId)?.file || null;

  useEffect(() => {
    const connectTimer = setTimeout(() => {
      wsService.connect()
        .then(() => setConnected(true))
        .catch(console.error);
    }, 1000);

    const handleBackendRestarted = () => {
      console.log('Backend restarted, reconnecting WebSocket...');
      wsService.reconnect();
    };

    if (window.electronAPI) {
      window.electronAPI.onBackendRestarted?.(handleBackendRestarted);
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
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleFileOpenDialog, activeTabId, handleFileClose, handleFileReload]);

  const handleTextSelect = (text: string) => {
    setSelectedText(text);
  };

  return (
    <div className="app">
      <div className="app-toolbar">
        <button className="toolbar-button" onClick={handleFileOpenDialog} title="打开文件 (Ctrl+O)">
          📂 打开
        </button>
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
      </div>

      <div className="app-content">
        {sidebarOpen && (
          <Sidebar
            onFileOpen={handleFileOpen}
            currentFile={currentFile}
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
    </div>
  );
}

export default App;
