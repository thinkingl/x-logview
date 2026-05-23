import React, { useMemo } from 'react';
import { FileInfo } from '../../shared/types';

interface RightPanelProps {
  selectedText: string;
  file: FileInfo | null;
}

export const RightPanel: React.FC<RightPanelProps> = ({ selectedText, file }) => {
  const numberInfo = useMemo(() => {
    if (!selectedText) return null;

    const num = Number(selectedText);
    if (isNaN(num)) return null;

    const hex = num.toString(16).toUpperCase();
    const isTimestamp = num > 946684800 && num < 4102444800;
    const isMsTimestamp = num > 946684800000 && num < 4102444800000;

    let timestamp = null;
    if (isTimestamp) {
      timestamp = new Date(num * 1000).toISOString();
    } else if (isMsTimestamp) {
      timestamp = new Date(num).toISOString();
    }

    return {
      decimal: num,
      hex: `0x${hex}`,
      timestamp,
      isTimestamp,
      isMsTimestamp,
    };
  }, [selectedText]);

  return (
    <div className="right-panel">
      <div className="panel-header">
        <span className="panel-title">Details</span>
      </div>

      <div className="panel-content">
        {file && (
          <div className="info-section">
            <div className="info-label">File Info</div>
            <div className="info-value">
              <div>Path: {file.path}</div>
              <div>Size: {formatSize(file.size)}</div>
              <div>Type: {file.file_type}</div>
              <div>Encoding: <span className="encoding-badge">{file.encoding}</span></div>
              <div>Lines: {file.total_lines >= 0 ? file.total_lines : '...'}</div>
            </div>
          </div>
        )}

        {selectedText && (
          <div className="info-section">
            <div className="info-label">Selected Text</div>
            <div className="info-value" style={{ wordBreak: 'break-all' }}>
              {selectedText}
            </div>
          </div>
        )}

        {numberInfo && (
          <div className="info-section">
            <div className="info-label">Number Info</div>
            <div className="info-value">
              <div>Decimal: {numberInfo.decimal}</div>
              <div>Hex: {numberInfo.hex}</div>
              {numberInfo.timestamp && (
                <div>
                  <div>Time ({numberInfo.isMsTimestamp ? 'ms' : 's'}):</div>
                  <div style={{ color: '#4ec9b0' }}>{numberInfo.timestamp}</div>
                </div>
              )}
            </div>
          </div>
        )}

        {!selectedText && !numberInfo && (
          <div className="empty-state">
            <div>Select text to see details</div>
          </div>
        )}
      </div>
    </div>
  );
};

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}
