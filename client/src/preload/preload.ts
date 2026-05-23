import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  openFile: () => ipcRenderer.invoke('open-file'),
  getAppPath: () => ipcRenderer.invoke('get-app-path'),
  onFileUpdate: (callback: (data: any) => void) => {
    ipcRenderer.on('file-update', (_event, data) => callback(data));
  },
  removeAllListeners: (channel: string) => {
    ipcRenderer.removeAllListeners(channel);
  },
});
