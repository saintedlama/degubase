package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Store struct {
	root string
}

func NewLocal(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "thumbnails"), 0755); err != nil {
		return nil, fmt.Errorf("create thumbnails dir: %w", err)
	}
	return &Store{root: root}, nil
}

func (s *Store) Store(_ context.Context, name string, mimeType string, reader io.Reader) (*FileInfo, error) {
	id := uuid.New().String()
	ext := filepath.Ext(name)
	if ext == "" {
		ext = extByMime(mimeType)
	}
	filename := id + ext
	path := filepath.Join(s.root, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	size, err := io.Copy(f, reader)
	if err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("write file: %w", err)
	}

	return &FileInfo{
		ID:       id,
		Name:     name,
		Size:     size,
		MimeType: mimeType,
	}, nil
}

func (s *Store) Get(_ context.Context, id string) (io.ReadCloser, string, error) {
	entry, err := s.findFile(id)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(entry.path)
	if err != nil {
		return nil, "", fmt.Errorf("open file: %w", err)
	}
	return f, entry.mimeType, nil
}

func (s *Store) Delete(_ context.Context, id string) error {
	entry, err := s.findFile(id)
	if err != nil {
		return nil // idempotent: already gone
	}
	if err := os.Remove(entry.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if thumb := s.findThumbnailPath(id); thumb != "" {
		os.Remove(thumb)
	}
	return nil
}

func (s *Store) findThumbnailPath(id string) string {
	entries, err := os.ReadDir(filepath.Join(s.root, "thumbnails"))
	if err != nil {
		return ""
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, id+".") || name == id {
			return filepath.Join(s.root, "thumbnails", name)
		}
	}
	return ""
}

type fileEntry struct {
	path     string
	mimeType string
}

func (s *Store) findFile(id string) (fileEntry, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return fileEntry{}, fmt.Errorf("read storage dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, id+".") || name == id {
			return fileEntry{
				path:     filepath.Join(s.root, name),
				mimeType: mimeByExt(filepath.Ext(name)),
			}, nil
		}
	}
	return fileEntry{}, fmt.Errorf("file not found: %s", id)
}

func extByMime(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return ".bin"
	}
}

func mimeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	default:
		return "application/octet-stream"
	}
}
