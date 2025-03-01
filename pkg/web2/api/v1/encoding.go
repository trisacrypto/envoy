package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"
)

const (
	encodingNone    = "none"
	encodingBase64  = "base64"
	encodingUnknown = "unknown"
	encodingDefault = encodingBase64
	formatPB        = "pb"
	formatJSON      = "json"
	formatDefault   = formatJSON
)

// EncodingQuery manages how IVMS101 data is returned.
type EncodingQuery struct {
	Encoding string `json:"encoding,omitempty" url:"encoding,omitempty" form:"encoding"`
	Format   string `json:"format,omitempty" url:"format,omitempty" form:"format"`
	b64std   bool   `json:"-"`
}

func (q *EncodingQuery) Validate() (err error) {
	q.Encoding = strings.ToLower(strings.TrimSpace(q.Encoding))
	q.Format = strings.ToLower(strings.TrimSpace(q.Format))

	if q.Encoding != "" && q.Encoding != encodingNone && q.Encoding != encodingBase64 {
		err = ValidationError(err, IncorrectField("encoding", "specify either 'none' or 'base64'"))
	}

	if q.Format != "" && q.Format != formatPB && q.Format != formatJSON {
		err = ValidationError(err, IncorrectField("format", "specify either 'pb' or 'json'"))
	}

	if q.Format == formatPB && !(q.Encoding == "" || q.Encoding == encodingBase64) {
		err = ValidationError(err, IncorrectField("format", "when format is 'pb' encoding must be 'base64'"))
	}

	return err
}

func (q *EncodingQuery) Marshal(in any) (_ string, err error) {
	if q.Format == "" {
		q.Format = formatDefault
	}

	var data []byte
	switch q.Format {
	case formatPB:
		// NOTE: type cast will panic if input is not a protocol buffer message
		if data, err = proto.Marshal(in.(proto.Message)); err != nil {
			return "", err
		}
	case formatJSON:
		if data, err = json.Marshal(in); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("%q is not a valid format", q.Format)
	}

	return q.Encode(data)
}

func (q *EncodingQuery) Unmarshal(in string, v any) (err error) {
	var data []byte
	if data, err = q.Decode(in); err != nil {
		return err
	}

	// Attempt to unmarshal protocol buffers
	m, isPB := v.(proto.Message)
	if q.Format == formatPB || (isPB && q.Format != formatJSON) {
		if !isPB {
			return fmt.Errorf("cannot unmarshal protocol buffers into %T", v)
		}
		return proto.Unmarshal(data, m)
	}

	// Unmarshal JSON data
	return json.Unmarshal(data, v)
}

func (q *EncodingQuery) Encode(data []byte) (string, error) {
	if q.Encoding == "" {
		q.Encoding = encodingDefault
	}

	switch q.Encoding {
	case encodingBase64:
		switch {
		case q.b64std:
			return base64.StdEncoding.EncodeToString(data), nil
		default:
			return base64.URLEncoding.EncodeToString(data), nil
		}
	case encodingNone:
		return string(data), nil
	default:
		return "", fmt.Errorf("%q is not a valid encoding", q.Encoding)
	}
}

func (q *EncodingQuery) Decode(data string) ([]byte, error) {
	// Attempt to detect the format and decode appropriately
	// NOTE: even if we don't set the decoding we still have to determine std vs url base64.
	if encoding := q.DetectEncoding(data); q.Encoding == "" {
		q.Encoding = encoding
	}

	switch q.Encoding {
	case encodingBase64:
		switch {
		case q.b64std:
			return base64.StdEncoding.DecodeString(data)
		default:
			return base64.URLEncoding.DecodeString(data)
		}
	case encodingNone:
		return []byte(data), nil
	case encodingUnknown:
		return nil, ErrUnknownEncoding
	default:
		return nil, fmt.Errorf("%q is not a valid encoding", q.Encoding)
	}
}

var (
	b64stdre = regexp.MustCompile(`^[A-Za-z0-9+/]*={0,3}$`)
	b64urlre = regexp.MustCompile(`^[A-Za-z0-9\-_]*={0,3}$`)
)

func (q *EncodingQuery) DetectEncoding(data string) string {
	// NOTE: this only allows JSON objects, arrays, and null to be decoded.
	if data == "" || data[0] == '{' || data[0] == '[' || data == "null" {
		return encodingNone
	}

	if b64stdre.MatchString(data) {
		q.b64std = true
		return encodingBase64
	}

	if b64urlre.MatchString(data) {
		q.b64std = false
		return encodingBase64
	}

	return encodingUnknown
}
