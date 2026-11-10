package assets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

const (
	tinyWebP2x2 = "UklGRiwAAABXRUJQVlA4TB8AAAAvAUAAAB8gEEjeHzqN+RcQFPwf3fxHZA/gBgwR/Q8BAA=="
	tinyWebP3x3 = "UklGRhwAAABXRUJQVlA4TBAAAAAvAoAAAAdQwOh//wMR0f8A"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func mustWebP(t *testing.T, encoded string) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode webp fixture: %v", err)
	}
	return data
}

func encodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: 200, G: uint8(x), B: uint8(y), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func withPNGChunk(t *testing.T, source []byte, kind string, payload []byte) []byte {
	t.Helper()
	if len(source) < 12 || string(source[len(source)-8:len(source)-4]) != "IEND" {
		t.Fatal("png fixture does not end with IEND")
	}
	end := len(source) - 12
	chunk := make([]byte, 12+len(payload))
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(payload)))
	copy(chunk[4:8], kind)
	copy(chunk[8:], payload)
	crc := crc32.ChecksumIEEE(chunk[4 : 8+len(payload)])
	binary.BigEndian.PutUint32(chunk[8+len(payload):], crc)
	out := make([]byte, 0, len(source)+len(chunk))
	out = append(out, source[:end]...)
	out = append(out, chunk...)
	out = append(out, source[end:]...)
	return out
}

func animatedWebP() []byte {
	payload := make([]byte, 10)
	payload[0] = 0x02 // VP8X animation flag
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(4+8+len(payload)))
	buf.WriteString("WEBP")
	buf.WriteString("VP8X")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(payload)))
	buf.Write(payload)
	return buf.Bytes()
}

func TestSaveSourceAcceptsPNGJPEGWebP(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	cases := []struct {
		name      string
		data      []byte
		filename  string
		mediaType string
		extension string
		width     int
		height    int
	}{
		{"png", encodePNG(t, 5, 4), "../../sheet.png", "image/png", "png", 5, 4},
		{"jpeg", encodeJPEG(t, 6, 3), "sheet.jpeg", "image/jpeg", "jpg", 6, 3},
		{"webp", mustWebP(t, tinyWebP2x2), "sheet.webp", "image/webp", "webp", 2, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := store.SaveSource(ctx, bytes.NewReader(tc.data), tc.filename)
			if err != nil {
				t.Fatalf("SaveSource: %v", err)
			}
			if _, err := uuid.Parse(source.ID); err != nil {
				t.Fatalf("id %q is not a server uuid", source.ID)
			}
			if source.Filename != filepath.Base(tc.filename) {
				t.Errorf("filename = %q, want sanitized base", source.Filename)
			}
			if source.MediaType != tc.mediaType || source.Width != tc.width || source.Height != tc.height {
				t.Errorf("source = %+v", source)
			}
			if source.ByteSize != int64(len(tc.data)) {
				t.Errorf("byte size = %d, want %d", source.ByteSize, len(tc.data))
			}
			sum := sha256.Sum256(tc.data)
			if source.ChecksumSHA256 != hex.EncodeToString(sum[:]) {
				t.Errorf("checksum mismatch")
			}
			if source.RelativePath != "sources/"+source.ID+"/original."+tc.extension {
				t.Errorf("relative path = %q", source.RelativePath)
			}
			path, err := store.SourcePath(source.ID)
			if err != nil {
				t.Fatalf("SourcePath: %v", err)
			}
			if filepath.Dir(path) != filepath.Join(store.dataDir, "sources", source.ID) {
				t.Errorf("source path %q outside expected dir", path)
			}
			stored, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read stored source: %v", err)
			}
			if !bytes.Equal(stored, tc.data) {
				t.Errorf("stored bytes differ")
			}
			f, err := store.OpenSource(source.ID)
			if err != nil {
				t.Fatalf("OpenSource: %v", err)
			}
			defer f.Close()
			if info, err := f.Stat(); err != nil || info.Size() != int64(len(tc.data)) {
				t.Errorf("opened source stat = %v, %v", info, err)
			}
		})
	}
}

func TestSaveSourceRejectsBadInputs(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	validPNG := encodePNG(t, 4, 4)

	tooLarge := make([]byte, MaxSourceBytes+1)
	copy(tooLarge, validPNG)
	apng := withPNGChunk(t, validPNG, "acTL", make([]byte, 8))

	cases := []struct {
		name     string
		data     []byte
		expected error
	}{
		{"empty", nil, ErrInvalidImage},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), ErrInvalidImage},
		{"truncated png", validPNG[:20], ErrInvalidImage},
		{"too large", tooLarge, ErrImageTooLarge},
		{"too wide", encodePNG(t, MaxImageDimension+1, 1), ErrImageDimensions},
		{"animated png", apng, ErrAnimatedImage},
		{"animated webp", animatedWebP(), ErrAnimatedImage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := store.SaveSource(ctx, bytes.NewReader(tc.data), "sheet.png")
			if !errors.Is(err, tc.expected) {
				t.Fatalf("error = %v, want %v", err, tc.expected)
			}
		})
	}
}

func TestPathTraversalAndSymlinkAreRejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	packID := uuid.NewString()

	badCommitted := []string{
		"",
		"../secret",
		"/etc/passwd",
		"packs/../secret",
		"packs/" + packID + "/v1/../../secret",
		"packs/not-a-uuid/v1/000.webp",
		"bundles/" + packID + "/v0.zip",
		"bundles/" + packID + "/v1.zip/extra",
	}
	for _, rel := range badCommitted {
		if f, err := store.OpenCommitted(rel); !errors.Is(err, ErrUnsafePath) {
			if f != nil {
				f.Close()
			}
			t.Errorf("OpenCommitted(%q) error = %v, want ErrUnsafePath", rel, err)
		}
	}

	secret := filepath.Join(t.TempDir(), "secret.webp")
	if err := os.WriteFile(secret, mustWebP(t, tinyWebP2x2), 0o644); err != nil {
		t.Fatal(err)
	}
	packDir := filepath.Join(store.dataDir, "packs", packID, "v1")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(packDir, "000.webp")
	if err := os.Symlink(secret, linked); err != nil {
		t.Fatal(err)
	}
	if f, err := store.OpenCommitted("packs/" + packID + "/v1/000.webp"); !errors.Is(err, ErrUnsafePath) {
		if f != nil {
			f.Close()
		}
		t.Errorf("symlinked committed file error = %v, want ErrUnsafePath", err)
	}

	source, err := store.SaveSource(ctx, bytes.NewReader(encodePNG(t, 2, 2)), "sheet.png")
	if err != nil {
		t.Fatalf("SaveSource: %v", err)
	}
	sourcePath, err := store.SourcePath(source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(sourcePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, sourcePath); err != nil {
		t.Fatal(err)
	}
	if f, err := store.OpenSource(source.ID); !errors.Is(err, ErrNotFound) {
		if f != nil {
			f.Close()
		}
		t.Errorf("symlinked source error = %v, want ErrNotFound", err)
	}

	if _, err := store.SourcePath("../" + source.ID); !errors.Is(err, ErrInvalidSourceID) {
		t.Errorf("SourcePath traversal error = %v", err)
	}
	if _, err := store.WorkDir("../job"); !errors.Is(err, ErrInvalidJobID) {
		t.Errorf("WorkDir traversal error = %v", err)
	}
}
