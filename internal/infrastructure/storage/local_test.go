package storage_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"testing"

	storage "github.com/saintedlama/degubase/internal/infrastructure/storage"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "degubase-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestStoreAndGet(t *testing.T) {
	dir := tempDir(t)
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	data := []byte("hello, world!")
	reader := bytes.NewReader(data)

	info, err := st.Store(ctx, "test.txt", "text/plain", reader)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if info.ID == "" {
		t.Error("expected non-empty ID")
	}
	if info.Name != "test.txt" {
		t.Errorf("Name = %q, want %q", info.Name, "test.txt")
	}
	if info.Size != int64(len(data)) {
		t.Errorf("Size = %d, want %d", info.Size, len(data))
	}
	if info.MimeType != "text/plain" {
		t.Errorf("MimeType = %q, want %q", info.MimeType, "text/plain")
	}

	// Verify file exists on disk
	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		t.Fatal("no files in storage dir")
	}

	// Download via Get
	r, mime, err := st.Get(ctx, info.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer r.Close()

	if mime != "text/plain" {
		t.Errorf("Get mime = %q, want %q", mime, "text/plain")
	}

	got, _ := io.ReadAll(r)
	if !bytes.Equal(got, data) {
		t.Errorf("Get data = %q, want %q", got, data)
	}
}

func TestStorePreservesExtension(t *testing.T) {
	dir := tempDir(t)
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	data := bytes.Repeat([]byte("x"), 100)
	reader := bytes.NewReader(data)

	info, err := st.Store(ctx, "photo.jpg", "image/jpeg", reader)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if e.Name() == info.ID+".jpg" {
			found = true
			break
		}
	}
	if !found {
		t.Error("file not found with .jpg extension on disk")
	}
}

func TestStoreWithoutExtension(t *testing.T) {
	dir := tempDir(t)
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	data := bytes.Repeat([]byte("x"), 100)
	reader := bytes.NewReader(data)

	info, err := st.Store(ctx, "datafile", "application/pdf", reader)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if e.Name() == info.ID+".pdf" {
			found = true
			break
		}
	}
	if !found {
		t.Error("file not found with .pdf extension inferred from mime")
	}
}

func TestGetNotFound(t *testing.T) {
	dir := tempDir(t)
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, _, err = st.Get(context.Background(), "nonexistent-id")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestDelete(t *testing.T) {
	dir := tempDir(t)
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	data := bytes.Repeat([]byte("x"), 100)
	info, err := st.Store(ctx, "temp.bin", "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	if err := st.Delete(ctx, info.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, _, err = st.Get(ctx, info.ID)
	if err == nil {
		t.Error("expected error after delete")
	}

	// Delete non-existent should not error
	if err := st.Delete(ctx, "nonexistent"); err != nil {
		t.Errorf("Delete non-existent should not error: %v", err)
	}
}

func TestThumbnail(t *testing.T) {
	dir := tempDir(t)
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	// Create a test image (200x100 red rectangle)
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode test image: %v", err)
	}

	info, err := st.Store(ctx, "test.jpg", "image/jpeg", &buf)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	// Generate thumbnail
	r, mime, err := st.Thumbnail(ctx, info.ID, 50)
	if err != nil {
		t.Fatalf("Thumbnail: %v", err)
	}
	defer r.Close()

	if mime != "image/jpeg" {
		t.Errorf("thumbnail mime = %q, want image/jpeg", mime)
	}

	// Decode thumbnail and check dimensions
	thumb, _, err := image.Decode(r)
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}
	bounds := thumb.Bounds()
	if bounds.Dx() > 50 || bounds.Dy() > 50 {
		t.Errorf("thumbnail size %dx%d exceeds max 50", bounds.Dx(), bounds.Dy())
	}
	// Original is 200x100, so thumbnail should be 50x25
	if bounds.Dx() != 50 {
		t.Errorf("thumbnail width = %d, want 50", bounds.Dx())
	}
	if bounds.Dy() != 25 {
		t.Errorf("thumbnail height = %d, want 25", bounds.Dy())
	}

	// Second call should fetch from cache
	r2, _, err := st.Thumbnail(ctx, info.ID, 50)
	if err != nil {
		t.Fatalf("Thumbnail (cached): %v", err)
	}
	r2.Close()

	// Verify thumbnail file exists on disk
	thumbPath := filepath.Join(dir, "thumbnails", info.ID+".jpg")
	if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
		t.Error("thumbnail cache file not found on disk")
	}
}

func TestThumbnailSmallImage(t *testing.T) {
	dir := tempDir(t)
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	// Create a small image (20x20) that doesn't need resizing
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode test image: %v", err)
	}

	info, err := st.Store(ctx, "small.jpg", "image/jpeg", &buf)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	r, _, err := st.Thumbnail(ctx, info.ID, 256)
	if err != nil {
		t.Fatalf("Thumbnail: %v", err)
	}
	defer r.Close()

	thumb, _, err := image.Decode(r)
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}
	bounds := thumb.Bounds()
	if bounds.Dx() != 20 || bounds.Dy() != 20 {
		t.Errorf("small image should not be resized, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestThumbnailNonImage(t *testing.T) {
	dir := tempDir(t)
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	data := []byte("plain text file")
	info, err := st.Store(ctx, "readme.txt", "text/plain", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	_, _, err = st.Thumbnail(ctx, info.ID, 256)
	if err == nil {
		t.Error("expected error generating thumbnail for non-image")
	}
}

func TestDeleteRemovesThumbnail(t *testing.T) {
	dir := tempDir(t)
	st, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	img := image.NewRGBA(image.Rect(0, 0, 200, 150))
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)

	info, err := st.Store(ctx, "photo.jpg", "image/jpeg", &buf)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	// Generate thumbnail
	r, _, err := st.Thumbnail(ctx, info.ID, 50)
	if err != nil {
		t.Fatalf("Thumbnail: %v", err)
	}
	r.Close()

	// Delete origin
	if err := st.Delete(ctx, info.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Thumbnail should also be removed
	thumbPath := filepath.Join(dir, "thumbnails", info.ID+".jpg")
	if _, err := os.Stat(thumbPath); !os.IsNotExist(err) {
		t.Error("thumbnail should be removed when origin is deleted")
	}
}
