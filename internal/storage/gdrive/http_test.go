package gdrive

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestHTTPClient(handler http.Handler) (*driveHTTPClient, *httptest.Server) {
	srv := httptest.NewServer(handler)
	return newDriveHTTPClient(srv.Client(), srv.URL, srv.URL), srv
}

func absURL(srv *httptest.Server, p string) string {
	return srv.URL + p
}

func TestHTTPCreateFolder(t *testing.T) {
	var gotBody map[string]any
	client, srv := newTestHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/drive/v3/files") {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"id":"f1","name":"backups","mimeType":"application/vnd.google-apps.folder","parents":["root"],"modifiedTime":"2024-01-01T00:00:00Z"}`)
	}))
	defer srv.Close()

	f, err := client.CreateFile(context.Background(), CreateFileInput{
		Name: "backups", MimeType: folderMIME, ParentID: "root",
	})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if f.ID != "f1" || f.MimeType != folderMIME {
		t.Errorf("file = %+v", f)
	}
	if gotBody["name"] != "backups" {
		t.Errorf("request body = %v", gotBody)
	}
}

func TestHTTPUpdateFileWithBodyUsesResumable(t *testing.T) {
	var initMethod string
	var initPath string
	var contentLength int64
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Query().Get("uploadType") == "resumable":
			initMethod = r.Method
			initPath = r.URL.Path
			w.Header().Set("Location", absURL(srv, "/update-session-1"))
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == "/update-session-1":
			contentLength = r.ContentLength
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"id":"f5","name":"db.sql.gz","mimeType":"application/gzip","size":5,"modifiedTime":"2024-01-01T00:00:00Z"}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	client := newDriveHTTPClient(srv.Client(), srv.URL, srv.URL)

	f, err := client.UpdateFile(context.Background(), "f5", UpdateFileInput{
		ContentType: "application/gzip", Body: bytes.NewReader([]byte("hello")),
	})
	if err != nil {
		t.Fatalf("UpdateFile: %v", err)
	}
	if initMethod != http.MethodPatch {
		t.Errorf("resumable init method = %s, want PATCH", initMethod)
	}
	if initPath != "/drive/v3/files/f5" {
		t.Errorf("resumable init path = %s, want /drive/v3/files/f5", initPath)
	}
	if contentLength != 5 {
		t.Errorf("ContentLength = %d, want 5", contentLength)
	}
	if f.ID != "f5" || f.Size != 5 {
		t.Errorf("file = %+v", f)
	}
}

func TestHTTPResumableUploadOneShot(t *testing.T) {
	var sessionStarted bool
	var putBody string
	var contentLength int64
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Query().Get("uploadType"), "resumable"):
			sessionStarted = true
			if got := r.Header.Get("X-Upload-Content-Type"); got != "application/gzip" {
				t.Errorf("X-Upload-Content-Type = %q", got)
			}
			if got := r.Header.Get("X-Upload-Content-Length"); got != "11" {
				t.Errorf("X-Upload-Content-Length = %q", got)
			}
			w.Header().Set("Location", absURL(srv, "/upload-session-1"))
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == "/upload-session-1":
			contentLength = r.ContentLength
			data, _ := io.ReadAll(r.Body)
			putBody = string(data)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"id":"f9","name":"db.sql.gz","mimeType":"application/gzip","size":11,"modifiedTime":"2024-01-01T00:00:00Z"}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	client := newDriveHTTPClient(srv.Client(), srv.URL, srv.URL)

	f, err := client.CreateFile(context.Background(), CreateFileInput{
		Name: "db.sql.gz", MimeType: "application/gzip", ContentType: "application/gzip",
		ParentID: "root", Body: bytes.NewReader([]byte("hello world")),
	})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if !sessionStarted {
		t.Fatal("resumable session not started")
	}
	if contentLength != 11 {
		t.Errorf("ContentLength = %d, want 11", contentLength)
	}
	if putBody != "hello world" {
		t.Errorf("uploaded body = %q", putBody)
	}
	if f.ID != "f9" || f.Size != 11 {
		t.Errorf("file = %+v", f)
	}
}

func TestHTTPResumableUploadChunked(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Location", absURL(srv, "/upload-session-2"))
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPut || r.URL.Path != "/upload-session-2" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		rng := r.Header.Get("Content-Range")
		if !strings.Contains(rng, "bytes 0-") && !strings.Contains(rng, "/*") && !strings.Contains(rng, "256000") {
			if rng != "bytes 0-262143/*" && rng != "bytes 0-255999/*" && rng != "bytes 0-1000/*" {
				t.Errorf("unexpected Content-Range %q", rng)
			}
		}
		if strings.Contains(rng, "/2500") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"id":"fc","name":"chunk.bin","mimeType":"application/octet-stream","size":2500,"modifiedTime":"2024-01-01T00:00:00Z"}`)
			return
		}
		w.WriteHeader(http.StatusPartialContent)
	}))
	defer srv.Close()
	client := newDriveHTTPClient(srv.Client(), srv.URL, srv.URL)

	data := bytes.Repeat([]byte("x"), 2500)
	media := &notSizedReader{r: bytes.NewReader(data)}
	f, err := client.CreateFile(context.Background(), CreateFileInput{
		Name: "chunk.bin", MimeType: "application/octet-stream", ParentID: "root", Body: media,
	})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if f.Size != 2500 {
		t.Errorf("size = %d", f.Size)
	}
}

func TestHTTPErrorEnvelope(t *testing.T) {
	client, srv := newTestHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"error":{"code":404,"message":"File not found: f123."}}`)
	}))
	defer srv.Close()

	_, err := client.GetFile(context.Background(), "f123")
	if err == nil {
		t.Fatal("expected error")
	}
	gerr, ok := err.(*googleError)
	if !ok {
		t.Fatalf("err type = %T", err)
	}
	if gerr.Code != 404 || !strings.Contains(gerr.Message, "File not found") {
		t.Errorf("googleError = %+v", gerr)
	}
}

func TestHTTPDownload(t *testing.T) {
	client, srv := newTestHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("alt") != "media" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "streamed-content")
	}))
	defer srv.Close()

	body, err := client.DownloadFile(context.Background(), "f1")
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if string(data) != "streamed-content" {
		t.Errorf("body = %q", data)
	}
}

func TestHTTPListPagination(t *testing.T) {
	client, srv := newTestHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("pageToken")
		w.Header().Set("Content-Type", "application/json")
		if page == "" {
			io.WriteString(w, `{"files":[{"id":"a","name":"a.txt","mimeType":"text/plain","size":1}],"nextPageToken":"tok2"}`)
		} else {
			io.WriteString(w, `{"files":[{"id":"b","name":"b.txt","mimeType":"text/plain","size":2}]}`)
		}
	}))
	defer srv.Close()

	out, err := client.ListFiles(context.Background(), ListFilesInput{ParentID: "root", PageSize: 1})
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(out.Files) != 1 || out.Files[0].Name != "a.txt" {
		t.Errorf("first page = %+v", out)
	}
	if out.NextPageToken != "tok2" {
		t.Errorf("next token = %q", out.NextPageToken)
	}
	out2, err := client.ListFiles(context.Background(), ListFilesInput{ParentID: "root", PageSize: 1, PageToken: out.NextPageToken})
	if err != nil {
		t.Fatalf("ListFiles page2: %v", err)
	}
	if len(out2.Files) != 1 || out2.Files[0].Name != "b.txt" {
		t.Errorf("second page = %+v", out2)
	}
}

// notSizedReader hides Len/Size so the upload path falls back to chunked mode.
type notSizedReader struct{ r *bytes.Reader }

func (n *notSizedReader) Read(p []byte) (int, error) { return n.r.Read(p) }
