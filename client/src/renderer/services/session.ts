import { wsService } from './websocket';

export interface SessionData {
  id: string;
  name: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  files: FileState[];
}

export interface FileState {
  id?: number;
  session_id: string;
  file_path: string;
  isUntitled: boolean;
  content: string;
  cursor_line: number;
  cursor_col: number;
  scroll_top: number;
  scroll_left: number;
  is_active: boolean;
  edit_history: EditEntry[];
  created_at?: string;
  updated_at?: string;
}

export interface EditEntry {
  timestamp: string;
  content: string;
}

class SessionService {
  private currentSession: SessionData | null = null;

  async getActiveSession(): Promise<SessionData | null> {
    try {
      const response = await wsService.send('session:get', {});
      this.currentSession = response.payload;
      return this.currentSession;
    } catch (error) {
      console.error('Failed to get session:', error);
      return null;
    }
  }

  async createSession(id: string, name: string): Promise<void> {
    await wsService.send('session:create', { id, name });
  }

  async updateSession(id: string, name: string): Promise<void> {
    await wsService.send('session:update', { id, name });
  }

  async addFile(sessionId: string, file: FileState): Promise<void> {
    await wsService.send('session:addFile', { session_id: sessionId, file });
  }

  async removeFile(sessionId: string, filePath: string): Promise<void> {
    await wsService.send('session:removeFile', { session_id: sessionId, file_path: filePath });
  }

  async updateFile(sessionId: string, filePath: string, file: FileState): Promise<void> {
    await wsService.send('session:updateFile', { 
      session_id: sessionId, 
      file_path: filePath, 
      file 
    });
  }

  async setActiveFile(sessionId: string, filePath: string): Promise<void> {
    await wsService.send('session:setActive', { 
      session_id: sessionId, 
      file_path: filePath 
    });
  }

  async addEditHistory(sessionId: string, filePath: string, entry: EditEntry): Promise<void> {
    await wsService.send('session:addEditHistory', { 
      session_id: sessionId, 
      file_path: filePath, 
      entry 
    });
  }

  getCurrentSession(): SessionData | null {
    return this.currentSession;
  }

  getCurrentSessionId(): string | null {
    return this.currentSession?.id || null;
  }
}

export const sessionService = new SessionService();
