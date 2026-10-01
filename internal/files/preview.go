package files

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"persistty/internal/storage"
)

const MaxPreviewBytes = 16 << 20
const MaxPreviewPixels = 16_000_000

type Inspection struct {
	Kind        string `json:"kind"`
	MIME        string `json:"mime"`
	Size        int64  `json:"size"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Editable    bool   `json:"editable"`
	Previewable bool   `json:"previewable"`
}

func validDimensions(w, h int) bool {
	return w > 0 && h > 0 && w <= 8192 && h <= 8192 && int64(w)*int64(h) <= MaxPreviewPixels
}
func previewBytes(root storage.RegisteredFolder, relative string) ([]byte, int64, error) {
	if !ValidRelative(relative, false) {
		return nil, 0, ErrInvalidPath
	}
	file, err := openConstrained(root, relative, false)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, ErrUnsupported
	}
	if info.Size() > MaxPreviewBytes {
		return nil, info.Size(), nil
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxPreviewBytes+1))
	if err != nil {
		return nil, 0, err
	}
	if len(data) > MaxPreviewBytes {
		return nil, 0, ErrTooLarge
	}
	final, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	if info.Size() != final.Size() || !info.ModTime().Equal(final.ModTime()) || int64(len(data)) != info.Size() {
		return nil, 0, ErrConflict
	}
	return data, info.Size(), nil
}
func Inspect(root storage.RegisteredFolder, relative string) (Inspection, error) {
	data, size, err := previewBytes(root, relative)
	if err != nil {
		return Inspection{}, err
	}
	return inspectBytes(data, size), nil
}
func ReadPreview(root storage.RegisteredFolder, relative string) ([]byte, Inspection, error) {
	data, size, err := previewBytes(root, relative)
	if err != nil {
		return nil, Inspection{}, err
	}
	info := inspectBytes(data, size)
	if !info.Previewable {
		return nil, info, ErrUnsupported
	}
	return data, info, nil
}
func inspectBytes(data []byte, size int64) Inspection {
	info := Inspection{Kind: "binary", MIME: "application/octet-stream", Size: size}
	if data == nil && size > 0 {
		return info
	}
	mime := http.DetectContentType(data)
	info.MIME = mime
	text := utf8.Valid(data) && !bytes.ContainsRune(data, 0)
	info.Editable = text && size <= 8<<20
	if text {
		info.Kind = "text"
		info.MIME = "text/plain;charset=utf-8"
	}
	if config, format, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		info.Kind = "image"
		info.Editable = false
		info.MIME = "image/" + format
		info.Width = config.Width
		info.Height = config.Height
		info.Previewable = validDimensions(config.Width, config.Height)
		return info
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		w, h := webpDimensions(data)
		info.Kind = "image"
		info.Editable = false
		info.MIME = "image/webp"
		info.Width = w
		info.Height = h
		info.Previewable = validDimensions(w, h)
		return info
	}
	if len(data) >= 16 && string(data[4:8]) == "ftyp" && (string(data[8:12]) == "avif" || string(data[8:12]) == "avis") {
		w, h := avifDimensions(data, 0)
		info.Kind = "image"
		info.Editable = false
		info.MIME = "image/avif"
		info.Width = w
		info.Height = h
		info.Previewable = validDimensions(w, h)
		return info
	}
	if text {
		trimmed := bytes.TrimSpace(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}))
		if bytes.HasPrefix(trimmed, []byte("<svg")) || bytes.HasPrefix(trimmed, []byte("<?xml")) {
			if w, h, ok := safeSVG(trimmed); ok {
				info.Kind = "image"
				info.MIME = "image/svg+xml"
				info.Width = w
				info.Height = h
				info.Previewable = true
			}
		}
	}
	return info
}
func webpDimensions(data []byte) (int, int) {
	for offset := 12; offset+8 <= len(data); {
		kind := string(data[offset : offset+4])
		size := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if size < 0 || size > len(data)-offset {
			return 0, 0
		}
		chunk := data[offset : offset+size]
		switch kind {
		case "VP8X":
			if len(chunk) >= 10 {
				return 1 + int(chunk[4]) + int(chunk[5])<<8 + int(chunk[6])<<16, 1 + int(chunk[7]) + int(chunk[8])<<8 + int(chunk[9])<<16
			}
		case "VP8 ":
			if len(chunk) >= 10 && bytes.Equal(chunk[3:6], []byte{0x9d, 0x01, 0x2a}) {
				return int(binary.LittleEndian.Uint16(chunk[6:8]) & 0x3fff), int(binary.LittleEndian.Uint16(chunk[8:10]) & 0x3fff)
			}
		case "VP8L":
			if len(chunk) >= 5 && chunk[0] == 0x2f {
				bits := binary.LittleEndian.Uint32(chunk[1:5])
				return int(bits&0x3fff) + 1, int(bits>>14&0x3fff) + 1
			}
		}
		offset += size + (size & 1)
	}
	return 0, 0
}

// Only walk declared BMFF container boxes, never search arbitrary compressed bytes.
func avifDimensions(data []byte, depth int) (int, int) {
	w, h, ok := avifProperties(data, depth)
	if !ok {
		return 0, 0
	}
	return w, h
}
func avifProperties(data []byte, depth int) (int, int, bool) {
	if depth > 8 {
		return 0, 0, false
	}
	width, height := 0, 0
	for offset := 0; offset < len(data); {
		if offset+8 > len(data) {
			return 0, 0, false
		}
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		kind := string(data[offset+4 : offset+8])
		header := 8
		if size == 1 {
			if offset+16 > len(data) {
				return 0, 0, false
			}
			n := binary.BigEndian.Uint64(data[offset+8 : offset+16])
			if n > uint64(len(data)-offset) {
				return 0, 0, false
			}
			size = int(n)
			header = 16
		}
		if size == 0 {
			size = len(data) - offset
		}
		if size < header || size > len(data)-offset {
			return 0, 0, false
		}
		payload := data[offset+header : offset+size]
		if kind == "ispe" {
			if len(payload) != 12 {
				return 0, 0, false
			}
			w, h := int(binary.BigEndian.Uint32(payload[4:8])), int(binary.BigEndian.Uint32(payload[8:12]))
			if w <= 0 || h <= 0 {
				return 0, 0, false
			}
			width = max(width, w)
			height = max(height, h)
		}
		if kind == "meta" {
			if len(payload) < 4 {
				return 0, 0, false
			}
			payload = payload[4:]
		}
		if kind == "meta" || kind == "iprp" || kind == "ipco" {
			w, h, ok := avifProperties(payload, depth+1)
			if !ok {
				return 0, 0, false
			}
			width = max(width, w)
			height = max(height, h)
		}
		offset += size
	}
	return width, height, true
}

var svgElements = map[string]bool{"svg": true, "g": true, "path": true, "rect": true, "circle": true, "ellipse": true, "line": true, "polyline": true, "polygon": true, "title": true, "desc": true, "text": true, "tspan": true, "defs": true, "linearGradient": true, "radialGradient": true, "stop": true, "clipPath": true}
var svgAttributes = map[string]bool{"xmlns": true, "viewBox": true, "width": true, "height": true, "x": true, "y": true, "x1": true, "x2": true, "y1": true, "y2": true, "cx": true, "cy": true, "r": true, "rx": true, "ry": true, "d": true, "points": true, "fill": true, "stroke": true, "stroke-width": true, "stroke-linecap": true, "stroke-linejoin": true, "opacity": true, "fill-opacity": true, "stroke-opacity": true, "fill-rule": true, "clip-rule": true, "transform": true, "id": true, "offset": true, "stop-color": true, "stop-opacity": true, "gradientUnits": true, "gradientTransform": true, "font-size": true, "text-anchor": true, "preserveAspectRatio": true}

func safeSVG(data []byte) (int, int, bool) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	depth, count := 0, 0
	w, h := 300, 150
	root := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return w, h, root && depth == 0 && validDimensions(w, h)
		}
		if err != nil {
			return 0, 0, false
		}
		count++
		if count > 10000 {
			return 0, 0, false
		}
		switch value := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				if root || value.Name.Local != "svg" {
					return 0, 0, false
				}
				root = true
			}
			if value.Name.Space != "" && value.Name.Space != "http://www.w3.org/2000/svg" {
				return 0, 0, false
			}
			if !svgElements[value.Name.Local] {
				return 0, 0, false
			}
			depth++
			if depth > 64 {
				return 0, 0, false
			}
			for _, attr := range value.Attr {
				if !svgAttributes[attr.Name.Local] || (attr.Name.Space != "" && attr.Name.Local != "xmlns") || strings.ContainsAny(attr.Value, "\\@&") || strings.Contains(strings.ToLower(attr.Value), "url(") {
					return 0, 0, false
				}
				if attr.Name.Local == "xmlns" && attr.Value != "http://www.w3.org/2000/svg" {
					return 0, 0, false
				}
				if depth == 1 && (attr.Name.Local == "width" || attr.Name.Local == "height") {
					number, err := strconv.ParseFloat(strings.TrimSuffix(attr.Value, "px"), 64)
					if err != nil || number <= 0 || number > 8192 {
						return 0, 0, false
					}
					if attr.Name.Local == "width" {
						w = int(number)
					} else {
						h = int(number)
					}
				}
			}
		case xml.EndElement:
			depth--
		case xml.Directive:
			return 0, 0, false
		case xml.ProcInst:
			if value.Target != "xml" {
				return 0, 0, false
			}
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(value)) != "" {
				return 0, 0, false
			}
		}
	}
}
