package gdrive

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const folderMIME = "application/vnd.google-apps.folder"

const fileFields = "id,name,mimeType,size,modifiedTime,appProperties,parents,trashed"

type File struct {
	ID            string
	Name          string
	MimeType      string
	Size          int64
	ModifiedTime  time.Time
	AppProperties map[string]string
	Parents       []string
	Trashed       bool
}

type CreateFileInput struct {
	Name        string
	MimeType    string
	ContentType string
	ParentID    string
	Metadata    map[string]string
	Body        io.Reader
}

type UpdateFileInput struct {
	MimeType    string
	ContentType string
	Metadata    map[string]string
	Body        io.Reader
}

type ListFilesInput struct {
	ParentID  string
	Name      string
	MimeType  string
	PageSize  int64
	PageToken string
}

type ListFilesOutput struct {
	Files         []*File
	NextPageToken string
}

type CopyFileInput struct {
	Name     string
	ParentID string
}

// DriveClient isolates every Google Drive integration detail. The provider
// only depends on this interface; tests inject fakes.
type DriveClient interface {
	CreateFile(ctx context.Context, input CreateFileInput) (*File, error)
	GetFile(ctx context.Context, fileID string) (*File, error)
	UpdateFile(ctx context.Context, fileID string, input UpdateFileInput) (*File, error)
	DeleteFile(ctx context.Context, fileID string) error
	ListFiles(ctx context.Context, input ListFilesInput) (*ListFilesOutput, error)
	CopyFile(ctx context.Context, fileID string, input CopyFileInput) (*File, error)
	DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error)
}

type staticTokenTransport struct {
	next  http.RoundTripper
	token string
}

func (t *staticTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+t.token)
	return t.next.RoundTrip(req2)
}

// NewTokenClient builds a Drive client authenticated with a static bearer token.
func NewTokenClient(accessToken string) (DriveClient, error) {
	hc := &http.Client{Transport: &staticTokenTransport{next: http.DefaultTransport, token: accessToken}}
	return newDriveServiceClient(hc, "")
}

// NewClientFromHTTP builds a Drive client from an HTTP client.
func NewClientFromHTTP(hc *http.Client) (DriveClient, error) {
	return newDriveServiceClient(hc, "")
}

type driveServiceClient struct {
	svc *drive.Service
}

func newDriveServiceClient(hc *http.Client, endpoint string) (*driveServiceClient, error) {
	var opts []option.ClientOption
	opts = append(opts, option.WithHTTPClient(hc))
	if endpoint != "" {
		opts = append(opts, option.WithEndpoint(endpoint))
	}
	svc, err := drive.NewService(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	return &driveServiceClient{svc: svc}, nil
}

func fileFromDrive(f *drive.File) *File {
	if f == nil {
		return nil
	}
	out := &File{
		ID:            f.Id,
		Name:          f.Name,
		MimeType:      f.MimeType,
		Size:          f.Size,
		AppProperties: f.AppProperties,
		Parents:       f.Parents,
		Trashed:       f.Trashed,
	}
	if f.ModifiedTime != "" {
		if t, err := time.Parse(time.RFC3339, f.ModifiedTime); err == nil {
			out.ModifiedTime = t
		}
	}
	return out
}

func (c *driveServiceClient) CreateFile(ctx context.Context, input CreateFileInput) (*File, error) {
	meta := &drive.File{
		Name:          input.Name,
		MimeType:      input.MimeType,
		AppProperties: input.Metadata,
	}
	if input.ParentID != "" {
		meta.Parents = []string{input.ParentID}
	}
	call := c.svc.Files.Create(meta).Fields(fileFields).Context(ctx)
	if input.MimeType != folderMIME {
		body := input.Body
		if body == nil {
			body = strings.NewReader("")
		}
		call = call.Media(body)
	}
	res, err := call.Do()
	if err != nil {
		return nil, err
	}
	return fileFromDrive(res), nil
}

func (c *driveServiceClient) GetFile(ctx context.Context, fileID string) (*File, error) {
	res, err := c.svc.Files.Get(fileID).Fields(fileFields).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return fileFromDrive(res), nil
}

func (c *driveServiceClient) UpdateFile(ctx context.Context, fileID string, input UpdateFileInput) (*File, error) {
	meta := &drive.File{
		AppProperties: input.Metadata,
	}
	if input.MimeType != "" {
		meta.MimeType = input.MimeType
	}
	call := c.svc.Files.Update(fileID, meta).Fields(fileFields).Context(ctx)
	if input.Body != nil {
		call = call.Media(input.Body)
	}
	res, err := call.Do()
	if err != nil {
		return nil, err
	}
	return fileFromDrive(res), nil
}

func (c *driveServiceClient) DeleteFile(ctx context.Context, fileID string) error {
	return c.svc.Files.Delete(fileID).Context(ctx).Do()
}

func (c *driveServiceClient) ListFiles(ctx context.Context, input ListFilesInput) (*ListFilesOutput, error) {
	q := "'" + escapeQ(input.ParentID) + "' in parents and trashed = false"
	if input.Name != "" {
		q += " and name = '" + escapeQ(input.Name) + "'"
	}
	if input.MimeType != "" {
		q += " and mimeType = '" + input.MimeType + "'"
	}
	pageSize := input.PageSize
	if pageSize <= 0 || pageSize > 1000 {
		pageSize = 1000
	}
	call := c.svc.Files.List().
		Q(q).
		PageSize(pageSize).
		Fields("nextPageToken,files(" + fileFields + ")").
		Context(ctx)
	if input.PageToken != "" {
		call = call.PageToken(input.PageToken)
	}
	res, err := call.Do()
	if err != nil {
		return nil, err
	}
	out := &ListFilesOutput{NextPageToken: res.NextPageToken}
	for _, f := range res.Files {
		out.Files = append(out.Files, fileFromDrive(f))
	}
	return out, nil
}

func (c *driveServiceClient) CopyFile(ctx context.Context, fileID string, input CopyFileInput) (*File, error) {
	body := &drive.File{Name: input.Name}
	if input.ParentID != "" {
		body.Parents = []string{input.ParentID}
	}
	res, err := c.svc.Files.Copy(fileID, body).Fields(fileFields).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return fileFromDrive(res), nil
}

func (c *driveServiceClient) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error) {
	res, err := c.svc.Files.Get(fileID).Context(ctx).Download()
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

func escapeQ(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `'`, `\'`)
}
