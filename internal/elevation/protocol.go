package elevation

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"unicode/utf8"

	"persistty/internal/storage"
)

func WriteBytes(w io.Writer, data []byte, max int) error {
	if len(data) > max {
		return ErrInvalid
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	if _, err := io.Copy(w, bytes.NewReader(header[:])); err != nil {
		return err
	}
	_, err := io.Copy(w, bytes.NewReader(data))
	return err
}
func ReadBytes(r io.Reader, max int) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if uint64(size) > uint64(max) {
		return nil, ErrInvalid
	}
	data := make([]byte, int(size))
	_, err := io.ReadFull(r, data)
	if err != nil {
		clear(data)
		return nil, err
	}
	return data, err
}
func WriteFrame(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return WriteBytes(w, data, 64<<10)
}
func ReadFrame(r io.Reader, value any) error {
	data, err := ReadBytes(r, 64<<10)
	if err != nil {
		return err
	}
	defer clear(data)
	return strictJSON(data, value)
}
func strictJSON(data []byte, value any) error {
	if !utf8.Valid(data) {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := checkJSON(d, 0); err != nil {
		return ErrInvalid
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrInvalid
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		return ErrInvalid
	}
	// encoding/json otherwise accepts case-insensitive aliases of struct keys.
	// Compare against the actual DTO keys so ID and id cannot silently override.
	canonical, err := json.Marshal(value)
	if err != nil {
		return ErrInvalid
	}
	var input, output any
	if json.Unmarshal(data, &input) != nil || json.Unmarshal(canonical, &output) != nil || !matchingKeys(input, output) {
		return ErrInvalid
	}
	return nil
}
func matchingKeys(input, output any) bool {
	switch in := input.(type) {
	case map[string]any:
		out, ok := output.(map[string]any)
		if !ok {
			return false
		}
		for key, value := range in {
			expected, exists := out[key]
			if !exists || !matchingKeys(value, expected) {
				return false
			}
		}
	case []any:
		out, ok := output.([]any)
		if !ok || len(in) != len(out) {
			return false
		}
		for i, value := range in {
			if !matchingKeys(value, out[i]) {
				return false
			}
		}
	}
	return true
}
func checkJSON(d *json.Decoder, depth int) error {
	if depth > 16 {
		return ErrInvalid
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			t, err = d.Token()
			if err != nil {
				return err
			}
			key, ok := t.(string)
			if !ok || seen[key] {
				return ErrInvalid
			}
			seen[key] = true
			if err = checkJSON(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err = checkJSON(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return ErrInvalid
	}
	_, err = d.Token()
	return err
}

type brokerRequest struct {
	Action string                   `json:"action"`
	Grant  Grant                    `json:"grant"`
	Root   storage.RegisteredFolder `json:"root"`
	Path   string                   `json:"path"`
}
type reply struct {
	Phase  string  `json:"phase"`
	Target *Target `json:"target"`
	Result *Result `json:"result"`
	Code   string  `json:"code"`
}
