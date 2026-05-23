package encoding

import (
	"bytes"
	"io"
	"strings"
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

func TestIsGBK(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "GBK数据",
			data:     []byte{0xC4, 0xE3, 0xBA, 0xC3}, // 你好 GBK
			expected: true,
		},
		{
			name:     "纯ASCII",
			data:     []byte("Hello"),
			expected: false,
		},
		{
			name:     "空数据",
			data:     []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGBK(tt.data)
			if result != tt.expected {
				t.Errorf("isGBK() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsBig5(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "Big5数据",
			data:     []byte{0xA4, 0xA4, 0xA4, 0xE5}, // 你好 Big5
			expected: true,
		},
		{
			name:     "纯ASCII",
			data:     []byte("Hello"),
			expected: false,
		},
		{
			name:     "空数据",
			data:     []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBig5(tt.data)
			if result != tt.expected {
				t.Errorf("isBig5() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsShiftJIS(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "ShiftJIS数据",
			data:     []byte{0x82, 0xA0, 0x82, 0xA2}, // あい ShiftJIS
			expected: true,
		},
		{
			name:     "纯ASCII",
			data:     []byte("Hello"),
			expected: false,
		},
		{
			name:     "空数据",
			data:     []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isShiftJIS(tt.data)
			if result != tt.expected {
				t.Errorf("isShiftJIS() = %v, want %v", result, tt.expected)
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

func TestConvertEncodingUTF8ToGBK(t *testing.T) {
	data := []byte("Hello")
	result, err := ConvertEncoding(data, UTF8, GBK)
	if err != nil {
		t.Fatalf("ConvertEncoding() error = %v", err)
	}
	if result == nil {
		t.Error("ConvertEncoding() returned nil result")
	}
	if len(result) == 0 {
		t.Error("ConvertEncoding() returned empty result")
	}
}

func TestConvertEncodingGBKToUTF8(t *testing.T) {
	// GBK "你好"
	data := []byte{0xC4, 0xE3, 0xBA, 0xC3}
	result, err := ConvertEncoding(data, GBK, UTF8)
	if err != nil {
		t.Fatalf("ConvertEncoding() error = %v", err)
	}
	if result == nil {
		t.Error("ConvertEncoding() returned nil result")
	}
	expected := "你好"
	if string(result) != expected {
		t.Errorf("ConvertEncoding() = %v, want %v", string(result), expected)
	}
}

func TestConvertEncodingUTF16ToUTF8(t *testing.T) {
	// UTF-16 LE "Hello"
	data := []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00}
	result, err := ConvertEncoding(data, UTF16LE, UTF8)
	if err != nil {
		t.Fatalf("ConvertEncoding() error = %v", err)
	}
	if string(result) != "Hello" {
		t.Errorf("ConvertEncoding() = %v, want Hello", string(result))
	}
}

func TestConvertEncodingUTF8ToUTF8BOM(t *testing.T) {
	data := []byte("Hello")
	result, err := ConvertEncoding(data, UTF8, UTF8BOM)
	if err != nil {
		t.Fatalf("ConvertEncoding() error = %v", err)
	}
	if !bytes.HasPrefix(result, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("ConvertEncoding() result should have BOM prefix")
	}
	if string(result[3:]) != "Hello" {
		t.Errorf("ConvertEncoding() content = %v, want Hello", string(result[3:]))
	}
}

func TestConvertEncodingUTF8BOMToUTF8(t *testing.T) {
	data := []byte{0xEF, 0xBB, 0xBF, 0x48, 0x65, 0x6C, 0x6C, 0x6F}
	result, err := ConvertEncoding(data, UTF8BOM, UTF8)
	if err != nil {
		t.Fatalf("ConvertEncoding() error = %v", err)
	}
	if string(result) != "Hello" {
		t.Errorf("ConvertEncoding() = %v, want Hello", string(result))
	}
}

func TestNewDecodedReader(t *testing.T) {
	tests := []struct {
		name string
		data string
		enc  EncodingType
	}{
		{
			name: "UTF-8 Reader",
			data: "Hello World",
			enc:  UTF8,
		},
		{
			name: "GBK Reader",
			data: "Hello World",
			enc:  GBK,
		},
		{
			name: "UTF-16 LE Reader",
			data: "Hello World",
			enc:  UTF16LE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.data)
			decoded := NewDecodedReader(reader, tt.enc)
			if decoded == nil {
				t.Error("NewDecodedReader() returned nil")
			}

			data, err := io.ReadAll(decoded)
			if err != nil {
				t.Errorf("ReadAll() error = %v", err)
			}
			if len(data) == 0 {
				t.Error("ReadAll() returned empty data")
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

func TestDecodeBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		enc      EncodingType
		expected string
	}{
		{
			name:     "UTF-8 decode",
			data:     []byte("Hello"),
			enc:      UTF8,
			expected: "Hello",
		},
		{
			name:     "UTF-8 BOM decode",
			data:     []byte{0xEF, 0xBB, 0xBF, 0x48, 0x65, 0x6C, 0x6C, 0x6F},
			enc:      UTF8BOM,
			expected: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeBytes(tt.data, tt.enc)
			if err != nil {
				t.Errorf("decodeBytes() error = %v", err)
			}
			if result != tt.expected {
				t.Errorf("decodeBytes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEncodeBytes(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		enc      EncodingType
		expected []byte
	}{
		{
			name:     "UTF-8 encode",
			text:     "Hello",
			enc:      UTF8,
			expected: []byte("Hello"),
		},
		{
			name:     "UTF-8 BOM encode",
			text:     "Hello",
			enc:      UTF8BOM,
			expected: append([]byte{0xEF, 0xBB, 0xBF}, []byte("Hello")...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encodeBytes(tt.text, tt.enc)
			if err != nil {
				t.Errorf("encodeBytes() error = %v", err)
			}
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("encodeBytes() = %v, want %v", result, tt.expected)
			}
		})
	}
}
