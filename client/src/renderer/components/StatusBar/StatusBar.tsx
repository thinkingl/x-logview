import React from 'react';
import { FileInfo } from '../../shared/types';

interface StatusBarProps {
  file: FileInfo | null;
  connected: boolean;
}

export const StatusBar: React.FC<StatusBarProps> = ({ file, connected }) => {
  return (
    <div className="status-bar">
      <div className="status-item">
        <span className={`status-dot ${connected ? '' : 'disconnected'}`} />
        <span>{connected ? 'Connected' : 'Disconnected'}</span>
      </div>

      {file && (
        <>
          <div className="status-item">
            <span className="encoding-badge">{file.encoding}</span>
          </div>

          <div className="status-item">
            <span>{file.file_type}</span>
          </div>

          <div className="status-item">
            <span>Lines: {file.total_lines >= 0 ? file.total_lines : '...'}</span>
          </div>

          <div className="status-item">
            <span>Size: {formatSize(file.size)}</span>
          </div>

          <div className="status-item" style={{ marginLeft: 'auto' }}>
            <span>{file.path}</span>
          </div>
        </>
      )}
    </div>
  );
};

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}
