package ws

import (
	"encoding/json"
	"io"
)

func toJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func fromJSON[T any](r io.Reader) (T, error) {
	var v T
	if r == nil {
		return v, nil
	}

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	err := dec.Decode(&v)
	if err != nil {
		return v, err
	}
	return v, nil
}
