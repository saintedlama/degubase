package records

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
)

func TestRawToCSVString(t *testing.T) {
	tests := []struct {
		name string
		in   json.RawMessage
		want string
	}{
		{"nil", nil, ""},
		{"empty", json.RawMessage(""), ""},
		{"string", json.RawMessage(`"hello world"`), "hello world"},
		{"string with quotes", json.RawMessage(`"hello \"world\""`), `hello "world"`},
		{"integer", json.RawMessage(`42`), "42"},
		{"float", json.RawMessage(`3.14`), "3.14"},
		{"boolean true", json.RawMessage(`true`), "true"},
		{"boolean false", json.RawMessage(`false`), "false"},
		{"array", json.RawMessage(`[1, 2, 3]`), "[1, 2, 3]"},
		{"object", json.RawMessage(`{"a":1}`), `{"a":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rawToCSVString(tt.in)
			if got != tt.want {
				t.Errorf("rawToCSVString(%s) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestEscapeJSONString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"backslash", `a\b`, `a\\b`},
		{"double quote", `a"b`, `a\"b`},
		{"newline", "a\nb", `a\nb`},
		{"carriage return", "a\rb", `a\rb`},
		{"tab", "a\tb", `a\tb`},
		{"combined", "a\\\"b\n\r\t", `a\\\"b\n\r\t`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeJSONString(tt.in)
			if got != tt.want {
				t.Errorf("escapeJSONString(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

type errColLister struct{}

func (errColLister) ListColumns(ctx context.Context, tableID int64) ([]models.Column, error) {
	return nil, errors.New("col list error")
}

type okColLister struct {
	cols []models.Column
}

func (l okColLister) ListColumns(ctx context.Context, tableID int64) ([]models.Column, error) {
	return l.cols, nil
}

type stubRowStore struct {
	RowStore
	rows []models.Row
	err  error
}

func (s stubRowStore) ListRows(ctx context.Context, tableID int64, limit, offset int, filters []models.RowFilter, sorts []models.RowSort, columns []models.Column, search string) ([]models.Row, int64, error) {
	if s.err != nil {
		return nil, 0, s.err
	}
	return s.rows, int64(len(s.rows)), nil
}

func TestExportErrors(t *testing.T) {
	table := &models.Table{ID: 10, Name: "Test"}

	t.Run("ListRows failure", func(t *testing.T) {
		h := &CSVHandler{
			Cols: okColLister{},
			Svc:  &RowService{Store: stubRowStore{err: errors.New("list rows error")}, Cols: okColLister{}},
		}
		req := httptest.NewRequest("GET", "/export.csv", nil)
		req = req.WithContext(context.WithValue(req.Context(), httplib.CtxTable, table))
		rec := httptest.NewRecorder()

		h.Export(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("ListColumns failure", func(t *testing.T) {
		h := &CSVHandler{
			Cols: errColLister{},
			Svc:  &RowService{Store: stubRowStore{}, Cols: errColLister{}},
		}
		req := httptest.NewRequest("GET", "/export.csv", nil)
		req = req.WithContext(context.WithValue(req.Context(), httplib.CtxTable, table))
		rec := httptest.NewRecorder()

		h.Export(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})
}

func TestImportListColumnsError(t *testing.T) {
	table := &models.Table{ID: 10, Name: "Test"}
	h := &CSVHandler{
		Cols: errColLister{},
		Svc:  &RowService{Store: stubRowStore{}, Cols: errColLister{}},
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "test.csv")
	part.Write([]byte("col1,col2\nval1,val2\n"))
	w.Close()

	req := httptest.NewRequest("POST", "/import", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = req.WithContext(context.WithValue(req.Context(), httplib.CtxTable, table))
	rec := httptest.NewRecorder()

	h.Import(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
