import { FileService } from '../file';

// Mock wsService
jest.mock('../websocket', () => ({
  wsService: {
    send: jest.fn(),
    on: jest.fn(),
    off: jest.fn(),
  },
}));

describe('FileService', () => {
  let fileService: FileService;

  beforeEach(() => {
    fileService = new FileService();
    jest.clearAllMocks();
  });

  describe('openFile', () => {
    it('should open file successfully', async () => {
      const mockResponse = {
        payload: {
          path: '/test/file.txt',
          size: 1024,
          file_type: 'text',
          encoding: 'utf-8',
        },
      };

      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue(mockResponse);

      const result = await fileService.openFile('/test/file.txt');
      expect(result).toEqual(mockResponse.payload);
      expect(wsService.send).toHaveBeenCalledWith('file:open', { path: '/test/file.txt' });
    });

    it('should throw error when open fails', async () => {
      const { wsService } = require('../websocket');
      wsService.send.mockRejectedValue(new Error('File not found'));

      await expect(fileService.openFile('/nonexistent/file.txt')).rejects.toThrow('File not found');
    });
  });

  describe('closeFile', () => {
    it('should close file successfully', async () => {
      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue({});

      await fileService.closeFile('/test/file.txt');
      expect(wsService.send).toHaveBeenCalledWith('file:close', { path: '/test/file.txt' });
    });
  });

  describe('readContent', () => {
    it('should read content successfully', async () => {
      const mockResponse = {
        payload: {
          lines: ['line1', 'line2'],
          start_line: 0,
          end_line: 2,
          total_lines: 100,
          has_more: true,
        },
      };

      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue(mockResponse);

      const result = await fileService.readContent('/test/file.txt', 0, 100);
      expect(result).toEqual(mockResponse.payload);
      expect(wsService.send).toHaveBeenCalledWith('file:content', {
        path: '/test/file.txt',
        start_line: 0,
        num_lines: 100,
      });
    });
  });

  describe('detectEncoding', () => {
    it('should detect encoding successfully', async () => {
      const mockResponse = {
        payload: { encoding: 'utf-8' },
      };

      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue(mockResponse);

      const result = await fileService.detectEncoding('/test/file.txt');
      expect(result).toBe('utf-8');
      expect(wsService.send).toHaveBeenCalledWith('encoding:detect', { path: '/test/file.txt' });
    });
  });

  describe('cancelSearch', () => {
    it('should cancel search successfully', async () => {
      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue({});

      await fileService.cancelSearch();
      expect(wsService.send).toHaveBeenCalledWith('search:cancel', {});
    });
  });

  describe('formatJSON', () => {
    it('should format JSON successfully', async () => {
      const mockResponse = {
        payload: { formatted: '{\n  "key": "value"\n}' },
      };

      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue(mockResponse);

      const result = await fileService.formatJSON('{"key":"value"}');
      expect(result).toBe('{\n  "key": "value"\n}');
      expect(wsService.send).toHaveBeenCalledWith('format:json', { data: '{"key":"value"}' });
    });
  });

  describe('formatXML', () => {
    it('should format XML successfully', async () => {
      const mockResponse = {
        payload: { formatted: '<root>\n  <item/>\n</root>' },
      };

      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue(mockResponse);

      const result = await fileService.formatXML('<root><item/></root>');
      expect(result).toBe('<root>\n  <item/>\n</root>');
      expect(wsService.send).toHaveBeenCalledWith('format:xml', { data: '<root><item/></root>' });
    });
  });

  describe('connectRemote', () => {
    it('should connect remote successfully', async () => {
      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue({});

      await fileService.connectRemote('ssh-1', {
        type: 'ssh',
        ssh: { host: 'localhost', port: 22, username: 'test' },
      });
      expect(wsService.send).toHaveBeenCalledWith('remote:connect', {
        id: 'ssh-1',
        config: {
          type: 'ssh',
          ssh: { host: 'localhost', port: 22, username: 'test' },
        },
      });
    });
  });

  describe('disconnectRemote', () => {
    it('should disconnect remote successfully', async () => {
      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue({});

      await fileService.disconnectRemote('ssh-1');
      expect(wsService.send).toHaveBeenCalledWith('remote:disconnect', { id: 'ssh-1' });
    });
  });

  describe('listRemoteConnections', () => {
    it('should list remote connections successfully', async () => {
      const mockResponse = {
        payload: { connections: ['ssh-1', 'ssh-2'] },
      };

      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue(mockResponse);

      const result = await fileService.listRemoteConnections();
      expect(result).toEqual(['ssh-1', 'ssh-2']);
    });
  });

  describe('execRemoteCommand', () => {
    it('should execute remote command successfully', async () => {
      const mockResponse = {
        payload: { output: 'command output' },
      };

      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue(mockResponse);

      const result = await fileService.execRemoteCommand('ssh-1', 'ls -la');
      expect(result).toBe('command output');
      expect(wsService.send).toHaveBeenCalledWith('remote:exec', { id: 'ssh-1', cmd: 'ls -la' });
    });
  });

  describe('registerAutoSave', () => {
    it('should register auto save successfully', async () => {
      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue({});

      await fileService.registerAutoSave('session-1', '/test/file.txt');
      expect(wsService.send).toHaveBeenCalledWith('autosave:save', {
        id: 'session-1',
        file_path: '/test/file.txt',
      });
    });
  });

  describe('restoreAutoSave', () => {
    it('should restore auto save successfully', async () => {
      const mockResponse = {
        payload: {
          id: 'session-1',
          file_path: '/test/file.txt',
          cursor_line: 10,
          cursor_column: 5,
        },
      };

      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue(mockResponse);

      const result = await fileService.restoreAutoSave('session-1');
      expect(result).toEqual(mockResponse.payload);
    });

    it('should return null when restore fails', async () => {
      const { wsService } = require('../websocket');
      wsService.send.mockRejectedValue(new Error('Session not found'));

      const result = await fileService.restoreAutoSave('nonexistent');
      expect(result).toBeNull();
    });
  });

  describe('updateAutoSave', () => {
    it('should update auto save successfully', async () => {
      const { wsService } = require('../websocket');
      wsService.send.mockResolvedValue({});

      await fileService.updateAutoSave('session-1', 10, 5, 100.5, 50.2);
      expect(wsService.send).toHaveBeenCalledWith('autosave:update', {
        id: 'session-1',
        cursor_line: 10,
        cursor_column: 5,
        scroll_top: 100.5,
        scroll_left: 50.2,
      });
    });
  });

  describe('onFileUpdate', () => {
    it('should register file update handler', () => {
      const handler = jest.fn();
      fileService.onFileUpdate(handler);
      // No error should be thrown
    });
  });

  describe('offFileUpdate', () => {
    it('should unregister file update handler', () => {
      const handler = jest.fn();
      fileService.onFileUpdate(handler);
      fileService.offFileUpdate(handler);
      // No error should be thrown
    });
  });
});
