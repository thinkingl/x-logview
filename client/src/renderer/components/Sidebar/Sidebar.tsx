import React, { useState, useEffect, useRef } from 'react';
import { FileInfo } from '../../shared/types';

interface SidebarProps {
  onFileOpen: (path: string) => void;
  currentFile: FileInfo | null;
  files: FileInfo[];
  onRefreshFiles: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ onFileOpen, currentFile, files, onRefreshFiles }) => {
  const [sessions, setSessions] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const retryCountRef = useRef(0);
  const maxRetries = 3;

  useEffect(() => {
    // 延迟加载，等待后端启动完成
    const loadTimer = setTimeout(() => {
      loadSessions();
    }, 4000);

    return () => clearTimeout(loadTimer);
  }, []);

  const loadSessions = async () => {
    if (retryCountRef.current >= maxRetries) {
      setLoading(false);
      return;
    }

    try {
      const response = await fetch('http://127.0.0.1:8090/api/sessions');
      if (!response.ok) {
        throw new Error('Failed to fetch');
      }
      const data = await response.json();
      setSessions(data || []);
      setLoading(false);
    } catch (error) {
      console.error('Failed to load sessions:', error);
      retryCountRef.current += 1;
      if (retryCountRef.current < maxRetries) {
        setTimeout(loadSessions, 5000);
      } else {
        setLoading(false);
      }
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
        <button className="toolbar-button" onClick={onRefreshFiles}>
          Refresh
        </button>
      </div>

      <div className="sidebar-content">
        {loading ? (
          <div className="empty-state">
            <div className="empty-state-icon">⏳</div>
            <div>正在连接后端服务...</div>
          </div>
        ) : files.length === 0 ? (
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
