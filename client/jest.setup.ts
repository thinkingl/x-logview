// Jest setup file

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  url: string;
  readyState: number = MockWebSocket.CONNECTING;
  onopen: ((event: any) => void) | null = null;
  onclose: ((event: any) => void) | null = null;
  onmessage: ((event: any) => void) | null = null;
  onerror: ((event: any) => void) | null = null;

  constructor(url: string) {
    this.url = url;
  }

  send(data: string | ArrayBuffer | ArrayBufferView): void {
    // Mock send
  }

  close(code?: number, reason?: string): void {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose({ code, reason });
    }
  }
}

// Mock global WebSocket
(global as any).WebSocket = MockWebSocket;

// Mock fetch
(global as any).fetch = jest.fn(() =>
  Promise.resolve({
    ok: true,
    json: () => Promise.resolve({}),
    text: () => Promise.resolve(''),
    status: 200,
    headers: new Headers(),
  })
);

// Mock electronAPI
Object.defineProperty(window, 'electronAPI', {
  value: {
    openFile: jest.fn(),
    getAppPath: jest.fn(),
    checkBackend: jest.fn(),
    restartBackend: jest.fn(),
    getBackendStatus: jest.fn(),
    onBackendRestarted: jest.fn(),
    onFileUpdate: jest.fn(),
    removeAllListeners: jest.fn(),
  },
  writable: true,
});
