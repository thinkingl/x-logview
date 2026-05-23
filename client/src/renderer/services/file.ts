import { wsService } from './websocket';
import { FileInfo, ReadResult, SearchResult, ReplaceResult, RemoteConfig, AutoSaveState } from '../shared/types';

export class FileService {
  async openFile(path: string): Promise<FileInfo> {
    const response = await wsService.send('file:open', { path });
    return response.payload;
  }

  async closeFile(path: string): Promise<void> {
    await wsService.send('file:close', { path });
  }

  async readContent(path: string, startLine: number, numLines: number): Promise<ReadResult> {
    const response = await wsService.send('file:content', {
      path,
      start_line: startLine,
      num_lines: numLines,
    });
    return response.payload;
  }

  async detectEncoding(path: string): Promise<string> {
    const response = await wsService.send('encoding:detect', { path });
    return response.payload.encoding;
  }

  async search(path: string, pattern: string, isRegex: boolean, caseSensitive: boolean): Promise<SearchResult[]> {
    const results: SearchResult[] = [];

    const handler = (msg: any) => {
      results.push(msg.payload);
    };

    wsService.on('search:result', handler);

    await wsService.send('search:start', {
      path,
      pattern,
      is_regex: isRegex,
      case_sensitive: caseSensitive,
    });

    wsService.off('search:result', handler);

    return results;
  }

  async searchReplace(
    path: string,
    pattern: string,
    replace: string,
    isRegex: boolean,
    caseSensitive: boolean
  ): Promise<ReplaceResult> {
    const response = await wsService.send('search:replace', {
      path,
      pattern,
      replace,
      is_regex: isRegex,
      case_sensitive: caseSensitive,
    });
    return response.payload;
  }

  async cancelSearch(): Promise<void> {
    await wsService.send('search:cancel', {});
  }

  async formatJSON(data: string): Promise<string> {
    const response = await wsService.send('format:json', { data });
    return response.payload.formatted;
  }

  async formatXML(data: string): Promise<string> {
    const response = await wsService.send('format:xml', { data });
    return response.payload.formatted;
  }

  async connectRemote(id: string, config: RemoteConfig): Promise<void> {
    await wsService.send('remote:connect', { id, config });
  }

  async disconnectRemote(id: string): Promise<void> {
    await wsService.send('remote:disconnect', { id });
  }

  async listRemoteConnections(): Promise<string[]> {
    const response = await wsService.send('remote:list', {});
    return response.payload.connections;
  }

  async execRemoteCommand(id: string, cmd: string): Promise<string> {
    const response = await wsService.send('remote:exec', { id, cmd });
    return response.payload.output;
  }

  async registerAutoSave(id: string, filePath: string): Promise<void> {
    await wsService.send('autosave:save', { id, file_path: filePath });
  }

  async restoreAutoSave(id: string): Promise<AutoSaveState | null> {
    try {
      const response = await wsService.send('autosave:restore', { id });
      return response.payload;
    } catch {
      return null;
    }
  }

  async updateAutoSave(
    id: string,
    cursorLine: number,
    cursorColumn: number,
    scrollTop: number,
    scrollLeft: number
  ): Promise<void> {
    await wsService.send('autosave:update', {
      id,
      cursor_line: cursorLine,
      cursor_column: cursorColumn,
      scroll_top: scrollTop,
      scroll_left: scrollLeft,
    });
  }

  onFileUpdate(callback: (info: FileInfo) => void) {
    wsService.on('file:update', (msg) => callback(msg.payload));
  }

  offFileUpdate(callback: (info: FileInfo) => void) {
    wsService.off('file:update', (msg) => callback(msg.payload));
  }
}

export const fileService = new FileService();
