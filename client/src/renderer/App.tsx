import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Editor } from './components/Editor/Editor';
import { Sidebar } from './components/Sidebar/Sidebar';
import { RightPanel } from './components/RightPanel/RightPanel';
import { StatusBar } from './components/StatusBar/StatusBar';
import { TabBar } from './components/TabBar/TabBar';
import { Settings } from './components/Settings/Settings';
import { wsService } from './services/websocket';
import { sessionService, SessionData, FileState } from './services/session';
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
  const [theme, setTheme] = useState(() => {
    return localStorage.getItem('x-logview-theme') || 'opencode';
  });
  const [editorContent, setEditorContent] = useState<Record<string, string>>({});
  const [sessionId, setSessionId] = useState<string | null>(null);
  const sessionInitialized = useRef(false);

  const currentFile = tabs.find(t => t.id === activeTabId)?.file || null;

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('x-logview-theme', theme);
  }, [theme]);

  const syncSession = useCallback(async (tabList: Tab[], content: Record<string, string>, activeId: string | null) => {
    if (!sessionId) return;

    for (const tab of tabList) {
      const fileState: FileState = {
        session_id: sessionId,
        file_path: tab.file.path,
        isUntitled: tab.file.path.includes('untitled-'),
        content: content[tab.id] || '',
        cursor_line: 0,
        cursor_col: 0,
        scroll_top: 0,
        scroll_left: 0,
        is_active: tab.id === activeId,
        edit_history: [],
      };

      try {
        await sessionService.updateFile(sessionId, tab.file.path, fileState);
      } catch {
        await sessionService.addFile(sessionId, fileState);
      }
    }
  }, [sessionId]);

  useEffect(() => {
    if (sessionId && tabs.length >= 0) {
      syncSession(tabs, editorContent, activeTabId);
    }
  }, [tabs, editorContent, activeTabId, sessionId, syncSession]);

  useEffect(() => {
    const connectTimer = setTimeout(async () => {
      try {
        await wsService.connect();
        setConnected(true);

        if (!sessionInitialized.current) {
          sessionInitialized.current = true;
          let session = await sessionService.getActiveSession();
          
          if (!session) {
            const newId = `session-${Date.now()}`;
            await sessionService.createSession(newId, 'Default Session');
            session = await sessionService.getActiveSession();
          }

          if (session) {
            setSessionId(session.id);
            
            if (session.files && session.files.length > 0) {
              const newTabs: Tab[] = [];
              const newContent: Record<string, string> = {};

              for (const file of session.files) {
                const tabId = `tab-${Date.now()}-${Math.random()}`;
                
                if (file.isUntitled) {
                  newTabs.push({
                    id: tabId,
                    file: {
                      path: file.file_path,
                      size: 0,
                      mod_time: new Date().toISOString(),
                      file_type: 'text',
                      encoding: 'utf-8',
                      total_lines: 0,
                      loaded: true,
                    },
                    modified: true,
                  });
                  newContent[tabId] = file.content;
                } else {
                  try {
                    const response = await wsService.send('file:open', { path: file.file_path });
                    const fileInfo: FileInfo = response.payload;
                    newTabs.push({
                      id: tabId,
                      file: fileInfo,
                      modified: false,
                    });
                    newContent[tabId] = file.content;
                  } catch (error) {
                    console.error('Failed to restore file:', file.file_path, error);
                  }
                }
              }

              if (newTabs.length > 0) {
                setTabs(newTabs);
                setEditorContent(newContent);
                
                const activeFile = session.files.find(f => f.is_active);
                if (activeFile) {
                  const activeTab = newTabs.find(t => t.file.path === activeFile.file_path);
                  if (activeTab) {
                    setActiveTabId(activeTab.id);
                  }
                } else if (newTabs.length > 0) {
                  setActiveTabId(newTabs[0].id);
                }
              }
            }
          }
        }
      } catch (error) {
        console.error('Failed to connect:', error);
      }
    }, 3000);

    if (window.electronAPI) {
      window.electronAPI.onBackendRestarted?.(() => {
        wsService.reconnect();
      });
    }

    return () => {
      clearTimeout(connectTimer);
      wsService.disconnect();
    };
  }, []);

  const handleFileOpen = useCallback(async (path: string) => {
    const existingTab = tabs.find(t => t.file.path === path);
    if (existingTab) {
      setActiveTabId(existingTab.id);
      return;
    }

    try {
      const response = await wsService.send('file:open', { path });
      const fileInfo: FileInfo = response.payload;

      const newTab: Tab = {
        id: `tab-${Date.now()}`,
        file: fileInfo,
        modified: false,
      };

      setTabs(prev => [...prev, newTab]);
      setActiveTabId(newTab.id);
    } catch (error) {
      console.error('Failed to open file:', error);
    }
  }, [tabs]);

  const handleFileClose = useCallback(async (tabId: string) => {
    const tab = tabs.find(t => t.id === tabId);
    if (tab && sessionId) {
      await sessionService.removeFile(sessionId, tab.file.path);
      wsService.send('file:close', { path: tab.file.path }).catch(console.error);
    }

    setTabs(prev => prev.filter(t => t.id !== tabId));
    setEditorContent(prev => {
      const newContent = { ...prev };
      delete newContent[tabId];
      return newContent;
    });

    if (activeTabId === tabId) {
      const remainingTabs = tabs.filter(t => t.id !== tabId);
      setActiveTabId(remainingTabs.length > 0 ? remainingTabs[remainingTabs.length - 1].id : null);
    }
  }, [tabs, activeTabId, sessionId]);

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

  const handleNewFile = useCallback(() => {
    const userDataPath = localStorage.getItem('x-logview-user-data-path') || '~/.x-logview';
    const tempDir = `${userDataPath}/temp`;
    const fileName = `untitled-${Date.now()}.txt`;
    const filePath = `${tempDir}/${fileName}`;
    
    const newTab: Tab = {
      id: `tab-${Date.now()}`,
      file: {
        path: filePath,
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
        const content = editorContent[activeTab.id] || '';
        const result = await window.electronAPI.saveFile(currentFile.path, content);
        if (result?.success) {
          setTabs(prev => prev.map(t =>
            t.id === activeTabId ? { ...t, modified: false } : t
          ));
        }
      }
    }
  }, [currentFile, activeTabId, tabs, editorContent]);

  const handleSaveFileAs = useCallback(async () => {
    if (!currentFile) return;

    if (window.electronAPI) {
      const activeTab = tabs.find(t => t.id === activeTabId);
      const content = activeTab ? editorContent[activeTab.id] || '' : '';
      const result = await window.electronAPI.saveFileAs(content);
      if (result?.success && result.path) {
        if (activeTabId) {
          setTabs(prev => prev.map(t =>
            t.id === activeTabId
              ? { ...t, file: { ...t.file, path: result.path }, modified: false }
              : t
          ));
        }
      }
    }
  }, [currentFile, activeTabId, editorContent]);

  const handleContentChange = useCallback((tabId: string, content: string) => {
    setEditorContent(prev => ({ ...prev, [tabId]: content }));
    setTabs(prev => prev.map(t =>
      t.id === tabId ? { ...t, modified: true } : t
    ));
  }, []);

  const handleTextSelect = (text: string) => {
    setSelectedText(text);
  };

  const handleFileOpenDialog = useCallback(async () => {
    if (window.electronAPI) {
      const path = await window.electronAPI.openFile();
      if (path) {
        handleFileOpen(path);
      }
    }
  }, [handleFileOpen]);

  const handleFormatJSON = useCallback(async () => {
    if (!currentFile) return;
    try {
      await wsService.send('format:json', { path: currentFile.path });
    } catch (error) {
      console.error('Failed to format JSON:', error);
    }
  }, [currentFile]);

  const handleMinifyJSON = useCallback(async () => {
    if (!currentFile) return;
    try {
      await wsService.send('format:json', { path: currentFile.path, minify: true });
    } catch (error) {
      console.error('Failed to minify JSON:', error);
    }
  }, [currentFile]);

  const handleFormatXML = useCallback(async () => {
    if (!currentFile) return;
    try {
      await wsService.send('format:xml', { path: currentFile.path });
    } catch (error) {
      console.error('Failed to format XML:', error);
    }
  }, [currentFile]);

  const handleMinifyXML = useCallback(async () => {
    if (!currentFile) return;
    try {
      await wsService.send('format:xml', { path: currentFile.path, minify: true });
    } catch (error) {
      console.error('Failed to minify XML:', error);
    }
  }, [currentFile]);

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
        <button className="toolbar-button" onClick={handleFormatJSON} disabled={!currentFile} title="格式化 JSON">
          { } 格式化
        </button>
        <button className="toolbar-button" onClick={handleMinifyJSON} disabled={!currentFile} title="压缩 JSON">
          { } 压缩
        </button>
        <div className="toolbar-separator" />
        <button className="toolbar-button" onClick={() => setSidebarOpen(!sidebarOpen)}>
          {sidebarOpen ? '◀' : '▶'} 侧边栏
        </button>
        <button className="toolbar-button" onClick={() => setRightPanelOpen(!rightPanelOpen)}>
          {rightPanelOpen ? '▶' : '◀'} 面板
        </button>
        <div className="toolbar-separator" />
        <button className="toolbar-button" onClick={() => setSettingsOpen(true)} title="设置">
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
              if (activeTabId) {
                handleContentChange(activeTabId, content);
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
      <Settings
        isOpen={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        onThemeChange={setTheme}
      />
    </div>
  );
}

export default App;
