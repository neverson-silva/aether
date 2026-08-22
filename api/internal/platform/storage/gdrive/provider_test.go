package gdrive

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/api/googleapi"

	"aether/internal/platform/storage"
)

func newTestProvider(t *testing.T) (*Provider, *fakeDriveClient) {
	t.Helper()
	client := newFakeDriveClient("root")
	return providerWithClient(client, "root"), client
}

func TestNewProviderValidation(t *testing.T) {
	if _, err := NewProvider(Config{}); err == nil {
		t.Error("expected error for missing client")
	}
	if _, err := NewProvider(Config{Client: &http.Client{}}); err == nil {
		t.Error("expected error for empty root folder")
	}
	p, err := NewProvider(Config{Client: &http.Client{}, RootFolderID: "root"})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if p.rootID != "root" {
		t.Errorf("rootID = %q", p.rootID)
	}
}

func TestETagEmptyWhenProviderHasNone(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()
	if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: "e.txt", Body: strings.NewReader("x")}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	head, err := p.HeadObject(ctx, storage.HeadObjectInput{Key: "e.txt"})
	if err != nil {
		t.Fatalf("HeadObject: %v", err)
	}
	if head.ETag != "" {
		t.Errorf("ETag = %q, want empty", head.ETag)
	}
	obj, err := p.GetObject(ctx, storage.GetObjectInput{Key: "e.txt"})
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	obj.Body.Close()
	if obj.ETag != "" {
		t.Errorf("GetObject ETag = %q, want empty", obj.ETag)
	}
}

func TestContractAgainstGDriveProvider(t *testing.T) {
	p, _ := newTestProvider(t)
	storage.CheckProviderContract(t, p)
}

func TestEnsureRootFolder(t *testing.T) {
	client := newFakeDriveClient("root")
	ctx := context.Background()

	id, err := EnsureRootFolder(ctx, client, "my-bucket")
	if err != nil {
		t.Fatalf("EnsureRootFolder: %v", err)
	}
	if id == "" || id == "root" {
		t.Errorf("expected a created folder id, got %q", id)
	}

	again, err := EnsureRootFolder(ctx, client, "my-bucket")
	if err != nil {
		t.Fatalf("EnsureRootFolder again: %v", err)
	}
	if again != id {
		t.Errorf("second call id = %q, want %q (idempotent)", again, id)
	}
}

func TestCapabilities(t *testing.T) {
	p, _ := newTestProvider(t)
	caps := p.Capabilities()
	if !caps.Streaming || !caps.ResumableUpload || !caps.CopyObject || !caps.Metadata {
		t.Errorf("unexpected capabilities: %+v", caps)
	}
	if caps.RangeRequests || caps.Versioning || caps.PresignedURLs {
		t.Errorf("capabilities must not advertise unsupported features: %+v", caps)
	}
}

func TestPutObjectMapsToDriveHierarchy(t *testing.T) {
	p, client := newTestProvider(t)
	ctx := context.Background()

	key := "backups/2026/database.sql.gz"
	if _, err := p.PutObject(ctx, storage.PutObjectInput{
		Key: key, Body: strings.NewReader("sql-data"), ContentType: "application/gzip",
	}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if !client.hasFile(key) {
		t.Errorf("key %q not stored in hierarchy", key)
	}
	if got := client.content(key); got != "sql-data" {
		t.Errorf("content = %q", got)
	}

	obj, err := p.GetObject(ctx, storage.GetObjectInput{Key: key})
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	defer obj.Body.Close()
	if obj.ContentType != "application/gzip" {
		t.Errorf("ContentType = %q", obj.ContentType)
	}
}

func TestPutObjectContentTypeDetection(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()
	if _, err := p.PutObject(ctx, storage.PutObjectInput{
		Key: "report.pdf", Body: strings.NewReader("x"),
	}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	head, err := p.HeadObject(ctx, storage.HeadObjectInput{Key: "report.pdf"})
	if err != nil {
		t.Fatalf("HeadObject: %v", err)
	}
	if head.ContentType != "application/pdf" {
		t.Errorf("ContentType = %q, want application/pdf", head.ContentType)
	}
}

func TestPutObjectMetadataRoundTrip(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()
	meta := map[string]string{"env": "prod", "owner": "infra"}
	if _, err := p.PutObject(ctx, storage.PutObjectInput{
		Key: "meta.txt", Body: strings.NewReader("m"), Metadata: meta,
	}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	head, err := p.HeadObject(ctx, storage.HeadObjectInput{Key: "meta.txt"})
	if err != nil {
		t.Fatalf("HeadObject: %v", err)
	}
	if head.Metadata["env"] != "prod" || head.Metadata["owner"] != "infra" {
		t.Errorf("Metadata = %v", head.Metadata)
	}
}

func TestDeleteObjectIdempotent(t *testing.T) {
	p, client := newTestProvider(t)
	ctx := context.Background()
	if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: "x.txt", Body: strings.NewReader("x")}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if err := p.DeleteObject(ctx, storage.DeleteObjectInput{Key: "x.txt"}); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}
	if err := p.DeleteObject(ctx, storage.DeleteObjectInput{Key: "x.txt"}); err != nil {
		t.Fatalf("DeleteObject twice: %v", err)
	}
	if _, err := p.HeadObject(ctx, storage.HeadObjectInput{Key: "x.txt"}); !errors.Is(err, storage.ErrObjectNotFound) {
		t.Errorf("HeadObject after delete = %v", err)
	}
	_ = client
}

func TestGetObjectStreaming(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()
	big := strings.Repeat("0123456789abcdef", 4096)
	if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: "big.bin", Body: strings.NewReader(big)}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	obj, err := p.GetObject(ctx, storage.GetObjectInput{Key: "big.bin"})
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	defer obj.Body.Close()
	n, err := io.Copy(io.Discard, obj.Body)
	if err != nil {
		t.Fatalf("streaming read: %v", err)
	}
	if n != int64(len(big)) {
		t.Errorf("streamed %d bytes, want %d", n, len(big))
	}
}

func TestListObjectsExcludesFolders(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()
	for _, k := range []string{"a/b.txt", "a/c/d.txt", "top.txt"} {
		if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: k, Body: strings.NewReader("x")}); err != nil {
			t.Fatalf("PutObject %q: %v", k, err)
		}
	}
	out, err := p.ListObjects(ctx, storage.ListObjectsInput{Prefix: ""})
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	keys := map[string]bool{}
	for _, o := range out.Objects {
		keys[o.Key] = true
	}
	for _, want := range []string{"a/b.txt", "a/c/d.txt", "top.txt"} {
		if !keys[want] {
			t.Errorf("missing %q in %v", want, keys)
		}
	}
	for _, bad := range []string{"a", "a/c"} {
		if keys[bad] {
			t.Errorf("folder %q leaked as object", bad)
		}
	}
}

func TestListObjectsPrefixFilter(t *testing.T) {
	p, _ := newTestProvider(t)
	ctx := context.Background()
	for _, k := range []string{"logs/2026/01.txt", "logs/2025/12.txt", "other.txt"} {
		if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: k, Body: strings.NewReader("x")}); err != nil {
			t.Fatalf("PutObject: %v", err)
		}
	}
	out, err := p.ListObjects(ctx, storage.ListObjectsInput{Prefix: "logs/"})
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if len(out.Objects) != 2 {
		t.Fatalf("got %d objects, want 2: %+v", len(out.Objects), out.Objects)
	}
}

func TestListObjectsPaginationAcrossFolders(t *testing.T) {
	p, client := newTestProvider(t)
	client.listPageSize = 2
	ctx := context.Background()
	for _, k := range []string{"d1/a.txt", "d1/b.txt", "d1/c.txt", "d2/x.txt", "d2/y.txt"} {
		if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: k, Body: strings.NewReader("x")}); err != nil {
			t.Fatalf("PutObject %q: %v", k, err)
		}
	}
	seen := map[string]bool{}
	cursor := ""
	rounds := 0
	for {
		out, err := p.ListObjects(ctx, storage.ListObjectsInput{Prefix: "", Limit: 1, Cursor: cursor})
		if err != nil {
			t.Fatalf("ListObjects: %v", err)
		}
		rounds++
		for _, o := range out.Objects {
			seen[o.Key] = true
		}
		if out.NextCursor == "" {
			break
		}
		if rounds > 20 {
			t.Fatal("pagination did not terminate")
		}
		cursor = out.NextCursor
	}
	for _, want := range []string{"d1/a.txt", "d1/b.txt", "d1/c.txt", "d2/x.txt", "d2/y.txt"} {
		if !seen[want] {
			t.Errorf("pagination missed %q", want)
		}
	}
}

func TestListObjectsMalformedCursor(t *testing.T) {
	p, _ := newTestProvider(t)
	_, err := p.ListObjects(context.Background(), storage.ListObjectsInput{Cursor: "!!!not-base64!!!"})
	if !errors.Is(err, storage.ErrInvalidObjectKey) {
		t.Errorf("err = %v, want ErrInvalidObjectKey", err)
	}
}

func TestCopyObject(t *testing.T) {
	p, client := newTestProvider(t)
	ctx := context.Background()
	if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: "src/db.sql", Body: strings.NewReader("data")}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	out, err := p.CopyObject(ctx, storage.CopyObjectInput{SourceKey: "src/db.sql", DestinationKey: "archives/db.sql"})
	if err != nil {
		t.Fatalf("CopyObject: %v", err)
	}
	if out.SourceKey != "src/db.sql" || out.DestinationKey != "archives/db.sql" {
		t.Errorf("unexpected copy output: %+v", out)
	}
	if !client.hasFile("archives/db.sql") {
		t.Error("copy not present at destination")
	}
	if got := client.content("archives/db.sql"); got != "data" {
		t.Errorf("copy content = %q", got)
	}
}

func TestCopyObjectOverwritesDestination(t *testing.T) {
	p, client := newTestProvider(t)
	ctx := context.Background()
	if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: "src.txt", Body: strings.NewReader("new")}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: "dst.txt", Body: strings.NewReader("old")}); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if _, err := p.CopyObject(ctx, storage.CopyObjectInput{SourceKey: "src.txt", DestinationKey: "dst.txt"}); err != nil {
		t.Fatalf("CopyObject: %v", err)
	}
	if got := client.content("dst.txt"); got != "new" {
		t.Errorf("overwritten content = %q, want new", got)
	}
}

func TestCopyObjectMissingSource(t *testing.T) {
	p, _ := newTestProvider(t)
	_, err := p.CopyObject(context.Background(), storage.CopyObjectInput{
		SourceKey: "nope.txt", DestinationKey: "dst.txt",
	})
	if !errors.Is(err, storage.ErrObjectNotFound) {
		t.Errorf("err = %v, want ErrObjectNotFound", err)
	}
}

func TestErrorMapping(t *testing.T) {
	p, client := newTestProvider(t)
	ctx := context.Background()

	// Drive 404 -> ErrObjectNotFound
	_, err := p.GetObject(ctx, storage.GetObjectInput{Key: "missing.txt"})
	if !errors.Is(err, storage.ErrObjectNotFound) {
		t.Errorf("GetObject missing err = %v, want ErrObjectNotFound", err)
	}

	// Drive 403 -> ErrPermissionDenied
	client.mu.Lock()
	client.files["locked"] = &fakeFile{ID: "locked", Name: "locked", MimeType: "application/octet-stream", ParentID: client.rootID}
	client.mu.Unlock()
	client.deleteFile = func(_ context.Context, fileID string) error {
		return &googleapi.Error{Code: 403, Message: "forbidden"}
	}
	err = p.DeleteObject(ctx, storage.DeleteObjectInput{Key: "locked"})
	if !errors.Is(err, storage.ErrPermissionDenied) {
		t.Errorf("DeleteObject forbidden err = %v, want ErrPermissionDenied", err)
	}

	// Drive 401 -> ErrAuthentication
	client.listFiles = func(_ context.Context, _ ListFilesInput) (*ListFilesOutput, error) {
		return nil, &googleapi.Error{Code: 401, Message: "unauthorized"}
	}
	_, err = p.ListObjects(ctx, storage.ListObjectsInput{})
	if !errors.Is(err, storage.ErrAuthentication) {
		t.Errorf("ListObjects unauthorized err = %v, want ErrAuthentication", err)
	}
}

func TestContextPropagation(t *testing.T) {
	p, client := newTestProvider(t)

	client.createFile = func(ctx context.Context, _ CreateFileInput) (*File, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, &googleapi.Error{Code: 500, Message: "should not happen"}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.PutObject(ctx, storage.PutObjectInput{Key: "ctx.txt", Body: strings.NewReader("x")}); err == nil {
		t.Error("expected error on cancelled context")
	}
}
