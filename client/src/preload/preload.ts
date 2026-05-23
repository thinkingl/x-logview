import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  openFile: () => ipcRenderer.invoke('open-file'),
  getAppPath: () => ipcRenderer.invoke('get-app-path'),
  checkBackend: () => ipcRenderer.invoke('check-backend'),
  restartBackend: () => ipcRenderer.invoke('restart-backend'),
  getBackendStatus: () => ipcRenderer.invoke('get-backend-status'),
  onBackendRestarted: (callback: () => void) => {
    ipcRenderer.on('backend-restarted', () => callback());
  },
  onFileUpdate: (callback: (data: any) => void) => {
    ipcRenderer.on('file-update', (_event, data) => callback(data));
  },
  removeAllListeners: (channel: string) => {
    ipcRenderer.removeAllListeners(channel);
  },
});
