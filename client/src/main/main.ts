import { app, BrowserWindow, ipcMain, dialog, shell, Menu, MenuItemConstructorOptions } from 'electron';
import * as path from 'path';
import { spawn, ChildProcess } from 'child_process';
import * as fs from 'fs';
import * as net from 'net';
import * as http from 'http';

let mainWindow: BrowserWindow | null = null;
let backendProcess: ChildProcess | null = null;
let backendReady = false;
const BACKEND_PORT = 8090;
const BACKEND_HEALTH_URL = `http://localhost:${BACKEND_PORT}/health`;

// 设置应用名称
app.setName('x-logview');

// 获取后端可执行文件路径
function getBackendPath(): string {
  const isDev = process.env.NODE_ENV === 'development';
  if (isDev) {
    return path.join(__dirname, '..', '..', '..', 'bin', 'x-logview-server');
  }
  return path.join(process.resourcesPath, 'bin', 'x-logview-server');
}

// 检查端口是否被占用
function isPortInUse(port: number): Promise<boolean> {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once('error', () => resolve(true));
    server.once('listening', () => {
      server.close();
      resolve(false);
    });
    server.listen(port);
  });
}

// 检查后端健康状态
function checkBackendHealth(): Promise<boolean> {
  return new Promise((resolve) => {
    const options = {
      hostname: '127.0.0.1',
      port: BACKEND_PORT,
      path: '/health',
      method: 'GET',
      timeout: 3000,
    };

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        console.log(`Health check response: ${res.statusCode}`);
        resolve(res.statusCode === 200);
      });
    });

    req.on('error', (e) => {
      console.log(`Health check error: ${e.message}`);
      resolve(false);
    });

    req.on('timeout', () => {
      console.log('Health check timeout');
      req.destroy();
      resolve(false);
    });

    req.end();
  });
}

// 等待后端就绪
async function waitForBackend(maxWait: number = 30000): Promise<boolean> {
  const startTime = Date.now();
  console.log('Waiting for backend to be ready...');
  while (Date.now() - startTime < maxWait) {
    const healthy = await checkBackendHealth();
    console.log(`Backend health check: ${healthy} (${Date.now() - startTime}ms)`);
    if (healthy) {
      return true;
    }
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
  console.log('Backend health check timeout');
  return false;
}

// 启动后端服务
async function startBackend(): Promise<boolean> {
  const backendPath = getBackendPath();

  if (!fs.existsSync(backendPath)) {
    console.error('Backend executable not found:', backendPath);
    return false;
  }

  if (await isPortInUse(BACKEND_PORT)) {
    console.log('Port already in use, checking if backend is running...');
    if (await checkBackendHealth()) {
      console.log('Backend is already running');
      backendReady = true;
      return true;
    }
    console.log('Port in use but backend not healthy, will try to start anyway');
  }

  console.log('Starting backend:', backendPath);

  try {
    backendProcess = spawn(backendPath, ['-port', String(BACKEND_PORT)], {
      stdio: ['ignore', 'pipe', 'pipe'],
      detached: false,
    });

    backendProcess.on('error', (error) => {
      console.error('Backend spawn error:', error);
      backendReady = false;
      handleBackendExit(1);
    });

    backendProcess.on('exit', (code, signal) => {
      console.log(`Backend exited with code ${code}, signal ${signal}`);
      backendReady = false;
      handleBackendExit(code);
    });

    const ready = await waitForBackend();
    if (ready) {
      backendReady = true;
      console.log('Backend started successfully');
      return true;
    } else {
      console.error('Backend failed to start within timeout');
      return false;
    }
  } catch (error) {
    console.error('Failed to start backend:', error);
    return false;
  }
}

// 处理后端退出
async function handleBackendExit(code: number | null) {
  if (!mainWindow || mainWindow.isDestroyed()) {
    return;
  }

  const result = await dialog.showMessageBox(mainWindow, {
    type: 'warning',
    title: '后端服务已停止',
    message: '后端服务意外退出',
    detail: `退出代码: ${code ?? '未知'}\n\n是否重新启动后端服务？`,
    buttons: ['重新启动', '退出程序'],
    defaultId: 0,
    cancelId: 1,
  });

  if (result.response === 0) {
    console.log('User chose to restart backend');
    const started = await startBackend();
    if (started) {
      mainWindow?.webContents.send('backend-restarted');
    } else {
      await handleBackendExit(1);
    }
  } else {
    console.log('User chose to exit');
    app.quit();
  }
}

// 停止后端服务
function stopBackend(): Promise<void> {
  return new Promise((resolve) => {
    if (!backendProcess || backendProcess.killed) {
      console.log('No backend process to stop');
      resolve();
      return;
    }

    console.log('Stopping backend process...', backendProcess.pid);

    // 监听进程退出事件
    const onExit = (code: number | null, signal: string | null) => {
      console.log(`Backend process exited with code ${code}, signal ${signal}`);
      cleanup();
      resolve();
    };

    const cleanup = () => {
      backendProcess?.removeListener('exit', onExit);
      backendProcess = null;
      backendReady = false;
    };

    backendProcess.on('exit', onExit);

    // 发送 SIGTERM 信号
    backendProcess.kill('SIGTERM');

    // 设置超时，如果进程没有在 5 秒内退出，则强制杀死
    setTimeout(() => {
      if (backendProcess && !backendProcess.killed) {
        console.log('Backend process did not exit, sending SIGKILL...');
        backendProcess.kill('SIGKILL');
      }
    }, 5000);

    // 设置最大等待时间
    setTimeout(() => {
      console.log('Timeout waiting for backend to exit, force resolving');
      cleanup();
      resolve();
    }, 8000);
  });
}

// 创建菜单
function createMenu() {
  const template: MenuItemConstructorOptions[] = [
    {
      label: '文件',
      submenu: [
        {
          label: '新建',
          accelerator: 'CmdOrCtrl+N',
          click: () => {
            mainWindow?.webContents.send('menu-new-file');
          },
        },
        {
          label: '打开...',
          accelerator: 'CmdOrCtrl+O',
          click: () => {
            mainWindow?.webContents.send('menu-open-file');
          },
        },
        { type: 'separator' },
        {
          label: '保存',
          accelerator: 'CmdOrCtrl+S',
          click: () => {
            mainWindow?.webContents.send('menu-save');
          },
        },
        {
          label: '另存为...',
          accelerator: 'CmdOrCtrl+Shift+S',
          click: () => {
            mainWindow?.webContents.send('menu-save-as');
          },
        },
        { type: 'separator' },
        {
          label: '关闭标签页',
          accelerator: 'CmdOrCtrl+W',
          click: () => {
            mainWindow?.webContents.send('menu-close-tab');
          },
        },
        { type: 'separator' },
        { role: 'quit', label: '退出' },
      ],
    },
    {
      label: '编辑',
      submenu: [
        { role: 'undo', label: '撤销' },
        { role: 'redo', label: '重做' },
        { type: 'separator' },
        { role: 'cut', label: '剪切' },
        { role: 'copy', label: '复制' },
        { role: 'paste', label: '粘贴' },
        { role: 'selectAll', label: '全选' },
      ],
    },
    {
      label: '视图',
      submenu: [
        {
          label: '重新加载',
          accelerator: 'CmdOrCtrl+R',
          click: () => {
            mainWindow?.webContents.send('menu-reload');
          },
        },
        {
          label: '强制重新加载',
          accelerator: 'CmdOrCtrl+Shift+R',
          click: () => {
            mainWindow?.webContents.reloadIgnoringCache();
          },
        },
        { type: 'separator' },
        { role: 'toggleDevTools', label: '开发者工具' },
        { type: 'separator' },
        { role: 'resetZoom', label: '重置缩放' },
        { role: 'zoomIn', label: '放大' },
        { role: 'zoomOut', label: '缩小' },
        { type: 'separator' },
        { role: 'togglefullscreen', label: '全屏' },
      ],
    },
    {
      label: '窗口',
      submenu: [
        { role: 'minimize', label: '最小化' },
        { role: 'zoom', label: '缩放' },
        { role: 'close', label: '关闭' },
      ],
    },
    {
      label: '帮助',
      submenu: [
        {
          label: '关于 x-logview',
          click: async () => {
            await dialog.showMessageBox(mainWindow!, {
              type: 'info',
              title: '关于 x-logview',
              message: 'x-logview',
              detail: '万能的跨平台日志查看工具\n版本: 1.0.0',
              buttons: ['确定'],
            });
          },
        },
        {
          label: '检查更新',
          click: async () => {
            shell.openExternal('https://github.com/thinkingl/x-logview/releases');
          },
        },
      ],
    },
  ];

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 800,
    minHeight: 600,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
    titleBarStyle: 'hiddenInset',
    trafficLightPosition: { x: 10, y: 10 },
    backgroundColor: '#1e1e1e',
    title: 'x-logview',
  });

  if (process.env.NODE_ENV === 'development') {
    mainWindow.loadURL('http://localhost:5173');
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, '..', 'renderer', 'index.html'));
  }

  mainWindow.on('close', (e) => {
    e.preventDefault();
    
    console.log('Window closing, stopping backend...');
    
    if (backendProcess && !backendProcess.killed) {
      const onExit = () => {
        console.log('Backend process exited');
        mainWindow?.destroy();
      };
      
      backendProcess.on('exit', onExit);
      backendProcess.kill('SIGTERM');
      
      setTimeout(() => {
        if (backendProcess && !backendProcess.killed) {
          console.log('Backend did not exit, sending SIGKILL');
          backendProcess.kill('SIGKILL');
        }
        mainWindow?.destroy();
      }, 3000);
    } else {
      mainWindow?.destroy();
    }
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // 创建菜单
  createMenu();
}

// IPC 处理
ipcMain.handle('open-file', async () => {
  if (!mainWindow) return null;

  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openFile'],
    filters: [
      { name: 'All Files', extensions: ['*'] },
      { name: 'Log Files', extensions: ['log', 'txt'] },
      { name: 'JSON Files', extensions: ['json'] },
      { name: 'XML Files', extensions: ['xml'] },
    ],
  });

  if (result.canceled) {
    return null;
  }

  return result.filePaths[0];
});

ipcMain.handle('new-file', () => {
  return true;
});

ipcMain.handle('save-file', async (_, filePath: string, content: string) => {
  try {
    fs.writeFileSync(filePath, content, 'utf-8');
    return { success: true };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('save-file-as', async (_, content: string) => {
  if (!mainWindow) return null;

  const result = await dialog.showSaveDialog(mainWindow, {
    filters: [
      { name: 'Text Files', extensions: ['txt', 'log'] },
      { name: 'JSON Files', extensions: ['json'] },
      { name: 'XML Files', extensions: ['xml'] },
      { name: 'All Files', extensions: ['*'] },
    ],
  });

  if (result.canceled) {
    return null;
  }

  try {
    fs.writeFileSync(result.filePath, content, 'utf-8');
    return { path: result.filePath, success: true };
  } catch (error: any) {
    return { success: false, error: error.message };
  }
});

ipcMain.handle('get-app-path', () => {
  const userDataPath = app.getPath('userData');
  // 同时保存到 localStorage 供前端使用
  mainWindow?.webContents.executeJavaScript(`
    localStorage.setItem('x-logview-user-data-path', '${userDataPath}');
  `);
  return userDataPath;
});

ipcMain.handle('check-backend', async () => {
  return await checkBackendHealth();
});

ipcMain.handle('restart-backend', async () => {
  stopBackend();
  return await startBackend();
});

ipcMain.handle('get-backend-status', () => {
  return backendReady;
});

// 应用启动
app.whenReady().then(async () => {
  console.log('Starting backend service...');
  const started = await startBackend();

  if (!started) {
    const result = await dialog.showMessageBox({
      type: 'error',
      title: '启动失败',
      message: '后端服务启动失败',
      detail: '请检查端口 8090 是否被占用，或手动启动后端服务。',
      buttons: ['继续启动前端', '退出程序'],
      defaultId: 0,
      cancelId: 1,
    });

    if (result.response === 1) {
      app.quit();
      return;
    }
  }

  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', async () => {
  await stopBackend();
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', async () => {
  await stopBackend();
});
