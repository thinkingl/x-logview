import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  // 文件操作
  openFile: () => ipcRenderer.invoke('open-file'),
  newFile: () => ipcRenderer.invoke('new-file'),
  saveFile: (filePath: string, content: string) => ipcRenderer.invoke('save-file', filePath, content),
  saveFileAs: (content: string) => ipcRenderer.invoke('save-file-as', content),
  
  // 应用信息
  getAppPath: () => ipcRenderer.invoke('get-app-path'),
  
  // 后端服务
  checkBackend: () => ipcRenderer.invoke('check-backend'),
  restartBackend: () => ipcRenderer.invoke('restart-backend'),
  getBackendStatus: () => ipcRenderer.invoke('get-backend-status'),
  
  // 事件监听
  onBackendRestarted: (callback: () => void) => {
    ipcRenderer.on('backend-restarted', () => callback());
  },
  onFileUpdate: (callback: (data: any) => void) => {
    ipcRenderer.on('file-update', (_event, data) => callback(data));
  },
  
  // 菜单事件
  onMenuNewFile: (callback: () => void) => {
    ipcRenderer.on('menu-new-file', () => callback());
  },
  onMenuOpenFile: (callback: () => void) => {
    ipcRenderer.on('menu-open-file', () => callback());
  },
  onMenuSave: (callback: () => void) => {
    ipcRenderer.on('menu-save', () => callback());
  },
  onMenuSaveAs: (callback: () => void) => {
    ipcRenderer.on('menu-save-as', () => callback());
  },
  onMenuCloseTab: (callback: () => void) => {
    ipcRenderer.on('menu-close-tab', () => callback());
  },
  onMenuReload: (callback: () => void) => {
    ipcRenderer.on('menu-reload', () => callback());
  },
  
  removeAllListeners: (channel: string) => {
    ipcRenderer.removeAllListeners(channel);
  },
});
