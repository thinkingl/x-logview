package search

import (
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
)

type SearchRequest struct {
	Pattern       string `json:"pattern"`
	IsRegex       bool   `json:"is_regex"`
	CaseSensitive bool   `json:"case_sensitive"`
	StartOffset   int64  `json:"start_offset"`
	EndOffset     int64  `json:"end_offset"`
}

type SearchResult struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Length   int    `json:"length"`
	Match    string `json:"match"`
	Context  string `json:"context"`
	Offset   int64  `json:"offset"`
}

type SearchService struct {
	mu      sync.RWMutex
	cancel  context.CancelFunc
}

func NewSearchService() *SearchService {
	return &SearchService{}
}

func (ss *SearchService) Search(ctx context.Context, path string, req SearchRequest, resultChan chan<- SearchResult) error {
	ss.mu.Lock()
	searchCtx, cancel := context.WithCancel(ctx)
	ss.cancel = cancel
	ss.mu.Unlock()

	defer cancel()

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var re *regexp.Regexp
	if req.IsRegex {
		flags := ""
		if !req.CaseSensitive {
			flags = "(?i)"
		}
		re, err = regexp.Compile(flags + req.Pattern)
		if err != nil {
			return err
		}
	}

	buf := make([]byte, 64*1024)
	var remainder []byte
	lineNum := 0
	offset := int64(0)

	for {
		select {
		case <-searchCtx.Done():
			return nil
		default:
		}

		n, err := f.Read(buf)
		if n > 0 {
			data := append(remainder, buf[:n]...)

			lines := strings.Split(string(data), "\n")
			remainder = []byte(lines[len(lines)-1])

			for i := 0; i < len(lines)-1; i++ {
				line := lines[i]
				lineOffset := offset - int64(len(remainder)) + int64(i)*int64(len(line)+1)

				if req.EndOffset > 0 && lineOffset > req.EndOffset {
					return nil
				}

				if re != nil {
					matches := re.FindAllStringIndex(line, -1)
					for _, match := range matches {
						resultChan <- SearchResult{
							Line:    lineNum,
							Column:  match[0],
							Length:  match[1] - match[0],
							Match:   line[match[0]:match[1]],
							Context: line,
							Offset:  lineOffset,
						}
					}
				} else {
					searchStr := line
					pattern := req.Pattern
					if !req.CaseSensitive {
						searchStr = strings.ToLower(searchStr)
						pattern = strings.ToLower(pattern)
					}

					idx := 0
					for {
						pos := strings.Index(searchStr[idx:], pattern)
						if pos == -1 {
							break
						}
						resultChan <- SearchResult{
							Line:    lineNum,
							Column:  idx + pos,
							Length:  len(pattern),
							Match:   line[idx+pos : idx+pos+len(pattern)],
							Context: line,
							Offset:  lineOffset,
						}
						idx += pos + 1
					}
				}

				lineNum++
			}

			offset += int64(n)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	if len(remainder) > 0 {
		line := string(remainder)
		lineOffset := offset - int64(len(remainder))

		if re != nil {
			matches := re.FindAllStringIndex(line, -1)
			for _, match := range matches {
				resultChan <- SearchResult{
					Line:    lineNum,
					Column:  match[0],
					Length:  match[1] - match[0],
					Match:   line[match[0]:match[1]],
					Context: line,
					Offset:  lineOffset,
				}
			}
		} else {
			searchStr := line
			pattern := req.Pattern
			if !req.CaseSensitive {
				searchStr = strings.ToLower(searchStr)
				pattern = strings.ToLower(pattern)
			}

			idx := 0
			for {
				pos := strings.Index(searchStr[idx:], pattern)
				if pos == -1 {
					break
				}
				resultChan <- SearchResult{
					Line:    lineNum,
					Column:  idx + pos,
					Length:  len(pattern),
					Match:   line[idx+pos : idx+pos+len(pattern)],
					Context: line,
					Offset:  lineOffset,
				}
				idx += pos + 1
			}
		}
	}

	return nil
}

type ReplaceResult struct {
	Replaced int    `json:"replaced"`
	Content  string `json:"content"`
}

func (ss *SearchService) Replace(path, pattern, replace string, isRegex, caseSensitive bool) (*ReplaceResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	replaced := 0

	if isRegex {
		flags := ""
		if !caseSensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + pattern)
		if err != nil {
			return nil, err
		}

		newContent := re.ReplaceAllStringFunc(content, func(match string) string {
			replaced++
			return replace
		})
		content = newContent
	} else {
		searchStr := content
		searchPattern := pattern
		if !caseSensitive {
			searchStr = strings.ToLower(searchStr)
			searchPattern = strings.ToLower(searchPattern)
		}

		for {
			idx := strings.Index(searchStr, searchPattern)
			if idx == -1 {
				break
			}
			content = content[:idx] + replace + content[idx+len(pattern):]
			searchStr = strings.ToLower(content)
			replaced++
		}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}

	return &ReplaceResult{
		Replaced: replaced,
		Content:  content,
	}, nil
}

func (ss *SearchService) Cancel() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.cancel != nil {
		ss.cancel()
	}
}
