import React, { useState, useEffect } from 'react';
import { FileInfo } from '../../shared/types';
import { wsService } from '../../services/websocket';

interface SidebarProps {
  onFileOpen: (path: string) => void;
  currentFile: FileInfo | null;
}

export const Sidebar: React.FC<SidebarProps> = ({ onFileOpen, currentFile }) => {
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [sessions, setSessions] = useState<any[]>([]);

  useEffect(() => {
    loadFiles();
    loadSessions();
  }, []);

  const loadFiles = async () => {
    try {
      const response = await fetch('http://127.0.0.1:8090/api/files');
      const data = await response.json();
      setFiles(data || []);
    } catch (error) {
      console.error('Failed to load files:', error);
    }
  };

  const loadSessions = async () => {
    try {
      const response = await fetch('http://127.0.0.1:8090/api/sessions');
      const data = await response.json();
      setSessions(data || []);
    } catch (error) {
      console.error('Failed to load sessions:', error);
    }
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  };

  const getFileName = (path: string): string => {
    return path.split('/').pop() || path.split('\\').pop() || path;
  };

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <span className="sidebar-title">Open Files</span>
        <button className="toolbar-button" onClick={loadFiles}>
          Refresh
        </button>
      </div>

      <div className="sidebar-content">
        {files.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state-icon">📂</div>
            <div>No files open</div>
          </div>
        ) : (
          files.map((file, index) => (
            <div
              key={index}
              className={`file-item ${currentFile?.path === file.path ? 'active' : ''}`}
              onClick={() => onFileOpen(file.path)}
            >
              <span>📄</span>
              <div>
                <div>{getFileName(file.path)}</div>
                <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>
                  {formatFileSize(file.size)}
                </div>
              </div>
            </div>
          ))
        )}

        {sessions.length > 0 && (
          <>
            <div className="sidebar-header" style={{ marginTop: '16px' }}>
              <span className="sidebar-title">Recent Sessions</span>
            </div>
            {sessions.map((session, index) => (
              <div
                key={index}
                className="file-item"
                onClick={() => onFileOpen(session.file?.path)}
              >
                <span>💾</span>
                <div>
                  <div>{getFileName(session.file?.path || '')}</div>
                  <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>
                    {new Date(session.updated_at).toLocaleString()}
                  </div>
                </div>
              </div>
            ))}
          </>
        )}
      </div>
    </div>
  );
};
