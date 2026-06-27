// package storage manages file storage, retrieval, deletion, and thumbnail generation.
package storage

import (
	"context"
	"io"
)

// FileInfo describes a stored file.
type FileInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

// Storage is the file persistence contract.
type Storage interface {
	Store(ctx context.Context, name string, mimeType string, reader io.Reader) (*FileInfo, error)
	Get(ctx context.Context, id string) (io.ReadCloser, string, error)
	Delete(ctx context.Context, id string) error
	Thumbnail(ctx context.Context, id string, maxWidth int) (io.ReadCloser, string, error)
}
