import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { FileInfo, ReadResult, SearchResult } from '../../shared/types';
import { wsService } from '../../services/websocket';

interface EditorProps {
  file: FileInfo | null;
  onTextSelect: (text: string) => void;
  onFileModified?: (modified: boolean) => void;
}

export const Editor: React.FC<EditorProps> = ({
  file,
  onTextSelect,
  onFileModified,
}) => {
  const [lines, setLines] = useState<string[]>([]);
  const [totalLines, setTotalLines] = useState<number>(0);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [replaceQuery, setReplaceQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [currentResultIndex, setCurrentResultIndex] = useState(0);
  const [followMode, setFollowMode] = useState(false);
  const [selectedText, setSelectedText] = useState('');
  const [highlightedLines, setHighlightedLines] = useState<Set<number>>(new Set());
  const containerRef = useRef<HTMLDivElement>(null);
  const lastLineRef = useRef<number>(0);
  const loadingRef = useRef(false);

  const loadContent = useCallback(async (startLine: number, numLines: number) => {
    if (!file || loadingRef.current) return;

    loadingRef.current = true;
    setLoading(true);
    try {
      const response = await wsService.send('file:content', {
        path: file.path,
        start_line: startLine,
        num_lines: numLines,
      });

      const result: ReadResult = response.payload;
      setLines(prev => {
        if (startLine === 0) {
          return result.lines;
        }
        const newLines = [...prev];
        result.lines.forEach((line, i) => {
          newLines[startLine + i] = line;
        });
        return newLines;
      });
      setTotalLines(result.total_lines);
      lastLineRef.current = result.end_line;
    } catch (error) {
      console.error('Failed to load content:', error);
    } finally {
      setLoading(false);
      loadingRef.current = false;
    }
  }, [file]);

  useEffect(() => {
    if (file) {
      setLines([]);
      setTotalLines(0);
      lastLineRef.current = 0;
      loadingRef.current = false;
      loadContent(0, 100);
    }
  }, [file, loadContent]);

  useEffect(() => {
    if (!file) return;

    const handleFileUpdate = (info: FileInfo) => {
      if (followMode && containerRef.current) {
        const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
        const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;

        if (isAtBottom) {
          loadContent(lastLineRef.current, 50);
        }
      }
    };

    wsService.on('file:update', handleFileUpdate);

    return () => {
      wsService.off('file:update', handleFileUpdate);
    };
  }, [file, followMode, loadContent]);

  const handleScroll = useCallback(() => {
    if (!containerRef.current || loading || loadingRef.current) return;

    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    if (scrollHeight - scrollTop - clientHeight < 100) {
      loadContent(lastLineRef.current, 50);
    }
  }, [loading, loadContent]);

  const handleSearch = async () => {
    if (!file || !searchQuery) return;

    try {
      const results: SearchResult[] = [];
      const handler = (msg: any) => {
        results.push(msg.payload);
      };

      wsService.on('search:result', handler);

      await wsService.send('search:start', {
        path: file.path,
        pattern: searchQuery,
        is_regex: false,
        case_sensitive: false,
      });

      wsService.off('search:result', handler);

      setSearchResults(results);
      setCurrentResultIndex(0);

      const lineNumbers = new Set<number>();
      results.forEach(r => lineNumbers.add(r.line));
      setHighlightedLines(lineNumbers);
    } catch (error) {
      console.error('Search failed:', error);
    }
  };

  const handleReplace = async () => {
    if (!file || !searchQuery) return;

    try {
      await wsService.send('search:replace', {
        path: file.path,
        pattern: searchQuery,
        replace: replaceQuery,
        is_regex: false,
        case_sensitive: false,
      });

      loadContent(0, lines.length);
      setSearchResults([]);
      setHighlightedLines(new Set());
    } catch (error) {
      console.error('Replace failed:', error);
    }
  };

  const handleReplaceAll = async () => {
    if (!file || !searchQuery) return;

    try {
      await wsService.send('search:replace', {
        path: file.path,
        pattern: searchQuery,
        replace: replaceQuery,
        is_regex: false,
        case_sensitive: false,
      });

      loadContent(0, lines.length);
      setSearchResults([]);
      setHighlightedLines(new Set());
    } catch (error) {
      console.error('Replace all failed:', error);
    }
  };

  const handleNextResult = () => {
    if (searchResults.length === 0) return;
    setCurrentResultIndex(prev => (prev + 1) % searchResults.length);
  };

  const handlePrevResult = () => {
    if (searchResults.length === 0) return;
    setCurrentResultIndex(prev => (prev - 1 + searchResults.length) % searchResults.length);
  };

  const handleCancelSearch = async () => {
    try {
      await wsService.send('search:cancel', {});
      setSearchResults([]);
      setHighlightedLines(new Set());
    } catch (error) {
      console.error('Cancel search failed:', error);
    }
  };

  const handleTextSelection = () => {
    const selection = window.getSelection();
    if (selection && selection.toString()) {
      setSelectedText(selection.toString());
      onTextSelect(selection.toString());
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'f' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      const searchInput = document.querySelector('.search-input') as HTMLInputElement;
      searchInput?.focus();
    }
    if (e.key === 'h' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      const replaceInput = document.querySelector('.replace-input') as HTMLInputElement;
      replaceInput?.focus();
    }
    if (e.key === 'Escape') {
      handleCancelSearch();
    }
  };

  const highlightText = (text: string, lineIndex: number) => {
    if (!searchQuery || searchResults.length === 0) {
      return text;
    }

    const lineResults = searchResults.filter(r => r.line === lineIndex);
    if (lineResults.length === 0) {
      return text;
    }

    const parts: React.ReactNode[] = [];
    let lastIndex = 0;

    lineResults.sort((a, b) => a.column - b.column);

    lineResults.forEach((result, i) => {
      if (result.column > lastIndex) {
        parts.push(text.slice(lastIndex, result.column));
      }

      const isCurrent = searchResults.indexOf(result) === currentResultIndex;
      parts.push(
        <span
          key={i}
          className={isCurrent ? 'search-highlight-current' : 'search-highlight'}
        >
          {text.slice(result.column, result.column + result.length)}
        </span>
      );
      lastIndex = result.column + result.length;
    });

    if (lastIndex < text.length) {
      parts.push(text.slice(lastIndex));
    }

    return parts;
  };

  const highlightSelectedText = (text: string) => {
    if (!selectedText || selectedText.length < 2) {
      return text;
    }

    const parts: React.ReactNode[] = [];
    let lastIndex = 0;
    const lowerText = text.toLowerCase();
    const lowerSelected = selectedText.toLowerCase();

    let idx = 0;
    while (idx < text.length) {
      const pos = lowerText.indexOf(lowerSelected, idx);
      if (pos === -1) break;

      if (pos > lastIndex) {
        parts.push(text.slice(lastIndex, pos));
      }

      parts.push(
        <span key={`sel-${pos}`} className="selected-text-highlight">
          {text.slice(pos, pos + selectedText.length)}
        </span>
      );
      lastIndex = pos + selectedText.length;
      idx = lastIndex;
    }

    if (lastIndex < text.length) {
      parts.push(text.slice(lastIndex));
    }

    return parts.length > 0 ? parts : text;
  };

  const renderLine = (line: string, index: number) => {
    const isHighlighted = highlightedLines.has(index);
    const displayLine = highlightText(line, index);
    const finalLine = typeof displayLine === 'string'
      ? highlightSelectedText(displayLine)
      : displayLine;

    return (
      <div key={index} className={`line ${isHighlighted ? 'line-highlighted' : ''}`}>
        <span className="line-number">{index + 1}</span>
        <span className="line-content">{finalLine}</span>
      </div>
    );
  };

  const renderBinaryView = (data: string) => {
    const bytes = new TextEncoder().encode(data);
    const hexLines: string[] = [];
    const textLines: string[] = [];

    for (let i = 0; i < bytes.length; i += 16) {
      const chunk = bytes.slice(i, i + 16);
      const hex = Array.from(chunk)
        .map(b => b.toString(16).padStart(2, '0'))
        .join(' ');
      hexLines.push(hex);

      const text = Array.from(chunk)
        .map(b => (b >= 32 && b <= 126 ? String.fromCharCode(b) : '.'))
        .join('');
      textLines.push(text);
    }

    return (
      <div className="binary-view">
        <div className="hex-column">
          {hexLines.map((line, i) => (
            <div key={i}>{line}</div>
          ))}
        </div>
        <div className="text-column">
          {textLines.map((line, i) => (
            <div key={i}>{line}</div>
          ))}
        </div>
      </div>
    );
  };

  return (
    <div className="editor-container" onKeyDown={handleKeyDown}>
      <div className="editor-toolbar">
        <button
          className={`toolbar-button ${followMode ? 'active' : ''}`}
          onClick={() => setFollowMode(!followMode)}
        >
          {followMode ? 'Following' : 'Follow'}
        </button>
        <div className="search-bar">
          <input
            type="text"
            className="search-input"
            placeholder="Search..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
          />
          <input
            type="text"
            className="replace-input"
            placeholder="Replace..."
            value={replaceQuery}
            onChange={(e) => setReplaceQuery(e.target.value)}
          />
          <button className="toolbar-button" onClick={handleSearch}>Find</button>
          <button className="toolbar-button" onClick={handleReplace}>Replace</button>
          <button className="toolbar-button" onClick={handleReplaceAll}>All</button>
          <button className="toolbar-button" onClick={handlePrevResult}>↑</button>
          <button className="toolbar-button" onClick={handleNextResult}>↓</button>
          <button className="toolbar-button" onClick={handleCancelSearch}>✕</button>
          {searchResults.length > 0 && (
            <span className="search-count">{currentResultIndex + 1}/{searchResults.length}</span>
          )}
        </div>
      </div>

      <div
        ref={containerRef}
        className="editor-content"
        onScroll={handleScroll}
        onMouseUp={handleTextSelection}
      >
        {file?.file_type === 'binary' ? (
          renderBinaryView(lines.join('\n'))
        ) : (
          lines.map((line, index) => renderLine(line, index))
        )}
        {loading && <div className="loading-indicator">Loading...</div>}
      </div>
    </div>
  );
};
