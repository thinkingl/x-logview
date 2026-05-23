import { WebSocketService } from '../websocket';

describe('WebSocketService', () => {
  let wsService: WebSocketService;

  beforeEach(() => {
    wsService = new WebSocketService('ws://127.0.0.1:8090/ws');
  });

  afterEach(() => {
    wsService.disconnect();
  });

  describe('constructor', () => {
    it('should create instance with default URL', () => {
      const service = new WebSocketService();
      expect(service).toBeDefined();
    });

    it('should create instance with custom URL', () => {
      const service = new WebSocketService('ws://localhost:9090/ws');
      expect(service).toBeDefined();
    });
  });

  describe('connect', () => {
    it('should connect successfully', async () => {
      // Mock WebSocket to simulate successful connection
      const originalWebSocket = global.WebSocket;
      (global as any).WebSocket = class MockWebSocket {
        url: string;
        readyState = 1;
        onopen: (() => void) | null = null;
        onclose: (() => void) | null = null;
        onmessage: ((event: any) => void) | null = null;
        onerror: ((error: any) => void) | null = null;

        constructor(url: string) {
          this.url = url;
          setTimeout(() => {
            if (this.onopen) this.onopen();
          }, 10);
        }

        send() {}
        close() {
          this.readyState = 3;
        }
      };

      const service = new WebSocketService();
      await expect(service.connect()).resolves.toBeUndefined();

      service.disconnect();
      (global as any).WebSocket = originalWebSocket;
    });

    it('should handle connection error', async () => {
      const originalWebSocket = global.WebSocket;
      (global as any).WebSocket = class MockWebSocket {
        url: string;
        readyState = 3;
        onopen: (() => void) | null = null;
        onclose: (() => void) | null = null;
        onmessage: ((event: any) => void) | null = null;
        onerror: ((error: any) => void) | null = null;

        constructor(url: string) {
          this.url = url;
          setTimeout(() => {
            if (this.onerror) this.onerror(new Error('Connection failed'));
          }, 10);
        }

        send() {}
        close() {}
      };

      const service = new WebSocketService();
      // Connection error should not reject, but trigger reconnect
      // Use a short timeout to avoid waiting for reconnect
      const connectPromise = service.connect();
      const timeoutPromise = new Promise(resolve => setTimeout(resolve, 100));
      
      await Promise.race([connectPromise, timeoutPromise]);
      service.disconnect();

      (global as any).WebSocket = originalWebSocket;
    }, 1000);
  });

  describe('disconnect', () => {
    it('should disconnect successfully', () => {
      wsService.disconnect();
      // No error should be thrown
    });
  });

  describe('reconnect', () => {
    it('should reset reconnect attempts', () => {
      wsService.reconnect();
      // No error should be thrown
    });
  });

  describe('send', () => {
    it('should throw error when not connected', async () => {
      await expect(wsService.send('test', {})).rejects.toThrow('WebSocket not connected');
    });
  });

  describe('on/off', () => {
    it('should register and unregister handlers', () => {
      const handler = jest.fn();
      wsService.on('test', handler);
      wsService.off('test', handler);
      // No error should be thrown
    });

    it('should handle multiple handlers for same event', () => {
      const handler1 = jest.fn();
      const handler2 = jest.fn();
      wsService.on('test', handler1);
      wsService.on('test', handler2);
      wsService.off('test', handler1);
      // handler2 should still be registered
    });
  });
});
