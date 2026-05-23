import { app, BrowserWindow, ipcMain, dialog, shell } from 'electron';
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

// 获取后端可执行文件路径
function getBackendPath(): string {
  const isDev = process.env.NODE_ENV === 'development';
  if (isDev) {
    // 开发模式：项目根目录下的 bin/x-logview-server
    return path.join(__dirname, '..', '..', '..', 'bin', 'x-logview-server');
  }
  // 生产模式：打包后的目录
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

// 检查后端健康状态（使用 http 模块）
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

  // 检查后端文件是否存在
  if (!fs.existsSync(backendPath)) {
    console.error('Backend executable not found:', backendPath);
    return false;
  }

  // 检查端口是否已被占用
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

    // 等待后端启动
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
    // 重新启动
    console.log('User chose to restart backend');
    const started = await startBackend();
    if (started) {
      // 通知前端重新连接
      mainWindow?.webContents.send('backend-restarted');
    } else {
      // 启动失败，再次询问
      await handleBackendExit(1);
    }
  } else {
    // 退出程序
    console.log('User chose to exit');
    app.quit();
  }
}

// 停止后端服务
function stopBackend() {
  if (backendProcess) {
    console.log('Stopping backend...');
    backendProcess.kill('SIGTERM');

    // 给进程一些时间优雅退出
    setTimeout(() => {
      if (backendProcess && !backendProcess.killed) {
        backendProcess.kill('SIGKILL');
      }
    }, 5000);
  }
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

  mainWindow.on('close', () => {
    // 停止后端服务
    stopBackend();
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
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

ipcMain.handle('get-app-path', () => {
  return app.getPath('userData');
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
  // 启动后端服务
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

  // 创建窗口
  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  stopBackend();
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', () => {
  stopBackend();
});
