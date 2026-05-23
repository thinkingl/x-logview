import { WebSocketMessage } from '../shared/types';

type MessageHandler = (msg: WebSocketMessage) => void;

export class WebSocketService {
  private ws: WebSocket | null = null;
  private url: string;
  private handlers: Map<string, MessageHandler[]> = new Map();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  constructor(url: string = 'ws://localhost:8090/ws') {
    this.url = url;
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
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
        this.attemptReconnect();
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        reject(error);
      };
    });
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
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
      setTimeout(() => {
        console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
        this.connect().catch(() => {});
      }, this.reconnectDelay * this.reconnectAttempts);
    }
  }

  private generateId(): string {
    return Math.random().toString(36).substring(2, 15);
  }
}

export const wsService = new WebSocketService();
