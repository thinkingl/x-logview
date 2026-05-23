package format

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
)

type FormatService struct{}

func NewFormatService() *FormatService {
	return &FormatService{}
}

func (fs *FormatService) FormatJSON(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return buf.Bytes(), nil
}

func (fs *FormatService) MinifyJSON(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return buf.Bytes(), nil
}

func (fs *FormatService) ValidateJSON(data []byte) error {
	var v interface{}
	return json.Unmarshal(data, &v)
}

func (fs *FormatService) FormatXML(data []byte) ([]byte, error) {
	b := &bytes.Buffer{}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	var tokens []xml.Token
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		tokens = append(tokens, token)
	}

	encoder := xml.NewEncoder(b)
	encoder.Indent("", "  ")

	for _, token := range tokens {
		if err := encoder.EncodeToken(token); err != nil {
			return nil, fmt.Errorf("invalid XML: %w", err)
		}
	}

	if err := encoder.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (fs *FormatService) MinifyXML(data []byte) ([]byte, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false

	var buf bytes.Buffer
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			buf.WriteString("<" + t.Name.Local)
			for _, attr := range t.Attr {
				buf.WriteString(" " + attr.Name.Local + "=\"" + attr.Value + "\"")
			}
			buf.WriteString(">")
		case xml.EndElement:
			buf.WriteString("</" + t.Name.Local + ">")
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" {
				buf.WriteString(text)
			}
		case xml.Comment:
			buf.WriteString("<!--" + string(t) + "-->")
		case xml.ProcInst:
			buf.WriteString("<?" + t.Target + " " + string(t.Inst) + "?>")
		}
	}

	return buf.Bytes(), nil
}

func (fs *FormatService) ValidateXML(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
	}
}
