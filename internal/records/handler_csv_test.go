package records_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/saintedlama/degubase/internal/records"
	"github.com/saintedlama/degubase/internal/testutil"
)

func setupCSVTable(t *testing.T, srv *httptest.Server) (string, string, map[string]string) {
	t.Helper()
	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "csv-ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")

	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "People"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	colNames := []struct {
		name string
		typ  string
	}{
		{"Name", "text"},
		{"Age", "number"},
		{"Active", "checkbox"},
	}

	colsMap := make(map[string]string)
	for _, c := range colNames {
		cb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{
			"name": c.name,
			"type": c.typ,
		}, http.StatusCreated)
		code := testutil.Field(t, testutil.Obj(t, cb), "code")
		colsMap[c.name] = code
	}

	return wsCode, tblCode, colsMap
}

func createMultipartCSV(filename string, fileContent []byte, mapping string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if mapping != "" {
		if err := w.WriteField("mapping", mapping); err != nil {
			return nil, "", err
		}
	}

	if filename != "" {
		part, err := w.CreateFormFile("file", filename)
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(fileContent); err != nil {
			return nil, "", err
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

func TestCSVExportBasic(t *testing.T) {
	srv := testutil.NewServer(t)
	wsCode, tblCode, colsMap := setupCSVTable(t, srv)

	// Insert rows
	testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{
		"data": map[string]any{
			colsMap["Name"]:   "Alice",
			colsMap["Age"]:    30,
			colsMap["Active"]: true,
		},
	}, http.StatusCreated)

	testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{
		"data": map[string]any{
			colsMap["Name"]:   "Bob",
			colsMap["Age"]:    25,
			colsMap["Active"]: false,
		},
	}, http.StatusCreated)

	urlPath := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/export.csv", srv.URL, wsCode, tblCode)
	resp, err := http.Get(urlPath)
	if err != nil {
		t.Fatalf("get export: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("expected Content-Type text/csv, got %s", ct)
	}

	cd := resp.Header.Get("Content-Disposition")
	if !strings.Contains(cd, `attachment; filename="People.csv"`) {
		t.Errorf("expected Content-Disposition People.csv, got %s", cd)
	}

	r := csv.NewReader(resp.Body)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records (1 header + 2 rows), got %d", len(records))
	}

	// Verify headers contain Name, Age, Active
	headers := records[0]
	if headers[0] != "Name" || headers[1] != "Age" || headers[2] != "Active" {
		t.Errorf("unexpected headers: %v", headers)
	}
}

func TestCSVExportQueryParams(t *testing.T) {
	srv := testutil.NewServer(t)
	wsCode, tblCode, colsMap := setupCSVTable(t, srv)

	testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{
		"data": map[string]any{
			colsMap["Name"]:   "Alice",
			colsMap["Age"]:    30,
			colsMap["Active"]: true,
		},
	}, http.StatusCreated)

	testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{
		"data": map[string]any{
			colsMap["Name"]:   "Bob",
			colsMap["Age"]:    25,
			colsMap["Active"]: false,
		},
	}, http.StatusCreated)

	base := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/export.csv", srv.URL, wsCode, tblCode)

	t.Run("select single column", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s?select=%s", base, colsMap["Name"]))
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		recs, _ := csv.NewReader(resp.Body).ReadAll()
		if len(recs) != 3 {
			t.Fatalf("expected 3 records, got %d", len(recs))
		}
		if len(recs[0]) != 1 || recs[0][0] != "Name" {
			t.Errorf("expected single header 'Name', got %v", recs[0])
		}
	})

	t.Run("select multiple columns with whitespace", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s?select=%%20%s%%20,%%20%s%%20", base, colsMap["Active"], colsMap["Name"]))
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		recs, _ := csv.NewReader(resp.Body).ReadAll()
		if len(recs) != 3 {
			t.Fatalf("expected 3 records, got %d", len(recs))
		}
		// Table order preserved: Name then Active
		if len(recs[0]) != 2 || recs[0][0] != "Name" || recs[0][1] != "Active" {
			t.Errorf("expected [Name Active], got %v", recs[0])
		}
	})

	t.Run("filter shorthand valid", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s?filter=%s:is:Alice", base, colsMap["Name"]))
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		recs, _ := csv.NewReader(resp.Body).ReadAll()
		if len(recs) != 2 {
			t.Fatalf("expected 2 records (1 header + 1 row), got %d", len(recs))
		}
	})

	t.Run("filter shorthand invalid", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s?filter=invalid_without_colon", base))
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for bad filter, got %d", resp.StatusCode)
		}
	})

	t.Run("filters JSON valid", func(t *testing.T) {
		fJSON := fmt.Sprintf(`[{"col":"%s","op":"is","value":"Bob"}]`, colsMap["Name"])
		req, _ := http.NewRequestWithContext(context.Background(), "GET", base+"?filters="+url.QueryEscape(fJSON), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		recs, _ := csv.NewReader(resp.Body).ReadAll()
		if len(recs) != 2 {
			t.Fatalf("expected 2 records, got %d", len(recs))
		}
		if recs[1][0] != "Bob" {
			t.Errorf("expected Bob, got %s", recs[1][0])
		}
	})

	t.Run("filters JSON invalid", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(context.Background(), "GET", base+"?filters=invalid-json", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for bad filters JSON, got %d", resp.StatusCode)
		}
	})

	t.Run("sort shorthand valid", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s?sort=%s:desc", base, colsMap["Name"]))
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		recs, _ := csv.NewReader(resp.Body).ReadAll()
		if len(recs) != 3 {
			t.Fatalf("expected 3 records, got %d", len(recs))
		}
		// Bob before Alice in desc
		if recs[1][0] != "Bob" || recs[2][0] != "Alice" {
			t.Errorf("expected Bob then Alice, got %s then %s", recs[1][0], recs[2][0])
		}
	})

	t.Run("sort shorthand invalid", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s?sort=%s:bad-dir", base, colsMap["Name"]))
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for bad sort, got %d", resp.StatusCode)
		}
	})

	t.Run("search query", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s?q=Alice", base))
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		recs, _ := csv.NewReader(resp.Body).ReadAll()
		if len(recs) != 2 {
			t.Fatalf("expected 2 records, got %d", len(recs))
		}
	})
}

func TestCSVExportEmptyTable(t *testing.T) {
	srv := testutil.NewServer(t)
	wsCode, tblCode, _ := setupCSVTable(t, srv)

	urlPath := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/export.csv", srv.URL, wsCode, tblCode)
	resp, err := http.Get(urlPath)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	recs, err := csv.NewReader(resp.Body).ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record (headers only), got %d", len(recs))
	}
}

func TestCSVImportInvalidForm(t *testing.T) {
	srv := testutil.NewServer(t)
	wsCode, tblCode, _ := setupCSVTable(t, srv)

	urlPath := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/import", srv.URL, wsCode, tblCode)
	resp, err := http.Post(urlPath, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid multipart form, got %d", resp.StatusCode)
	}
}

func TestCSVImportMissingFile(t *testing.T) {
	srv := testutil.NewServer(t)
	wsCode, tblCode, _ := setupCSVTable(t, srv)

	body, ct, err := createMultipartCSV("", nil, `{"Name":"name"}`)
	if err != nil {
		t.Fatalf("create multipart: %v", err)
	}

	urlPath := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/import", srv.URL, wsCode, tblCode)
	resp, err := http.Post(urlPath, ct, body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing file field, got %d", resp.StatusCode)
	}
}

func TestCSVImportEmptyFile(t *testing.T) {
	srv := testutil.NewServer(t)
	wsCode, tblCode, _ := setupCSVTable(t, srv)

	body, ct, err := createMultipartCSV("empty.csv", []byte(""), "")
	if err != nil {
		t.Fatalf("create multipart: %v", err)
	}

	urlPath := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/import", srv.URL, wsCode, tblCode)
	resp, err := http.Post(urlPath, ct, body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty CSV file, got %d", resp.StatusCode)
	}
}

func TestCSVImportPreview(t *testing.T) {
	srv := testutil.NewServer(t)
	wsCode, tblCode, colsMap := setupCSVTable(t, srv)

	csvData := `name,Age,active,extra_col
Row1,10,true,v1
Row2,20,false,v2
Row3,30,true,v3
Row4,40,false,v4
Row5,50,true,v5
Row6,60,false,v6
Row7,70,true,v7
`
	body, ct, err := createMultipartCSV("test.csv", []byte(csvData), "")
	if err != nil {
		t.Fatalf("create multipart: %v", err)
	}

	urlPath := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/import", srv.URL, wsCode, tblCode)
	resp, err := http.Post(urlPath, ct, body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var preview records.ImportPreviewResponse
	if err := json.NewDecoder(resp.Body).Decode(&preview); err != nil {
		t.Fatalf("decode preview: %v", err)
	}

	if len(preview.Headers) != 4 {
		t.Errorf("expected 4 headers, got %d", len(preview.Headers))
	}
	// Sample rows capped at 5
	if len(preview.Sample) != 5 {
		t.Errorf("expected 5 sample rows, got %d", len(preview.Sample))
	}
	if len(preview.Columns) != 3 {
		t.Errorf("expected 3 table columns, got %d", len(preview.Columns))
	}

	// Suggested mappings
	if preview.SuggestedMapping["name"] != colsMap["Name"] {
		t.Errorf("expected name mapped to %s, got %s", colsMap["Name"], preview.SuggestedMapping["name"])
	}
	if preview.SuggestedMapping["Age"] != colsMap["Age"] {
		t.Errorf("expected Age mapped to %s, got %s", colsMap["Age"], preview.SuggestedMapping["Age"])
	}
	if preview.SuggestedMapping["active"] != colsMap["Active"] {
		t.Errorf("expected active mapped to %s, got %s", colsMap["Active"], preview.SuggestedMapping["active"])
	}
	if _, ok := preview.SuggestedMapping["extra_col"]; ok {
		t.Errorf("expected no mapping for extra_col, got %s", preview.SuggestedMapping["extra_col"])
	}
}

func TestCSVImportInvalidMappingJSON(t *testing.T) {
	srv := testutil.NewServer(t)
	wsCode, tblCode, _ := setupCSVTable(t, srv)

	csvData := "Name,Age\nAlice,30\n"
	body, ct, err := createMultipartCSV("test.csv", []byte(csvData), "{not-json")
	if err != nil {
		t.Fatalf("create multipart: %v", err)
	}

	urlPath := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/import", srv.URL, wsCode, tblCode)
	resp, err := http.Post(urlPath, ct, body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid mapping JSON, got %d", resp.StatusCode)
	}
}

func TestCSVImportExecute(t *testing.T) {
	srv := testutil.NewServer(t)
	wsCode, tblCode, colsMap := setupCSVTable(t, srv)

	csvData := `CSV Name,CSV Age,CSV Active,Unmapped
"Dan",40,true,"Hello\nWorld"
"Eve",22,false,"Quoted \"value\" and backslash \\"
"Frank",55,true,"Tab	separated"
"Grace",,"",""
`
	mapping := map[string]string{
		"CSV Name":   colsMap["Name"],
		"CSV Age":    colsMap["Age"],
		"CSV Active": colsMap["Active"],
		"Unmapped":   "", // explicitly skip
	}
	mapJSON, _ := json.Marshal(mapping)

	body, ct, err := createMultipartCSV("data.csv", []byte(csvData), string(mapJSON))
	if err != nil {
		t.Fatalf("create multipart: %v", err)
	}

	urlPath := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/import", srv.URL, wsCode, tblCode)
	resp, err := http.Post(urlPath, ct, body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, raw)
	}

	var result records.ImportResultResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Imported != 4 {
		t.Errorf("expected 4 imported rows, got %d", result.Imported)
	}

	// Verify rows in table
	rowsBase := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/", srv.URL, wsCode, tblCode)
	listResp, err := http.Get(rowsBase)
	if err != nil {
		t.Fatalf("get rows: %v", err)
	}
	defer listResp.Body.Close()

	var listData struct {
		Data []struct {
			Data map[string]any `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listData); err != nil {
		t.Fatalf("decode rows: %v", err)
	}
	if len(listData.Data) != 4 {
		t.Fatalf("expected 4 rows in database, got %d", len(listData.Data))
	}
}
