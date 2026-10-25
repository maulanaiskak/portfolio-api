package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder

	"github.com/maulanaiskak/portfolio-api/internal/entity"
	"github.com/maulanaiskak/portfolio-api/internal/repository"
)

const (
	// Reject images above this many pixels before decoding them fully
	// (decompression-bomb guard; ~40MP is a high-end phone photo).
	maxSourcePixels = 40_000_000
	// Longest side after resizing; plenty for full-width hero images.
	maxOutputSide = 2560
	jpegQuality   = 85
)

type UploadedImage struct {
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Bytes       int64  `json:"bytes"`
}

type UploadService interface {
	// UploadImage validates, normalizes and stores an image. The file is
	// fully re-encoded, which strips all metadata (EXIF GPS location,
	// camera serials) after applying EXIF orientation.
	UploadImage(ctx context.Context, r io.Reader) (UploadedImage, error)
}

type uploadService struct {
	storage  repository.ObjectStorage
	maxBytes int64
	now      func() time.Time
}

func (s *uploadService) UploadImage(ctx context.Context, r io.Reader) (UploadedImage, error) {
	data, err := io.ReadAll(io.LimitReader(r, s.maxBytes+1))
	if err != nil {
		return UploadedImage{}, err
	}
	if int64(len(data)) > s.maxBytes {
		return UploadedImage{}, entity.NewValidationError("image must be at most %d MB", s.maxBytes>>20)
	}
	if len(data) == 0 {
		return UploadedImage{}, entity.NewValidationError("file is empty")
	}

	// Check the real format and dimensions from the bytes, not the
	// client-supplied filename or Content-Type.
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png" && format != "webp") {
		return UploadedImage{}, entity.NewValidationError("file must be a JPEG, PNG or WebP image")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxSourcePixels {
		return UploadedImage{}, entity.NewValidationError("image must be at most %d megapixels", maxSourcePixels/1_000_000)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return UploadedImage{}, entity.NewValidationError("image could not be decoded")
	}
	// Resize before rotating: the pixel-by-pixel rotation is then bounded
	// by the output size, not a 40MP source.
	img = fitWithin(img, maxOutputSide)
	if format == "jpeg" {
		img = applyOrientation(img, jpegOrientation(data))
	}

	var (
		out         bytes.Buffer
		ext         string
		contentType string
	)
	if format == "jpeg" || isOpaque(img) {
		ext, contentType = "jpg", "image/jpeg"
		err = jpeg.Encode(&out, img, &jpeg.Options{Quality: jpegQuality})
	} else {
		ext, contentType = "png", "image/png"
		err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&out, img)
	}
	if err != nil {
		return UploadedImage{}, err
	}

	key := repository.ObjectKey("images", uuid.NewString(), ext, s.now().UTC())
	size := int64(out.Len())
	url, err := s.storage.Put(ctx, key, contentType, bytes.NewReader(out.Bytes()), size)
	if err != nil {
		return UploadedImage{}, err
	}
	b := img.Bounds()
	return UploadedImage{URL: url, ContentType: contentType, Width: b.Dx(), Height: b.Dy(), Bytes: size}, nil
}

func isOpaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	return false
}

// fitWithin downscales img so its longest side is at most max.
func fitWithin(img image.Image, max int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return img
	}
	if w >= h {
		h = h * max / w
		w = max
	} else {
		w = w * max / h
		h = max
	}
	dst := image.NewRGBA(image.Rect(0, 0, max1(w), max1(h)))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// jpegOrientation returns the EXIF Orientation tag (1-8) of a JPEG, or 1 if
// absent/unparseable. Only the APP1 Exif IFD0 is read.
func jpegOrientation(data []byte) int {
	r := bytes.NewReader(data)
	var marker [2]byte
	if _, err := io.ReadFull(r, marker[:]); err != nil || marker != [2]byte{0xFF, 0xD8} {
		return 1
	}
	for {
		if _, err := io.ReadFull(r, marker[:]); err != nil || marker[0] != 0xFF {
			return 1
		}
		var lenBuf [2]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return 1
		}
		segLen := int(lenBuf[0])<<8 | int(lenBuf[1])
		if segLen < 2 {
			return 1
		}
		seg := make([]byte, segLen-2)
		if _, err := io.ReadFull(r, seg); err != nil {
			return 1
		}
		if marker[1] == 0xDA { // start of scan: no more metadata
			return 1
		}
		if marker[1] == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			o, err := tiffOrientation(seg[6:])
			if err != nil {
				return 1
			}
			return o
		}
	}
}

func tiffOrientation(t []byte) (int, error) {
	errBad := errors.New("bad exif")
	if len(t) < 8 {
		return 0, errBad
	}
	var u16 func([]byte) int
	var u32 func([]byte) int
	switch string(t[:2]) {
	case "II":
		u16 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 }
		u32 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24 }
	case "MM":
		u16 = func(b []byte) int { return int(b[0])<<8 | int(b[1]) }
		u32 = func(b []byte) int { return int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3]) }
	default:
		return 0, errBad
	}
	ifd := u32(t[4:8])
	if ifd < 8 || ifd+2 > len(t) {
		return 0, errBad
	}
	n := u16(t[ifd : ifd+2])
	for i := 0; i < n; i++ {
		e := ifd + 2 + i*12
		if e+12 > len(t) {
			return 0, errBad
		}
		if u16(t[e:e+2]) == 0x0112 {
			o := u16(t[e+8 : e+10])
			if o < 1 || o > 8 {
				return 0, errBad
			}
			return o, nil
		}
	}
	return 1, nil
}

// applyOrientation rotates/flips img so it displays upright once the EXIF
// tag is gone.
func applyOrientation(img image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func NewUploadService(storage repository.ObjectStorage, maxBytes int64) UploadService {
	return &uploadService{storage: storage, maxBytes: maxBytes, now: time.Now}
}
