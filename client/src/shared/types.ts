export interface FileInfo {
  path: string;
  size: number;
  mod_time: string;
  file_type: 'text' | 'binary';
  encoding: string;
  total_lines: number;
  loaded: boolean;
}

export interface ReadResult {
  lines: string[];
  start_line: number;
  end_line: number;
  total_lines: number;
  has_more: boolean;
}

export interface SearchResult {
  line: number;
  column: number;
  length: number;
  match: string;
  context: string;
  offset: number;
}

export interface ReplaceResult {
  replaced: number;
  content: string;
}

export interface SessionState {
  id: string;
  file: FileInfo;
  editor: EditorState;
  changes: Change[];
  temp_file: string;
  created_at: string;
  updated_at: string;
}

export interface EditorState {
  cursor_position: CursorPosition;
  scroll_position: ScrollPosition;
  viewport: Viewport;
}

export interface CursorPosition {
  line: number;
  column: number;
}

export interface ScrollPosition {
  top: number;
  left: number;
}

export interface Viewport {
  width: number;
  height: number;
}

export interface Change {
  content: string;
  timestamp: string;
}

export interface WebSocketMessage {
  id: string;
  type: string;
  payload: any;
  timestamp: number;
}

export interface BufferConfig {
  initial_size: number;
  max_size: number;
  chunk_size: number;
  max_chunks: number;
}

export interface SSHConfig {
  host: string;
  port: number;
  username: string;
  password?: string;
  key_file?: string;
  passphrase?: string;
  timeout?: number;
}

export interface WSLConfig {
  distro?: string;
  shell?: string;
  timeout?: number;
}

export interface RemoteConfig {
  type: 'ssh' | 'wsl';
  ssh?: SSHConfig;
  wsl?: WSLConfig;
}

export interface AutoSaveState {
  id: string;
  file_path: string;
  cursor_line: number;
  cursor_column: number;
  scroll_top: number;
  scroll_left: number;
  unsaved_changes: Record<string, Uint8Array>;
  last_saved: string;
  created: string;
  metadata: Record<string, any>;
}
