package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewRecognizesContentAndEnforcesBounds(t *testing.T) {
	root, base := rootFixture(t)
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root.Path, "image.txt")
	if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := Inspect(root, "image.txt")
	if err != nil || info.Kind != "image" || info.MIME != "image/png" || info.Width != 2 || info.Height != 3 || !info.Previewable || info.Editable {
		t.Fatalf("%+v %v", info, err)
	}
	data, _, err := ReadPreview(root, "image.txt")
	if err != nil || !bytes.Equal(data, out.Bytes()) {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "outside"), out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "outside"), filepath.Join(root.Path, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../outside", "escape"} {
		if _, err := Inspect(root, path); err == nil {
			t.Fatalf("escape accepted %s", path)
		}
	}
	file, err := os.Create(filepath.Join(root.Path, "huge"))
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(MaxPreviewBytes + 1); err != nil {
		t.Fatal(err)
	}
	file.Close()
	info, err = Inspect(root, "huge")
	if err != nil || info.Previewable || info.Editable {
		t.Fatalf("oversize %+v %v", info, err)
	}
	if _, _, err := ReadPreview(root, "huge"); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{8193, 1}, {4096, 4096}, {0, 4}} {
		if validDimensions(size[0], size[1]) {
			t.Fatal("pixel bound")
		}
	}
	for _, data := range [][]byte{[]byte("\ufeffhello\r\n"), {}, []byte("你好")} {
		info := inspectBytes(data, int64(len(data)))
		if info.Kind != "text" || !info.Editable || info.Previewable {
			t.Fatalf("text %+v", info)
		}
	}
	if info := inspectBytes([]byte{0, 255}, 2); info.Kind != "binary" || info.Previewable || info.Editable {
		t.Fatal(info)
	}
}
func TestSVGPreviewRejectsExecutableAndExternalContent(t *testing.T) {
	for _, svg := range []string{`<svg xmlns="http://www.w3.org/2000/svg" width="32" height="20"><rect width="20" height="20" fill="#fff"/></svg>`, `<?xml version="1.0"?><svg viewBox="0 0 20 20"><path d="M 0 0 L 20 20"/></svg>`} {
		info := inspectBytes([]byte(svg), int64(len(svg)))
		if !info.Previewable || info.MIME != "image/svg+xml" {
			t.Fatalf("safe svg %+v", info)
		}
	}
	for _, svg := range []string{`<svg onload="alert(1)"/>`, `<svg><script>alert(1)</script></svg>`, `<svg><foreignObject/></svg>`, `<svg><image href="https://example.com/a"/></svg>`, `<svg><use href="#a"/></svg>`, `<svg><style>*{fill:red}</style></svg>`, `<?xml-stylesheet href="https://example.com/a"?><svg/>`, `<!DOCTYPE svg [<!ENTITY x SYSTEM "file:///etc/passwd">]><svg>&x;</svg>`, `<svg width="8193"/>`, `<svg width="4096" height="4096"/>`, `<svg><rect fill="url(https://example.com/a)"/></svg>`, `<svg/><svg/>`, `<svg xmlns="http://evil/"/>`} {
		info := inspectBytes([]byte(svg), int64(len(svg)))
		if info.Previewable {
			t.Fatalf("unsafe svg accepted %s", svg)
		}
	}
}
func TestWebPAndAVIFDimensionsUseDeclaredContainers(t *testing.T) {
	webp := make([]byte, 30)
	copy(webp, "RIFF")
	copy(webp[8:], "WEBPVP8X")
	binary.LittleEndian.PutUint32(webp[16:], 10)
	webp[24] = 31
	webp[27] = 19
	if w, h := webpDimensions(webp); w != 32 || h != 20 {
		t.Fatalf("webp %d %d", w, h)
	}
	if w, _ := webpDimensions(webp[:25]); w != 0 {
		t.Fatal("truncated webp")
	}
	box := func(kind string, data []byte) []byte {
		result := make([]byte, 8+len(data))
		binary.BigEndian.PutUint32(result, uint32(len(result)))
		copy(result[4:], kind)
		copy(result[8:], data)
		return result
	}
	size := make([]byte, 12)
	binary.BigEndian.PutUint32(size[4:], 32)
	binary.BigEndian.PutUint32(size[8:], 20)
	avif := box("meta", append(make([]byte, 4), box("iprp", box("ipco", box("ispe", size)))...))
	if w, h := avifDimensions(avif, 0); w != 32 || h != 20 {
		t.Fatalf("avif %d %d", w, h)
	}
	large := append([]byte(nil), size...)
	binary.BigEndian.PutUint32(large[4:], 8193)
	properties := append(box("ispe", size), box("ispe", large)...)
	multi := box("meta", append(make([]byte, 4), box("iprp", box("ipco", properties))...))
	if w, h := avifDimensions(multi, 0); validDimensions(w, h) {
		t.Fatal("small first property concealed oversized second image")
	}
	if w, _ := avifDimensions(append([]byte("random"), size...), 0); w != 0 {
		t.Fatal("arbitrary bytes accepted")
	}
}
