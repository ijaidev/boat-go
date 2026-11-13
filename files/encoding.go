package files

import "fmt"

// Encoding is the wire encoding of inline file content.
type Encoding string

const (
	// EncodingUTF8 sends text as UTF-8. It is the server default.
	EncodingUTF8 Encoding = "utf8"
	// EncodingBase64 sends or receives binary content as standard base64.
	EncodingBase64 Encoding = "base64"
)

func normalizeEncoding(enc Encoding) (Encoding, error) {
	if enc == "" {
		return EncodingUTF8, nil
	}
	switch enc {
	case EncodingUTF8, EncodingBase64:
		return enc, nil
	default:
		return "", fmt.Errorf("files: unsupported encoding %q", enc)
	}
}
