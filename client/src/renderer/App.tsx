import React, { useState, useEffect } from 'react';
import { Editor } from './components/Editor/Editor';
import { Sidebar } from './components/Sidebar/Sidebar';
import { RightPanel } from './components/RightPanel/RightPanel';
import { StatusBar } from './components/StatusBar/StatusBar';
import { wsService } from './services/websocket';
import { FileInfo } from '../shared/types';
import './App.css';

function App() {
  const [connected, setConnected] = useState(false);
  const [currentFile, setCurrentFile] = useState<FileInfo | null>(null);
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [rightPanelOpen, setRightPanelOpen] = useState(true);
  const [selectedText, setSelectedText] = useState('');

  useEffect(() => {
    wsService.connect()
      .then(() => setConnected(true))
      .catch(console.error);

    return () => {
      wsService.disconnect();
    };
  }, []);

  const handleFileOpen = async (path: string) => {
    try {
      const info = await wsService.send('file:open', { path });
      setCurrentFile(info.payload);
    } catch (error) {
      console.error('Failed to open file:', error);
    }
  };

  const handleTextSelect = (text: string) => {
    setSelectedText(text);
  };

  return (
    <div className="app">
      <div className="app-content">
        {sidebarOpen && (
          <Sidebar
            onFileOpen={handleFileOpen}
            currentFile={currentFile}
          />
        )}
        <Editor
          file={currentFile}
          onTextSelect={handleTextSelect}
          onToggleSidebar={() => setSidebarOpen(!sidebarOpen)}
          onToggleRightPanel={() => setRightPanelOpen(!rightPanelOpen)}
        />
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
