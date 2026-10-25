package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/maulanaiskak/portfolio-api/internal/entity"
)

type memStorage struct {
	key, contentType string
	data             []byte
}

func (m *memStorage) Put(_ context.Context, key, contentType string, body io.ReadSeeker, _ int64) (string, error) {
	m.key, m.contentType = key, contentType
	m.data, _ = io.ReadAll(body)
	return "https://cdn.test/" + key, nil
}

func encodeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withExif inserts an APP1 Exif segment (big-endian) with an Orientation
// tag and a fake GPS marker string right after the JPEG SOI marker.
func withExif(jpg []byte, orientation byte) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, // header, IFD0 at offset 8
		0, 1, // 1 entry
		0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, orientation, 0, 0, // Orientation SHORT
		0, 0, 0, 0, // next IFD
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	payload = append(payload, []byte("GPS-SECRET-LOCATION")...)
	seg := []byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}
	seg = append(seg, payload...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func TestUploadImageStripsExifAndAppliesOrientation(t *testing.T) {
	store := &memStorage{}
	svc := NewUploadService(store, 10<<20)

	src := withExif(encodeJPEG(t, 40, 20), 6) // 6 = rotate 90° CW
	if jpegOrientation(src) != 6 {
		t.Fatal("test fixture: orientation not parsed")
	}

	img, err := svc.UploadImage(context.Background(), bytes.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if img.Width != 20 || img.Height != 40 {
		t.Fatalf("expected rotated 20x40, got %dx%d", img.Width, img.Height)
	}
	if img.ContentType != "image/jpeg" || !strings.HasSuffix(store.key, ".jpg") || !strings.HasPrefix(store.key, "images/") {
		t.Fatalf("unexpected output: %+v key=%s", img, store.key)
	}
	if bytes.Contains(store.data, []byte("Exif")) || bytes.Contains(store.data, []byte("GPS-SECRET")) {
		t.Fatal("metadata was not stripped")
	}
}

func TestUploadImageKeepsAlphaAsPNG(t *testing.T) {
	store := &memStorage{}
	svc := NewUploadService(store, 10<<20)

	img := image.NewNRGBA(image.Rect(0, 0, 10, 10)) // fully transparent
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out, err := svc.UploadImage(context.Background(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	if out.ContentType != "image/png" {
		t.Fatalf("expected png for transparent image, got %s", out.ContentType)
	}
}

func TestUploadImageDownscales(t *testing.T) {
	store := &memStorage{}
	svc := NewUploadService(store, 50<<20)
	out, err := svc.UploadImage(context.Background(), bytes.NewReader(encodeJPEG(t, 3000, 1500)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != maxOutputSide || out.Height != maxOutputSide/2 {
		t.Fatalf("expected %dx%d, got %dx%d", maxOutputSide, maxOutputSide/2, out.Width, out.Height)
	}
}

func TestUploadImageRejects(t *testing.T) {
	svc := NewUploadService(&memStorage{}, 1<<20)
	cases := map[string][]byte{
		"empty":         {},
		"html as image": []byte("<html><script>alert(1)</script></html>"),
		"svg":           []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`),
		"too large":     bytes.Repeat([]byte{0}, 2<<20),
		"truncated jpg": encodeJPEG(t, 50, 50)[:40],
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.UploadImage(context.Background(), bytes.NewReader(data))
			if !entity.IsKind(err, entity.KindValidation) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}
