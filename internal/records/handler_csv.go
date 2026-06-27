package records

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
)

// CSVHandler provides CSV export and import endpoints.
type CSVHandler struct {
	Store Store
	Cols  ColumnLister
	Svc   *RowService
}

// ── Export ────────────────────────────────────────────────────────────────────

// @Summary     Export rows as CSV
// @Tags        rows
// @Produce     text/csv
// @Param       wsCode     path   string  true   "workspace code"
// @Param       tableCode  path   string  true   "table code"
// @Param       filter     query  string  false  "shorthand filter; repeat for AND"
// @Param       sort       query  string  false  "col:dir,col:dir"
// @Param       q          query  string  false  "search query"
// @Param       select     query  string  false  "comma-separated column codes"
// @Success     200
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/export.csv [get]
func (h *CSVHandler) Export(w http.ResponseWriter, r *http.Request) {
	t := httplib.TableFromCtx(r)

	// Parse filters (same as List)
	var filters []models.RowFilter
	if shorthands := r.URL.Query()["filter"]; len(shorthands) > 0 {
		parsed, err := parseFilterShorthands(shorthands)
		if err != nil {
			httplib.BadRequest(w, err.Error())
			return
		}
		filters = parsed
	}
	if raw := r.URL.Query().Get("filters"); raw != "" {
		var jsonFilters []models.RowFilter
		if err := json.Unmarshal([]byte(raw), &jsonFilters); err != nil {
			httplib.BadRequest(w, "invalid filters")
			return
		}
		filters = append(filters, jsonFilters...)
	}

	// Parse sort
	var sorts []models.RowSort
	if raw := r.URL.Query().Get("sort"); raw != "" {
		parsed, err := parseSortShorthand(raw)
		if err != nil {
			httplib.BadRequest(w, err.Error())
			return
		}
		sorts = parsed
	}

	// Parse select
	var selectedCols []string
	if sel := r.URL.Query().Get("select"); sel != "" {
		for s := range strings.SplitSeq(sel, ",") {
			if s = strings.TrimSpace(s); s != "" {
				selectedCols = append(selectedCols, s)
			}
		}
	}

	search := r.URL.Query().Get("q")

	// Fetch all rows (up to 10k for export)
	rows, _, err := h.Svc.ListRows(r.Context(), t.ID, 10000, 0, filters, sorts, search)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	// Determine ordered columns for headers
	cols, err := h.Cols.ListColumns(r.Context(), t.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}
	// Filter to selected cols if requested, preserving order
	var exportCols []models.Column
	if len(selectedCols) > 0 {
		codeSet := make(map[string]bool, len(selectedCols))
		for _, s := range selectedCols {
			codeSet[s] = true
		}
		for _, c := range cols {
			if codeSet[c.Code] {
				exportCols = append(exportCols, c)
			}
		}
	} else {
		exportCols = cols
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, t.Name))

	cw := csv.NewWriter(w)

	// Header row: column names
	headers := make([]string, len(exportCols))
	for i, c := range exportCols {
		headers[i] = c.Name
	}
	if err := cw.Write(headers); err != nil {
		return
	}

	// Data rows
	for _, row := range rows {
		var data map[string]json.RawMessage
		if len(row.Data) > 0 {
			_ = json.Unmarshal(row.Data, &data)
		}
		record := make([]string, len(exportCols))
		for i, c := range exportCols {
			if v, ok := data[c.Code]; ok {
				record[i] = rawToCSVString(v)
			}
		}
		if err := cw.Write(record); err != nil {
			return
		}
	}

	cw.Flush()
}

// rawToCSVString converts a JSON raw value to a plain string for CSV output.
func rawToCSVString(v json.RawMessage) string {
	if len(v) == 0 {
		return ""
	}
	// Unquote JSON strings; render everything else as-is
	var s string
	if err := json.Unmarshal(v, &s); err == nil {
		return s
	}
	// Numbers, booleans, arrays → stringify
	return strings.Trim(string(v), `"`)
}

// ── Import ────────────────────────────────────────────────────────────────────

// ImportPreviewResponse is returned when no mapping is provided.
type ImportPreviewResponse struct {
	Headers          []string          `json:"headers"`
	Sample           [][]string        `json:"sample"`
	Columns          []models.Column   `json:"columns"`
	SuggestedMapping map[string]string `json:"suggested_mapping"` // csvHeader → colCode (empty = no suggestion)
}

// ImportResultResponse is returned after a successful import.
type ImportResultResponse struct {
	Imported int `json:"imported"`
}

// @Summary     Import rows from CSV
// @Tags        rows
// @Accept      multipart/form-data
// @Produce     json
// @Param       wsCode     path      string  true   "workspace code"
// @Param       tableCode  path      string  true   "table code"
// @Param       file       formData  file    true   "CSV file"
// @Param       mapping    formData  string  false  "JSON mapping: {\"CSV Header\": \"col_code\"}"
// @Success     200  {object}  ImportPreviewResponse
// @Success     201  {object}  ImportResultResponse
// @Failure     400  {object}  http.ErrorResponse
// @Failure     500  {object}  http.ErrorResponse
// @Security    BearerAuth
// @Router      /workspaces/{wsCode}/tables/{tableCode}/rows/import [post]
func (h *CSVHandler) Import(w http.ResponseWriter, r *http.Request) {
	t := httplib.TableFromCtx(r)

	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB limit
		httplib.BadRequest(w, "invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		httplib.BadRequest(w, "file field is required")
		return
	}
	defer file.Close()

	cr := csv.NewReader(file)
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true

	// Read headers
	headers, err := cr.Read()
	if err != nil {
		httplib.BadRequest(w, "failed to read CSV headers")
		return
	}

	// Read sample rows (up to 5)
	var sample [][]string
	for range 5 {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		sample = append(sample, rec)
	}

	// Load table columns
	cols, err := h.Cols.ListColumns(r.Context(), t.ID)
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	// Build suggested mapping: case-insensitive match of CSV header to column name or code
	colByName := make(map[string]string) // lower(name) → code
	colByCode := make(map[string]string) // lower(code) → code
	for _, c := range cols {
		colByName[strings.ToLower(c.Name)] = c.Code
		colByCode[strings.ToLower(c.Code)] = c.Code
	}

	suggested := make(map[string]string, len(headers))
	for _, h := range headers {
		key := strings.ToLower(strings.TrimSpace(h))
		if code, ok := colByCode[key]; ok {
			suggested[h] = code
		} else if code, ok := colByName[key]; ok {
			suggested[h] = code
		}
	}

	// No mapping provided → return preview
	mappingStr := r.FormValue("mapping")
	if mappingStr == "" {
		httplib.Respond(w, http.StatusOK, ImportPreviewResponse{
			Headers:          headers,
			Sample:           sample,
			Columns:          cols,
			SuggestedMapping: suggested,
		})
		return
	}

	// Mapping provided → perform import
	var mapping map[string]string // csvHeader → colCode (empty string = skip)
	if err := json.Unmarshal([]byte(mappingStr), &mapping); err != nil {
		httplib.BadRequest(w, "invalid mapping JSON")
		return
	}

	// Re-read the file from the start to parse all rows
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		httplib.InternalErr(w, err)
		return
	}
	cr2 := csv.NewReader(file)
	cr2.LazyQuotes = true
	cr2.TrimLeadingSpace = true

	// Skip header row
	if _, err := cr2.Read(); err != nil {
		httplib.BadRequest(w, "failed to re-read CSV")
		return
	}

	// Build header index
	headerIdx := make(map[int]string) // csvColIndex → colCode
	for i, csvHeader := range headers {
		if code, ok := mapping[csvHeader]; ok && code != "" {
			headerIdx[i] = code
		}
	}

	imported := 0
	for {
		rec, err := cr2.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		rowData := make(map[string]json.RawMessage)
		for i, val := range rec {
			if i >= len(headers) {
				break
			}
			code, ok := headerIdx[i]
			if !ok || val == "" {
				continue
			}
			rowData[code] = json.RawMessage(`"` + escapeJSONString(val) + `"`)
		}
		data, err := json.Marshal(rowData)
		if err != nil {
			continue
		}
		if _, err := h.Svc.CreateRow(r.Context(), t.ID, json.RawMessage(data)); err != nil {
			continue
		}
		imported++
	}

	httplib.Respond(w, http.StatusCreated, ImportResultResponse{Imported: imported})
}

// escapeJSONString escapes special characters for embedding in a JSON string.
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
