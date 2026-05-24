import React, { useState, useEffect } from 'react';

interface AppConfig {
  buffer: {
    initial_size: number;
    max_size: number;
    chunk_size: number;
    max_chunks: number;
  };
  session: {
    auto_save: boolean;
    auto_save_interval: number;
    max_backups: number;
    restore_on_start: boolean;
  };
  editor: {
    font_size: number;
    tab_size: number;
    show_line_numbers: boolean;
    word_wrap: boolean;
    theme: string;
  };
  server: {
    port: number;
    auto_start: boolean;
    timeout: number;
  };
}

const defaultConfig: AppConfig = {
  buffer: {
    initial_size: 64 * 1024,
    max_size: 256 * 1024 * 1024,
    chunk_size: 4 * 1024,
    max_chunks: 1000,
  },
  session: {
    auto_save: true,
    auto_save_interval: 30,
    max_backups: 5,
    restore_on_start: true,
  },
  editor: {
    font_size: 14,
    tab_size: 4,
    show_line_numbers: true,
    word_wrap: false,
    theme: 'opencode',
  },
  server: {
    port: 8090,
    auto_start: true,
    timeout: 30,
  },
};

interface SettingsProps {
  isOpen: boolean;
  onClose: () => void;
}

export const Settings: React.FC<SettingsProps> = ({ isOpen, onClose }) => {
  const [config, setConfig] = useState<AppConfig>(defaultConfig);
  const [activeTab, setActiveTab] = useState('editor');
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      const saved = localStorage.getItem('x-logview-config');
      if (saved) {
        setConfig(JSON.parse(saved));
      }
    } catch (error) {
      console.error('Failed to load config:', error);
    }
  };

  const saveConfig = async () => {
    try {
      localStorage.setItem('x-logview-config', JSON.stringify(config));
      setHasChanges(false);
    } catch (error) {
      console.error('Failed to save config:', error);
    }
  };

  const updateConfig = (path: string, value: any) => {
    const keys = path.split('.');
    const newConfig = { ...config };
    let current: any = newConfig;

    for (let i = 0; i < keys.length - 1; i++) {
      current[keys[i]] = { ...current[keys[i]] };
      current = current[keys[i]];
    }

    current[keys[keys.length - 1]] = value;
    setConfig(newConfig);
    setHasChanges(true);
  };

  const formatSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  };

  const parseSize = (str: string): number => {
    const match = str.match(/^(\d+(?:\.\d+)?)\s*(KB|MB|GB|B)$/i);
    if (!match) return parseInt(str, 10);

    const value = parseFloat(match[1]);
    const unit = match[2].toUpperCase();

    switch (unit) {
      case 'KB': return Math.round(value * 1024);
      case 'MB': return Math.round(value * 1024 * 1024);
      case 'GB': return Math.round(value * 1024 * 1024 * 1024);
      default: return Math.round(value);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="settings-overlay">
      <div className="settings-modal">
        <div className="settings-header">
          <h2>设置</h2>
          <button className="close-button" onClick={onClose}>×</button>
        </div>

        <div className="settings-content">
          <div className="settings-sidebar">
            <button
              className={`settings-tab ${activeTab === 'editor' ? 'active' : ''}`}
              onClick={() => setActiveTab('editor')}
            >
              编辑器
            </button>
            <button
              className={`settings-tab ${activeTab === 'buffer' ? 'active' : ''}`}
              onClick={() => setActiveTab('buffer')}
            >
              文件缓冲
            </button>
            <button
              className={`settings-tab ${activeTab === 'session' ? 'active' : ''}`}
              onClick={() => setActiveTab('session')}
            >
              会话管理
            </button>
            <button
              className={`settings-tab ${activeTab === 'server' ? 'active' : ''}`}
              onClick={() => setActiveTab('server')}
            >
              后端服务
            </button>
          </div>

          <div className="settings-body">
            {activeTab === 'editor' && (
              <div className="settings-section">
                <h3>编辑器设置</h3>

                <div className="setting-item">
                  <label>主题</label>
                  <select
                    value={config.editor.theme}
                    onChange={(e) => updateConfig('editor.theme', e.target.value)}
                  >
                    <option value="opencode">OpenCode</option>
                    <option value="dark">Dark</option>
                    <option value="light">Light</option>
                  </select>
                </div>

                <div className="setting-item">
                  <label>字体大小</label>
                  <div className="setting-input">
                    <input
                      type="number"
                      min="10"
                      max="32"
                      value={config.editor.font_size}
                      onChange={(e) => updateConfig('editor.font_size', parseInt(e.target.value))}
                    />
                    <span>px</span>
                  </div>
                </div>

                <div className="setting-item">
                  <label>Tab 大小</label>
                  <div className="setting-input">
                    <input
                      type="number"
                      min="1"
                      max="8"
                      value={config.editor.tab_size}
                      onChange={(e) => updateConfig('editor.tab_size', parseInt(e.target.value))}
                    />
                    <span>空格</span>
                  </div>
                </div>

                <div className="setting-item">
                  <label>显示行号</label>
                  <input
                    type="checkbox"
                    checked={config.editor.show_line_numbers}
                    onChange={(e) => updateConfig('editor.show_line_numbers', e.target.checked)}
                  />
                </div>

                <div className="setting-item">
                  <label>自动换行</label>
                  <input
                    type="checkbox"
                    checked={config.editor.word_wrap}
                    onChange={(e) => updateConfig('editor.word_wrap', e.target.checked)}
                  />
                </div>
              </div>
            )}

            {activeTab === 'buffer' && (
              <div className="settings-section">
                <h3>文件缓冲设置</h3>
                <p className="setting-description">
                  调整缓冲区大小可以影响大文件的加载性能和内存使用。
                </p>

                <div className="setting-item">
                  <label>初始缓冲区大小</label>
                  <div className="setting-input">
                    <input
                      type="text"
                      value={formatSize(config.buffer.initial_size)}
                      onChange={(e) => updateConfig('buffer.initial_size', parseSize(e.target.value))}
                    />
                  </div>
                </div>

                <div className="setting-item">
                  <label>最大缓冲区大小</label>
                  <div className="setting-input">
                    <input
                      type="text"
                      value={formatSize(config.buffer.max_size)}
                      onChange={(e) => updateConfig('buffer.max_size', parseSize(e.target.value))}
                    />
                  </div>
                </div>

                <div className="setting-item">
                  <label>块大小</label>
                  <div className="setting-input">
                    <input
                      type="text"
                      value={formatSize(config.buffer.chunk_size)}
                      onChange={(e) => updateConfig('buffer.chunk_size', parseSize(e.target.value))}
                    />
                  </div>
                </div>

                <div className="setting-item">
                  <label>最大缓存块数</label>
                  <div className="setting-input">
                    <input
                      type="number"
                      min="100"
                      max="10000"
                      value={config.buffer.max_chunks}
                      onChange={(e) => updateConfig('buffer.max_chunks', parseInt(e.target.value))}
                    />
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'session' && (
              <div className="settings-section">
                <h3>会话管理设置</h3>

                <div className="setting-item">
                  <label>启动时恢复文件</label>
                  <input
                    type="checkbox"
                    checked={config.session.restore_on_start}
                    onChange={(e) => updateConfig('session.restore_on_start', e.target.checked)}
                  />
                </div>

                <div className="setting-item">
                  <label>自动保存</label>
                  <input
                    type="checkbox"
                    checked={config.session.auto_save}
                    onChange={(e) => updateConfig('session.auto_save', e.target.checked)}
                  />
                </div>

                <div className="setting-item">
                  <label>自动保存间隔</label>
                  <div className="setting-input">
                    <input
                      type="number"
                      min="5"
                      max="300"
                      value={config.session.auto_save_interval}
                      onChange={(e) => updateConfig('session.auto_save_interval', parseInt(e.target.value))}
                    />
                    <span>秒</span>
                  </div>
                </div>

                <div className="setting-item">
                  <label>最大备份数</label>
                  <div className="setting-input">
                    <input
                      type="number"
                      min="1"
                      max="50"
                      value={config.session.max_backups}
                      onChange={(e) => updateConfig('session.max_backups', parseInt(e.target.value))}
                    />
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'server' && (
              <div className="settings-section">
                <h3>后端服务设置</h3>

                <div className="setting-item">
                  <label>自动启动后端</label>
                  <input
                    type="checkbox"
                    checked={config.server.auto_start}
                    onChange={(e) => updateConfig('server.auto_start', e.target.checked)}
                  />
                </div>

                <div className="setting-item">
                  <label>后端端口</label>
                  <div className="setting-input">
                    <input
                      type="number"
                      min="1024"
                      max="65535"
                      value={config.server.port}
                      onChange={(e) => updateConfig('server.port', parseInt(e.target.value))}
                    />
                  </div>
                </div>

                <div className="setting-item">
                  <label>启动超时</label>
                  <div className="setting-input">
                    <input
                      type="number"
                      min="5"
                      max="120"
                      value={config.server.timeout}
                      onChange={(e) => updateConfig('server.timeout', parseInt(e.target.value))}
                    />
                    <span>秒</span>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        <div className="settings-footer">
          <button className="btn-cancel" onClick={onClose}>取消</button>
          <button
            className="btn-save"
            onClick={saveConfig}
            disabled={!hasChanges}
          >
            保存
          </button>
        </div>
      </div>
    </div>
  );
};
