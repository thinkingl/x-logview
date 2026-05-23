import React from 'react';
import { FileInfo } from '../../shared/types';

interface Tab {
  id: string;
  file: FileInfo;
  modified: boolean;
}

interface TabBarProps {
  tabs: Tab[];
  activeTabId: string | null;
  onTabSelect: (id: string) => void;
  onTabClose: (id: string) => void;
}

export const TabBar: React.FC<TabBarProps> = ({
  tabs,
  activeTabId,
  onTabSelect,
  onTabClose,
}) => {
  if (tabs.length === 0) {
    return null;
  }

  const getFileName = (path: string): string => {
    return path.split('/').pop() || path.split('\\').pop() || path;
  };

  return (
    <div className="tab-bar">
      {tabs.map((tab) => (
        <div
          key={tab.id}
          className={`tab ${tab.id === activeTabId ? 'active' : ''}`}
          onClick={() => onTabSelect(tab.id)}
        >
          <span className="tab-icon">📄</span>
          <span className="tab-name">
            {getFileName(tab.file.path)}
            {tab.modified && <span className="tab-modified">●</span>}
          </span>
          <button
            className="tab-close"
            onClick={(e) => {
              e.stopPropagation();
              onTabClose(tab.id);
            }}
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
};
