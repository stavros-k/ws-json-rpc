package ws

import (
	"bytes"
	"encoding/json"
)

func FromJSON[T any](data []byte) (T, error) {
	var result T
	if len(data) == 0 {
		return result, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&result)
	return result, err
}

func ToJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}
