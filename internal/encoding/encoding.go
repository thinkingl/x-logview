package encoding

import (
	"bytes"
	"io"
	"unicode/utf8"
	"unicode/utf16"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

type EncodingType string

const (
	UTF8      EncodingType = "utf-8"
	UTF8BOM   EncodingType = "utf-8-bom"
	UTF16LE   EncodingType = "utf-16-le"
	UTF16BE   EncodingType = "utf-16-be"
	GBK       EncodingType = "gbk"
	GB2312    EncodingType = "gb2312"
	BIG5      EncodingType = "big5"
	SHIFT_JIS EncodingType = "shift-jis"
	EUC_JP    EncodingType = "euc-jp"
	EUC_KR    EncodingType = "euc-kr"
	ISO88591  EncodingType = "iso-8859-1"
	UNKNOWN   EncodingType = "unknown"
)

func DetectEncoding(data []byte) EncodingType {
	if len(data) == 0 {
		return UTF8
	}

	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		return UTF8BOM
	}

	if len(data) >= 2 {
		if data[0] == 0xFF && data[1] == 0xFE {
			return UTF16LE
		}
		if data[0] == 0xFE && data[1] == 0xFF {
			return UTF16BE
		}
	}

	if utf8.Valid(data) {
		return UTF8
	}

	if isGBK(data) {
		return GBK
	}

	if isBig5(data) {
		return BIG5
	}

	if isShiftJIS(data) {
		return SHIFT_JIS
	}

	return ISO88591
}

func isGBK(data []byte) bool {
	for i := 0; i < len(data)-1; i++ {
		b1 := data[i]
		b2 := data[i+1]
		if b1 >= 0x81 && b1 <= 0xFE && b2 >= 0x40 && b2 <= 0xFE {
			return true
		}
	}
	return false
}

func isBig5(data []byte) bool {
	for i := 0; i < len(data)-1; i++ {
		b1 := data[i]
		b2 := data[i+1]
		if b1 >= 0x81 && b1 <= 0xFE && (b2 >= 0x40 && b2 <= 0x7E || b2 >= 0xA1 && b2 <= 0xFE) {
			return true
		}
	}
	return false
}

func isShiftJIS(data []byte) bool {
	for i := 0; i < len(data)-1; i++ {
		b1 := data[i]
		b2 := data[i+1]
		if (b1 >= 0x81 && b1 <= 0x9F || b1 >= 0xE0 && b1 <= 0xEF) && (b2 >= 0x40 && b2 <= 0x7E || b2 >= 0x80 && b2 <= 0xFC) {
			return true
		}
	}
	return false
}

func ConvertEncoding(data []byte, from, to EncodingType) ([]byte, error) {
	if from == to {
		return data, nil
	}

	text, err := decodeBytes(data, from)
	if err != nil {
		return nil, err
	}

	return encodeBytes(text, to)
}

func decodeBytes(data []byte, enc EncodingType) (string, error) {
	switch enc {
	case UTF8, UTF8BOM:
		if enc == UTF8BOM {
			data = data[3:]
		}
		return string(data), nil
	case UTF16LE:
		return decodeUTF16(data, true), nil
	case UTF16BE:
		return decodeUTF16(data, false), nil
	case GBK:
		decoder := simplifiedchinese.GBK.NewDecoder()
		text, err := decoder.Bytes(data)
		if err != nil {
			return string(data), err
		}
		return string(text), nil
	case GB2312:
		decoder := simplifiedchinese.HZGB2312.NewDecoder()
		text, err := decoder.Bytes(data)
		if err != nil {
			return string(data), err
		}
		return string(text), nil
	case BIG5:
		decoder := traditionalchinese.Big5.NewDecoder()
		text, err := decoder.Bytes(data)
		if err != nil {
			return string(data), err
		}
		return string(text), nil
	case SHIFT_JIS:
		decoder := japanese.ShiftJIS.NewDecoder()
		text, err := decoder.Bytes(data)
		if err != nil {
			return string(data), err
		}
		return string(text), nil
	case EUC_JP:
		decoder := japanese.EUCJP.NewDecoder()
		text, err := decoder.Bytes(data)
		if err != nil {
			return string(data), err
		}
		return string(text), nil
	case EUC_KR:
		decoder := korean.EUCKR.NewDecoder()
		text, err := decoder.Bytes(data)
		if err != nil {
			return string(data), err
		}
		return string(text), nil
	default:
		return string(data), nil
	}
}

func decodeUTF16(data []byte, littleEndian bool) string {
	if len(data) < 2 {
		return string(data)
	}

	var runes []rune
	for i := 0; i < len(data)-1; i += 2 {
		var word uint16
		if littleEndian {
			word = uint16(data[i]) | uint16(data[i+1])<<8
		} else {
			word = uint16(data[i])<<8 | uint16(data[i+1])
		}

		if utf16.IsSurrogate(rune(word)) {
			if i+3 < len(data) {
				var word2 uint16
				if littleEndian {
					word2 = uint16(data[i+2]) | uint16(data[i+3])<<8
				} else {
					word2 = uint16(data[i+2])<<8 | uint16(data[i+3])
				}
				r := utf16.DecodeRune(rune(word), rune(word2))
				runes = append(runes, r)
				i += 2
			} else {
				runes = append(runes, '\ufffd')
			}
		} else {
			runes = append(runes, rune(word))
		}
	}
	return string(runes)
}

func encodeBytes(text string, enc EncodingType) ([]byte, error) {
	switch enc {
	case UTF8:
		return []byte(text), nil
	case UTF8BOM:
		return append([]byte{0xEF, 0xBB, 0xBF}, []byte(text)...), nil
	case UTF16LE:
		return encodeUTF16(text, true), nil
	case UTF16BE:
		return encodeUTF16(text, false), nil
	case GBK:
		encoder := simplifiedchinese.GBK.NewEncoder()
		return encoder.Bytes([]byte(text))
	case BIG5:
		encoder := traditionalchinese.Big5.NewEncoder()
		return encoder.Bytes([]byte(text))
	case SHIFT_JIS:
		encoder := japanese.ShiftJIS.NewEncoder()
		return encoder.Bytes([]byte(text))
	default:
		return []byte(text), nil
	}
}

func encodeUTF16(text string, littleEndian bool) []byte {
	runes := []rune(text)
	pairs := utf16.Encode(runes)

	var buf bytes.Buffer
	for _, pair := range pairs {
		if littleEndian {
			buf.WriteByte(byte(pair))
			buf.WriteByte(byte(pair >> 8))
		} else {
			buf.WriteByte(byte(pair >> 8))
			buf.WriteByte(byte(pair))
		}
	}
	return buf.Bytes()
}

func NewDecodedReader(r io.Reader, enc EncodingType) io.Reader {
	return r
}
