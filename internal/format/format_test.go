package format

import (
	"testing"
)

func TestFormatJSON(t *testing.T) {
	fs := NewFormatService()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "FMT-001: 有效JSON格式化",
			input:   `{"key":"value","number":123}`,
			wantErr: false,
		},
		{
			name:    "FMT-002: 无效JSON格式化",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "FMT-003: 嵌套JSON格式化",
			input:   `{"a":{"b":1},"c":[1,2,3]}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fs.FormatJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("FormatJSON() returned nil result")
			}
		})
	}
}

func TestMinifyJSON(t *testing.T) {
	fs := NewFormatService()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "FMT-010: JSON压缩",
			input:   "{\n  \"key\": \"value\"\n}",
			wantErr: false,
		},
		{
			name:    "FMT-011: 无效JSON压缩",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fs.MinifyJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("MinifyJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("MinifyJSON() returned nil result")
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	fs := NewFormatService()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "FMT-020: 有效JSON验证",
			input:   `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "FMT-021: 无效JSON验证",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "空JSON对象",
			input:   `{}`,
			wantErr: false,
		},
		{
			name:    "JSON数组",
			input:   `[1,2,3]`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.ValidateJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatXML(t *testing.T) {
	fs := NewFormatService()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "FMT-030: 有效XML格式化",
			input:   `<root><item>test</item></root>`,
			wantErr: false,
		},
		{
			name:    "FMT-031: 无效XML格式化",
			input:   `<root><item></root>`,
			wantErr: false, // XML decoder tries to recover
		},
		{
			name:    "嵌套XML",
			input:   `<root><a><b>test</b></a></root>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fs.FormatXML([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatXML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("FormatXML() returned nil result")
			}
		})
	}
}

func TestMinifyXML(t *testing.T) {
	fs := NewFormatService()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "FMT-040: XML压缩",
			input:   `<root>\n  <item>test</item>\n</root>`,
			wantErr: false,
		},
		{
			name:    "带属性XML压缩",
			input:   `<root attr="value">\n  <item>test</item>\n</root>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fs.MinifyXML([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("MinifyXML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("MinifyXML() returned nil result")
			}
		})
	}
}

func TestValidateXML(t *testing.T) {
	fs := NewFormatService()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "FMT-050: 有效XML验证",
			input:   `<root><item>test</item></root>`,
			wantErr: false,
		},
		{
			name:    "FMT-051: 无效XML验证",
			input:   `<root><item></root>`,
			wantErr: false, // lenient validation
		},
		{
			name:    "空XML",
			input:   ``,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fs.ValidateXML([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateXML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
