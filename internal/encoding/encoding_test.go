package encoding

import (
	"bytes"
	"testing"
)

func TestDetectEncoding(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected EncodingType
	}{
		// UTF-8 tests
		{
			name:     "ENC-001: 纯ASCII文本",
			data:     []byte("Hello World"),
			expected: UTF8,
		},
		{
			name:     "ENC-002: 中文UTF-8文本",
			data:     []byte("你好世界"),
			expected: UTF8,
		},
		{
			name:     "ENC-003: 混合字符",
			data:     []byte("Hello 你好"),
			expected: UTF8,
		},
		{
			name:     "ENC-004: 空数据",
			data:     []byte(""),
			expected: UTF8,
		},
		// UTF-8 BOM tests
		{
			name:     "ENC-010: UTF-8 BOM文件",
			data:     []byte{0xEF, 0xBB, 0xBF, 0x48, 0x65, 0x6C, 0x6C, 0x6F},
			expected: UTF8BOM,
		},
		{
			name:     "ENC-011: 仅BOM头",
			data:     []byte{0xEF, 0xBB, 0xBF},
			expected: UTF8BOM,
		},
		// UTF-16 tests
		{
			name:     "ENC-020: UTF-16 LE",
			data:     []byte{0xFF, 0xFE, 0x48, 0x00},
			expected: UTF16LE,
		},
		{
			name:     "ENC-021: UTF-16 BE",
			data:     []byte{0xFE, 0xFF, 0x00, 0x48},
			expected: UTF16BE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectEncoding(tt.data)
			if result != tt.expected {
				t.Errorf("DetectEncoding() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertEncoding(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		from    EncodingType
		to      EncodingType
		wantErr bool
	}{
		// ENC-042: 同编码转换
		{
			name:    "ENC-042: UTF-8转UTF-8",
			data:    []byte("Hello"),
			from:    UTF8,
			to:      UTF8,
			wantErr: false,
		},
		// ENC-044: 空数据转换
		{
			name:    "ENC-044: 空数据转换",
			data:    []byte{},
			from:    UTF8,
			to:      GBK,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertEncoding(tt.data, tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertEncoding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("ConvertEncoding() returned nil result")
			}
		})
	}
}

func TestUTF16Decoding(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "UTF-16 LE ASCII",
			data:     []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00},
			expected: "Hello",
		},
		{
			name:     "UTF-16 BE ASCII",
			data:     []byte{0x00, 0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F},
			expected: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeUTF16(tt.data, tt.name == "UTF-16 LE ASCII")
			if result != tt.expected {
				t.Errorf("decodeUTF16() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestUTF16Encoding(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		littleEndian bool
		expected   []byte
	}{
		{
			name:         "UTF-16 LE Hello",
			text:         "Hello",
			littleEndian: true,
			expected:     []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00},
		},
		{
			name:         "UTF-16 BE Hello",
			text:         "Hello",
			littleEndian: false,
			expected:     []byte{0x00, 0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeUTF16(tt.text, tt.littleEndian)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("encodeUTF16() = %v, want %v", result, tt.expected)
			}
		})
	}
}
