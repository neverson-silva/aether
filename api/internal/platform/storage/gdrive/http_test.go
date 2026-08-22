package gdrive

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServiceClient(t *testing.T, handler http.Handler) (*driveServiceClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	client, err := newDriveServiceClient(srv.Client(), srv.URL+"/drive/v3/")
	if err != nil {
		t.Fatalf("newDriveServiceClient: %v", err)
	}
	return client, srv
}

func writeFileJSON(w http.ResponseWriter, raw string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, raw)
}

func TestServiceCreateFolder(t *testing.T) {
	var gotBody map[string]any
	client, srv := newTestServiceClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/drive/v3/files") {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		writeFileJSON(w, `{"id":"f1","name":"backups","mimeType":"application/vnd.google-apps.folder","parents":["root"],"modifiedTime":"2024-01-01T00:00:00Z"}`)
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

func TestServiceCreateFileWithBodyUsesUpload(t *testing.T) {
	var uploaded string
	client, srv := newTestServiceClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/upload/drive/v3/files") {
			t.Errorf("path = %s", r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		uploaded = string(data)
		writeFileJSON(w, `{"id":"f9","name":"db.sql.gz","mimeType":"application/gzip","size":"11","modifiedTime":"2024-01-01T00:00:00Z"}`)
	}))
	defer srv.Close()

	f, err := client.CreateFile(context.Background(), CreateFileInput{
		Name: "db.sql.gz", MimeType: "application/gzip", ParentID: "root", Body: strings.NewReader("hello world"),
	})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if !strings.Contains(uploaded, "hello world") {
		t.Errorf("uploaded body = %q", uploaded)
	}
	if f.ID != "f9" || f.Size != 11 {
		t.Errorf("file = %+v", f)
	}
}

func TestServiceListPagination(t *testing.T) {
	client, srv := newTestServiceClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("pageToken")
		if page == "" {
			writeFileJSON(w, `{"files":[{"id":"a","name":"a.txt","mimeType":"text/plain","size":"1"}],"nextPageToken":"tok2"}`)
		} else {
			writeFileJSON(w, `{"files":[{"id":"b","name":"b.txt","mimeType":"text/plain","size":"2"}]}`)
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

func TestServiceDownload(t *testing.T) {
	client, srv := newTestServiceClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestServiceDelete(t *testing.T) {
	var method, path string
	client, srv := newTestServiceClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := client.DeleteFile(context.Background(), "f1"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if method != http.MethodDelete || !strings.HasPrefix(path, "/drive/v3/files/f1") {
		t.Errorf("delete = %s %s", method, path)
	}
}

func TestServiceErrorMapping(t *testing.T) {
	client, srv := newTestServiceClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"error":{"code":404,"message":"File not found: f123."}}`)
	}))
	defer srv.Close()

	_, err := client.GetFile(context.Background(), "f123")
	if err == nil {
		t.Fatal("expected error")
	}
	mapped := mapError(err)
	if mapped == err {
		t.Errorf("mapError did not map googleapi error: %v", err)
	}
}
