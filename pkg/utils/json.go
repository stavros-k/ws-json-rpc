package utils

import (
	"bytes"
	"encoding/json"
	"io"
)

// FromJSON decodes JSON from byte slice (wrapper around streaming version)
func FromJSON[T any](data []byte) (T, error) {
	var result T
	if len(data) == 0 {
		return result, nil
	}
	return FromJSONStream[T](bytes.NewReader(data))
}

// ToJSON encodes to JSON byte slice (wrapper around streaming version)
func ToJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := ToJSONStream(&buf, v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

// FromJSONStream decodes JSON from io.Reader (streaming version)
func FromJSONStream[T any](r io.Reader) (T, error) {
	var result T
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&result)
	return result, err
}

// ToJSONStream encodes to JSON and writes to io.Writer (streaming version)
func ToJSONStream(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "")
	return encoder.Encode(v)
}
