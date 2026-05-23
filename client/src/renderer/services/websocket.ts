import { WebSocketMessage } from '../shared/types';

type MessageHandler = (msg: WebSocketMessage) => void;

export class WebSocketService {
  private ws: WebSocket | null = null;
  private url: string;
  private handlers: Map<string, MessageHandler[]> = new Map();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectDelay = 2000;
  private isConnecting = false;
  private shouldReconnect = true;

  constructor(url: string = 'ws://127.0.0.1:8090/ws') {
    this.url = url;
  }

  connect(): Promise<void> {
    if (this.isConnecting) {
      return Promise.resolve();
    }

    this.isConnecting = true;

    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
          console.log('WebSocket connected');
          this.reconnectAttempts = 0;
          this.isConnecting = false;
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const msg: WebSocketMessage = JSON.parse(event.data);
            this.handleMessage(msg);
          } catch (e) {
            console.error('Failed to parse message:', e);
          }
        };

        this.ws.onclose = () => {
          console.log('WebSocket closed');
          this.isConnecting = false;
          if (this.shouldReconnect) {
            this.attemptReconnect();
          }
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          this.isConnecting = false;
          // 不 reject，让 onclose 处理重连
        };
      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  disconnect() {
    this.shouldReconnect = false;
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  reconnect() {
    this.shouldReconnect = true;
    this.reconnectAttempts = 0;
    this.connect().catch(() => {});
  }

  send(type: string, payload: any): Promise<WebSocketMessage> {
    return new Promise((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        reject(new Error('WebSocket not connected'));
        return;
      }

      const id = this.generateId();
      const msg: WebSocketMessage = {
        id,
        type,
        payload,
        timestamp: Date.now(),
      };

      const responseHandler = (response: WebSocketMessage) => {
        if (response.id === id) {
          this.off(type, responseHandler);
          if (response.type === 'error') {
            reject(new Error(response.payload.message));
          } else {
            resolve(response);
          }
        }
      };

      this.on(type, responseHandler);

      this.ws.send(JSON.stringify(msg));
    });
  }

  on(type: string, handler: MessageHandler) {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, []);
    }
    this.handlers.get(type)!.push(handler);
  }

  off(type: string, handler: MessageHandler) {
    const handlers = this.handlers.get(type);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    }
  }

  private handleMessage(msg: WebSocketMessage) {
    const handlers = this.handlers.get(msg.type);
    if (handlers) {
      handlers.forEach((handler) => handler(msg));
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = Math.min(this.reconnectDelay * this.reconnectAttempts, 30000);
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts}) in ${delay}ms...`);
      setTimeout(() => {
        if (this.shouldReconnect) {
          this.connect().catch(() => {});
        }
      }, delay);
    } else {
      console.error('Max reconnect attempts reached');
    }
  }

  private generateId(): string {
    return Math.random().toString(36).substring(2, 15);
  }
}

export const wsService = new WebSocketService();
