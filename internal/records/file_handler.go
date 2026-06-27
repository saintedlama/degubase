package records

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/infrastructure/storage"
	"github.com/saintedlama/degubase/internal/models"
)

// UploadConfig holds per-column-type upload constraints.
type UploadConfig struct {
	ImageMaxBytes     int64
	FileMaxBytes      int64
	AllowedImageMIMEs []string
	ThumbnailWidth    int
}

func (c UploadConfig) allowedImageMIME(mime string) bool {
	for _, m := range c.AllowedImageMIMEs {
		if m == mime {
			return true
		}
	}
	return false
}

// FileHandler provides file upload, download, and thumbnail generation
// scoped under row routes. It depends on the RowStore, ColumnLister, and
// the storage.Storage infrastructure.
type FileHandler struct {
	Rows    Store
	Cols    ColumnLister
	Storage storage.Storage
	Config  UploadConfig
}

// @Summary     Upload file
// @Tags        files
// @Accept      multipart/form-data
// @Produce     json
// @Param       wsCode     path      string  true  "workspace code"
// @Param       tableCode  path      string  true  "table code"
// @Param       rowID      path      int     true  "row ID"
// @Param       colCode    formData  string  true  "column code"
// @Param       file       formData  file    true  "file to upload"
// @Success     200        {object}  models.Row
// @Failure     400        {object}  http.ErrorResponse
// @Failure     404        {object}  http.ErrorResponse
// @Failure     413        {object}  http.ErrorResponse
// @Failure     500        {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID}/files [post]
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	rowID, err := strconv.ParseInt(chi.URLParam(r, "rowID"), 10, 64)
	if err != nil {
		httplib.BadRequest(w, "invalid row id")
		return
	}

	// Reject bodies larger than the bigger of the two limits before parsing.
	maxBody := max(h.Config.ImageMaxBytes, h.Config.FileMaxBytes)
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			httplib.RespondErr(w, http.StatusRequestEntityTooLarge, "file_too_large",
				fmt.Sprintf("upload exceeds maximum allowed size of %d bytes", maxBody))
		} else {
			httplib.BadRequest(w, "invalid multipart form")
		}
		return
	}

	colCode := r.FormValue("colCode")
	if colCode == "" {
		httplib.BadRequest(w, "colCode is required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httplib.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		extMime := mimeByExt(filepath.Ext(header.Filename))
		if extMime != "application/octet-stream" {
			mimeType = extMime
		}
	}

	t := httplib.TableFromCtx(r)
	cols, err := h.Cols.ListColumns(r.Context(), t.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	col := findColumnByCode(cols, colCode)
	if col == nil {
		httplib.NotFound(w)
		return
	}

	sizeLimit := h.Config.FileMaxBytes
	if col.Type == models.ColumnTypeImage {
		sizeLimit = h.Config.ImageMaxBytes
	}
	if header.Size > sizeLimit {
		httplib.RespondErr(w, http.StatusRequestEntityTooLarge, "file_too_large",
			fmt.Sprintf("file exceeds maximum size of %d bytes", sizeLimit))
		return
	}

	if col.Type == models.ColumnTypeImage && !h.Config.allowedImageMIME(mimeType) {
		httplib.RespondErr(w, http.StatusUnsupportedMediaType, "unsupported_media_type",
			fmt.Sprintf("image type %q is not accepted; allowed: %s",
				mimeType, strings.Join(h.Config.AllowedImageMIMEs, ", ")))
		return
	}

	row, err := h.Rows.GetRow(r.Context(), rowID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	if row == nil {
		httplib.NotFound(w)
		return
	}

	// DB stores data with numeric ID keys — use the ID key for read/write.
	colIDStr := strconv.FormatInt(col.ID, 10)
	var existingData map[string]interface{}
	if len(row.Data) > 0 {
		json.Unmarshal(row.Data, &existingData)
	}
	if existingData == nil {
		existingData = make(map[string]interface{})
	}
	if prev, ok := existingData[colIDStr].(map[string]interface{}); ok {
		if prevFileID, ok := prev["fileId"].(string); ok {
			_ = h.Storage.Delete(r.Context(), prevFileID)
		}
	}

	info, err := h.Storage.Store(r.Context(), header.Filename, mimeType, file)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	existingData[colIDStr] = map[string]interface{}{
		"fileId":   info.ID,
		"filename": info.Name,
		"size":     info.Size,
		"mimeType": info.MimeType,
	}
	merged, _ := json.Marshal(existingData)

	updated, _, err := h.Rows.UpdateRow(r.Context(), rowID, merged, "", cols)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	cm := buildColMaps(models.AppendVirtualColumns(t.ID, cols))
	updated.Data = translateOutgoing(updated.Data, cm.idToCode)
	httplib.Respond(w, http.StatusOK, updated)
}

// @Summary     Download file
// @Tags        files
// @Produce     octet-stream
// @Param       wsCode     path  string  true  "workspace code"
// @Param       tableCode  path  string  true  "table code"
// @Param       rowID      path  int     true  "row ID"
// @Param       fileID     path  string  true  "file ID"
// @Success     200
// @Failure     404  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID}/files/{fileID} [get]
func (h *FileHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileID")
	if fileID == "" {
		httplib.BadRequest(w, "fileID is required")
		return
	}

	reader, mimeType, err := h.Storage.Get(r.Context(), fileID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httplib.NotFound(w)
		} else {
			httplib.InternalErr(w, err)
		}
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	io.Copy(w, reader)
}

// @Summary     File thumbnail
// @Tags        files
// @Produce     image/jpeg
// @Param       wsCode     path  string  true  "workspace code"
// @Param       tableCode  path  string  true  "table code"
// @Param       rowID      path  int     true  "row ID"
// @Param       fileID     path  string  true  "file ID"
// @Success     200
// @Failure     404  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/{rowID}/files/{fileID}/thumbnail [get]
func (h *FileHandler) ServeThumbnail(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileID")
	if fileID == "" {
		httplib.BadRequest(w, "fileID is required")
		return
	}

	reader, mimeType, err := h.Storage.Thumbnail(r.Context(), fileID, h.Config.ThumbnailWidth)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not an image") {
			httplib.NotFound(w)
		} else {
			httplib.InternalErr(w, err)
		}
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	io.Copy(w, reader)
}

// DeleteRowFiles removes all files referenced in a row's data.
func (h *FileHandler) DeleteRowFiles(ctx context.Context, rowID int64) {
	row, err := h.Rows.GetRow(ctx, rowID)
	if err != nil || row == nil {
		return
	}
	var data map[string]interface{}
	if err := json.Unmarshal(row.Data, &data); err != nil {
		return
	}
	for _, v := range data {
		if meta, ok := v.(map[string]interface{}); ok {
			if fileID, ok := meta["fileId"].(string); ok {
				_ = h.Storage.Delete(ctx, fileID)
			}
		}
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
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
