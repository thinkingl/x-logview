import React, { useState } from 'react';
import { FileInfo } from '../../shared/types';
import { wsService } from '../../services/websocket';

interface StatusBarProps {
  file: FileInfo | null;
  connected: boolean;
}

const ENCODINGS = [
  'utf-8',
  'utf-8-bom',
  'utf-16-le',
  'utf-16-be',
  'gbk',
  'gb2312',
  'big5',
  'shift-jis',
  'euc-jp',
  'euc-kr',
  'iso-8859-1',
];

export const StatusBar: React.FC<StatusBarProps> = ({ file, connected }) => {
  const [showEncodingMenu, setShowEncodingMenu] = useState(false);

  const handleEncodingChange = async (newEncoding: string) => {
    if (!file || file.encoding === newEncoding) return;

    try {
      await wsService.send('encoding:convert', {
        path: file.path,
        from: file.encoding,
        to: newEncoding,
      });
      // 重新加载文件
      await wsService.send('file:open', { path: file.path });
    } catch (error) {
      console.error('Failed to convert encoding:', error);
    }
    setShowEncodingMenu(false);
  };

  return (
    <div className="status-bar">
      <div className="status-item">
        <span className={`status-dot ${connected ? '' : 'disconnected'}`} />
        <span>{connected ? 'Connected' : 'Disconnected'}</span>
      </div>

      {file && (
        <>
          <div className="status-item encoding-selector">
            <span
              className="encoding-badge"
              onClick={() => setShowEncodingMenu(!showEncodingMenu)}
              style={{ cursor: 'pointer' }}
            >
              {file.encoding} ▼
            </span>
            {showEncodingMenu && (
              <div className="encoding-menu">
                {ENCODINGS.map(enc => (
                  <div
                    key={enc}
                    className={`encoding-menu-item ${enc === file.encoding ? 'active' : ''}`}
                    onClick={() => handleEncodingChange(enc)}
                  >
                    {enc}
                  </div>
                ))}
              </div>
            )}
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
