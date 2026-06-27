package storage

import (
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	// Decoder-only registrations (no encoder available in stdlib/x/image for webp).
	_ "golang.org/x/image/webp"

	"golang.org/x/image/draw"
)

// thumbFormat describes how to encode a thumbnail for a given source MIME type.
type thumbFormat struct {
	ext      string
	mimeType string
	encode   func(io.Writer, image.Image) error
}

// thumbFormatFor returns the thumbnail format to use for the given source MIME.
// WebP has no encoder available; it falls back to PNG (lossless).
func thumbFormatFor(sourceMIME string) thumbFormat {
	switch sourceMIME {
	case "image/png":
		return thumbFormat{".png", "image/png", func(w io.Writer, img image.Image) error {
			return png.Encode(w, img)
		}}
	case "image/gif":
		return thumbFormat{".gif", "image/gif", func(w io.Writer, img image.Image) error {
			return gif.Encode(w, img, nil)
		}}
	default: // image/jpeg + image/webp (no webp encoder) → JPEG
		return thumbFormat{".jpg", "image/jpeg", func(w io.Writer, img image.Image) error {
			return jpeg.Encode(w, img, &jpeg.Options{Quality: 80})
		}}
	}
}

func (s *Store) Thumbnail(_ context.Context, id string, maxWidth int) (io.ReadCloser, string, error) {
	if cached := s.findThumbnailPath(id); cached != "" {
		f, err := os.Open(cached)
		if err != nil {
			return nil, "", err
		}
		return f, mimeByExt(filepath.Ext(cached)), nil
	}

	entry, err := s.findFile(id)
	if err != nil {
		return nil, "", err
	}

	if !isImage(entry.mimeType) {
		return nil, "", fmt.Errorf("not an image")
	}

	src, err := os.Open(entry.path)
	if err != nil {
		return nil, "", fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	format := thumbFormatFor(entry.mimeType)
	thumbPath := filepath.Join(s.root, "thumbnails", id+format.ext)

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= maxWidth && h <= maxWidth {
		return s.encodeThumb(thumbPath, img, format)
	}

	newW, newH := w, h
	if w > h {
		newW = maxWidth
		newH = h * maxWidth / w
	} else {
		newH = maxWidth
		newW = w * maxWidth / h
	}
	if newH < 1 {
		newH = 1
	}
	if newW < 1 {
		newW = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	return s.encodeThumb(thumbPath, dst, format)
}

func (s *Store) encodeThumb(path string, img image.Image, format thumbFormat) (io.ReadCloser, string, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, "", fmt.Errorf("create thumbnail: %w", err)
	}
	if err := format.encode(f, img); err != nil {
		f.Close()
		os.Remove(path)
		return nil, "", fmt.Errorf("encode thumbnail: %w", err)
	}
	f.Close()

	r, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	return r, format.mimeType, nil
}

func isImage(mime string) bool {
	return strings.HasPrefix(mime, "image/")
}
