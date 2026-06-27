package records_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saintedlama/degubase/internal/testutil"
)

func testJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{0, 100, 200, 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func uploadFile(t *testing.T, srv *httptest.Server, workspaceCode, tableCode string, rowID float64, columnCode string, filename string, data []byte, mimeType string) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	_ = w.WriteField("colCode", columnCode)

	h := make(map[string][]string)
	h["Content-Type"] = []string{mimeType}
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)}
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	w.Close()

	url := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/%.0f/files", srv.URL, workspaceCode, tableCode, rowID)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	var out any
	json.NewDecoder(resp.Body).Decode(&out)
	if m, ok := out.(map[string]any); ok {
		return resp.StatusCode, m
	}
	return resp.StatusCode, nil
}

func getFile(t *testing.T, srv *httptest.Server, workspaceCode, tableCode string, rowID float64, fileID string) (int, []byte, string) {
	t.Helper()
	url := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/%.0f/files/%s", srv.URL, workspaceCode, tableCode, rowID, fileID)
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data, resp.Header.Get("Content-Type")
}

func TestFileUpload(t *testing.T) {
	srv, _ := testutil.NewServerWithStorage(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	cb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{
		"name": "attachment",
		"type": "file",
	}, http.StatusCreated)
	columnCode := testutil.Field(t, testutil.Obj(t, cb), "code")

	rb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{}, http.StatusCreated)
	rowID := testutil.Obj(t, rb)["id"].(float64)

	t.Run("upload text file", func(t *testing.T) {
		code, resp := uploadFile(t, srv, wsCode, tblCode, rowID, columnCode, "hello.txt", []byte("hello world"), "text/plain")
		if code != http.StatusOK {
			t.Fatalf("upload: want 200, got %d", code)
		}

		data := testutil.Obj(t, resp["data"])
		fileMeta := testutil.Obj(t, data[columnCode])
		if testutil.Field(t, fileMeta, "filename") != "hello.txt" {
			t.Errorf("filename = %q, want hello.txt", testutil.Field(t, fileMeta, "filename"))
		}
		if fileMeta["fileId"] == nil || fileMeta["fileId"] == "" {
			t.Error("fileId should not be empty")
		}
		if fileMeta["size"] == nil {
			t.Error("size should not be nil")
		}
		if testutil.Field(t, fileMeta, "mimeType") != "text/plain" {
			t.Errorf("mimeType = %q, want text/plain", testutil.Field(t, fileMeta, "mimeType"))
		}
	})
}

func TestFileServe(t *testing.T) {
	srv, _ := testutil.NewServerWithStorage(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	cb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{
		"name": "file",
		"type": "file",
	}, http.StatusCreated)
	columnCode := testutil.Field(t, testutil.Obj(t, cb), "code")

	rb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{}, http.StatusCreated)
	rowID := testutil.Obj(t, rb)["id"].(float64)

	fileData := []byte("served content")
	_, resp := uploadFile(t, srv, wsCode, tblCode, rowID, columnCode, "serve.txt", fileData, "text/plain")
	data := testutil.Obj(t, resp["data"])
	fileMeta := testutil.Obj(t, data[columnCode])
	fileID := testutil.Field(t, fileMeta, "fileId")

	t.Run("serve uploaded file", func(t *testing.T) {
		code, body, mime := getFile(t, srv, wsCode, tblCode, rowID, fileID)
		if code != http.StatusOK {
			t.Fatalf("serve: want 200, got %d", code)
		}
		if !bytes.Equal(body, fileData) {
			t.Errorf("body = %q, want %q", body, fileData)
		}
		if mime != "text/plain" {
			t.Errorf("mime = %q, want text/plain", mime)
		}
	})

	t.Run("serve non-existent file returns 404", func(t *testing.T) {
		code, _, _ := getFile(t, srv, wsCode, tblCode, rowID, "non-existent-id")
		if code != http.StatusNotFound {
			t.Errorf("want 404, got %d", code)
		}
	})
}

func TestFileThumbnail(t *testing.T) {
	srv, _ := testutil.NewServerWithStorage(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	cb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{
		"name": "photo",
		"type": "image",
	}, http.StatusCreated)
	columnCode := testutil.Field(t, testutil.Obj(t, cb), "code")

	rb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{}, http.StatusCreated)
	rowID := testutil.Obj(t, rb)["id"].(float64)

	_, resp := uploadFile(t, srv, wsCode, tblCode, rowID, columnCode, "photo.jpg", testJPEG(), "image/jpeg")
	data := testutil.Obj(t, resp["data"])
	fileMeta := testutil.Obj(t, data[columnCode])
	fileID := testutil.Field(t, fileMeta, "fileId")

	t.Run("thumbnail returns JPEG", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/%.0f/files/%s/thumbnail", srv.URL, wsCode, tblCode, rowID, fileID)
		resp, err := http.DefaultClient.Get(url)
		if err != nil {
			t.Fatalf("get thumbnail: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", resp.StatusCode)
		}
		if resp.Header.Get("Content-Type") != "image/jpeg" {
			t.Errorf("mime = %q, want image/jpeg", resp.Header.Get("Content-Type"))
		}

		img, _, err := image.Decode(resp.Body)
		if err != nil {
			t.Fatalf("decode thumbnail: %v", err)
		}
		bounds := img.Bounds()
		if bounds.Dx() > 256 || bounds.Dy() > 256 {
			t.Errorf("thumbnail %dx%d exceeds 256 max", bounds.Dx(), bounds.Dy())
		}
	})

	t.Run("thumbnail for non-existent returns 404", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/workspaces/%s/tables/%s/rows/%.0f/files/nope/thumbnail", srv.URL, wsCode, tblCode, rowID)
		resp, err := http.DefaultClient.Get(url)
		if err != nil {
			t.Fatalf("get thumbnail: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("want 404, got %d", resp.StatusCode)
		}
	})
}

func TestFileUploadReplacesPrevious(t *testing.T) {
	srv, _ := testutil.NewServerWithStorage(t)

	wb := testutil.MustReq(t, srv, "POST", "/api/workspaces/", map[string]any{"name": "ws"}, http.StatusCreated)
	wsCode := testutil.Field(t, testutil.Obj(t, wb), "code")
	tb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/", wsCode), map[string]any{"name": "t"}, http.StatusCreated)
	tblCode := testutil.Field(t, testutil.Obj(t, tb), "code")

	cb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/columns/", wsCode, tblCode), map[string]any{
		"name": "doc",
		"type": "file",
	}, http.StatusCreated)
	columnCode := testutil.Field(t, testutil.Obj(t, cb), "code")

	rb := testutil.MustReq(t, srv, "POST", fmt.Sprintf("/api/workspaces/%s/tables/%s/rows/", wsCode, tblCode), map[string]any{}, http.StatusCreated)
	rowID := testutil.Obj(t, rb)["id"].(float64)

	_, resp1 := uploadFile(t, srv, wsCode, tblCode, rowID, columnCode, "first.txt", []byte("first"), "text/plain")
	data1 := testutil.Obj(t, resp1["data"])
	meta1 := testutil.Obj(t, data1[columnCode])
	fileID1 := testutil.Field(t, meta1, "fileId")

	_, resp2 := uploadFile(t, srv, wsCode, tblCode, rowID, columnCode, "second.txt", []byte("second"), "text/plain")
	data2 := testutil.Obj(t, resp2["data"])
	meta2 := testutil.Obj(t, data2[columnCode])
	fileID2 := testutil.Field(t, meta2, "fileId")

	if fileID1 == fileID2 {
		t.Errorf("fileIds should differ after replacement")
	}
	if testutil.Field(t, meta2, "filename") != "second.txt" {
		t.Errorf("filename = %q, want second.txt", testutil.Field(t, meta2, "filename"))
	}

	code, _, _ := getFile(t, srv, wsCode, tblCode, rowID, fileID1)
	if code != http.StatusNotFound {
		t.Errorf("old file should be deleted, got status %d", code)
	}

	code, body, _ := getFile(t, srv, wsCode, tblCode, rowID, fileID2)
	if code != http.StatusOK {
		t.Fatalf("new file should exist, got %d", code)
	}
	if !bytes.Equal(body, []byte("second")) {
		t.Errorf("new file content = %q, want 'second'", body)
	}
}
