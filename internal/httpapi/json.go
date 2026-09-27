package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"unicode/utf8"
)

var errBodyLarge = errors.New("body too large")

func decodeJSON(w http.ResponseWriter, r *http.Request, target any, fields ...string) error {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return errors.New("expected json")
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8192))
	if err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			return errBodyLarge
		}
		return err
	}
	if !utf8.Valid(body) {
		return errors.New("invalid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err = checkValue(decoder, 0); err != nil {
		return err
	}
	if _, err = decoder.Token(); err != io.EOF {
		return errors.New("multiple json values")
	}
	var object map[string]json.RawMessage
	if err = json.Unmarshal(body, &object); err != nil || object == nil {
		return errors.New("expected json object")
	}
	for key := range object {
		known := false
		for _, field := range fields {
			if key == field {
				known = true
				break
			}
		}
		if !known {
			return errors.New("unknown field")
		}
	}
	decoder = json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
func checkValue(d *json.Decoder, depth int) error {
	if depth > 16 {
		return errors.New("json too deep")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]bool)
		for d.More() {
			key, e := d.Token()
			if e != nil {
				return e
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return errors.New("duplicate key")
			}
			seen[name] = true
			if e = checkValue(d, depth+1); e != nil {
				return e
			}
		}
	case '[':
		for d.More() {
			if err = checkValue(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return errors.New("invalid delimiter")
	}
	_, err = d.Token()
	return err
}
