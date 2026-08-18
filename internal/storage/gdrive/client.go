package gdrive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	folderMIME    = "application/vnd.google-apps.folder"
	defaultBase   = "https://www.googleapis.com"
	defaultUpload = "https://www.googleapis.com/upload"
	chunkSize     = 256 * 1024
)

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

type driveHTTPClient struct {
	http       *http.Client
	baseURL    string
	uploadBase string
}

func newDriveHTTPClient(hc *http.Client, baseURL, uploadBase string) *driveHTTPClient {
	if baseURL == "" {
		baseURL = defaultBase
	}
	if uploadBase == "" {
		uploadBase = defaultUpload
	}
	return &driveHTTPClient{http: hc, baseURL: baseURL, uploadBase: uploadBase}
}

type googleError struct {
	Code    int
	Message string
}

func (e *googleError) Error() string {
	return fmt.Sprintf("google drive: %d %s", e.Code, e.Message)
}

type driveErrorEnvelope struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *driveHTTPClient) doJSON(ctx context.Context, method, u string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	}
	return c.roundTrip(req, out)
}

func (c *driveHTTPClient) roundTrip(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return readGoogleError(resp)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func readGoogleError(resp *http.Response) error {
	var env driveErrorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err == nil && env.Error.Message != "" {
		return &googleError{Code: env.Error.Code, Message: env.Error.Message}
	}
	return &googleError{Code: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
}

const fileFields = "id,name,mimeType,size,modifiedTime,appProperties,parents,trashed"

func (c *driveHTTPClient) CreateFile(ctx context.Context, input CreateFileInput) (*File, error) {
	meta := map[string]any{"name": input.Name, "mimeType": input.MimeType}
	if input.ParentID != "" {
		meta["parents"] = []string{input.ParentID}
	}
	if len(input.Metadata) > 0 {
		meta["appProperties"] = input.Metadata
	}
	if input.MimeType != folderMIME {
		body := input.Body
		if body == nil {
			body = strings.NewReader("")
		}
		return c.uploadMedia(ctx, http.MethodPost, "/drive/v3/files?uploadType=resumable", meta, body, input.ContentType)
	}
	var raw fileRaw
	u := c.baseURL + "/drive/v3/files?fields=" + url.QueryEscape(fileFields)
	if err := c.doJSON(ctx, http.MethodPost, u, meta, &raw); err != nil {
		return nil, err
	}
	return fileFromRaw(raw), nil
}

func (c *driveHTTPClient) GetFile(ctx context.Context, fileID string) (*File, error) {
	var raw fileRaw
	u := c.baseURL + "/drive/v3/files/" + url.PathEscape(fileID) + "?fields=" + url.QueryEscape(fileFields)
	if err := c.doJSON(ctx, http.MethodGet, u, nil, &raw); err != nil {
		return nil, err
	}
	return fileFromRaw(raw), nil
}

func (c *driveHTTPClient) UpdateFile(ctx context.Context, fileID string, input UpdateFileInput) (*File, error) {
	meta := map[string]any{}
	if input.MimeType != "" {
		meta["mimeType"] = input.MimeType
	}
	if len(input.Metadata) > 0 {
		meta["appProperties"] = input.Metadata
	}
	if input.Body != nil {
		return c.uploadMedia(ctx, http.MethodPatch, "/drive/v3/files/"+url.PathEscape(fileID)+"?uploadType=resumable", meta, input.Body, input.ContentType)
	}
	var raw fileRaw
	u := c.baseURL + "/drive/v3/files/" + url.PathEscape(fileID) + "?fields=" + url.QueryEscape(fileFields)
	if err := c.doJSON(ctx, http.MethodPatch, u, meta, &raw); err != nil {
		return nil, err
	}
	return fileFromRaw(raw), nil
}

func (c *driveHTTPClient) DeleteFile(ctx context.Context, fileID string) error {
	u := c.baseURL + "/drive/v3/files/" + url.PathEscape(fileID)
	return c.doJSON(ctx, http.MethodDelete, u, nil, nil)
}

func (c *driveHTTPClient) ListFiles(ctx context.Context, input ListFilesInput) (*ListFilesOutput, error) {
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
	v := url.Values{}
	v.Set("q", q)
	v.Set("pageSize", strconv.FormatInt(pageSize, 10))
	v.Set("fields", "nextPageToken,"+fileFields)
	if input.PageToken != "" {
		v.Set("pageToken", input.PageToken)
	}
	u := c.baseURL + "/drive/v3/files?" + v.Encode()
	var raw struct {
		Files         []fileRaw `json:"files"`
		NextPageToken string    `json:"nextPageToken"`
	}
	if err := c.doJSON(ctx, http.MethodGet, u, nil, &raw); err != nil {
		return nil, err
	}
	out := &ListFilesOutput{NextPageToken: raw.NextPageToken}
	for _, f := range raw.Files {
		out.Files = append(out.Files, fileFromRaw(f))
	}
	return out, nil
}

func (c *driveHTTPClient) CopyFile(ctx context.Context, fileID string, input CopyFileInput) (*File, error) {
	body := map[string]any{"name": input.Name}
	if input.ParentID != "" {
		body["parents"] = []string{input.ParentID}
	}
	var raw fileRaw
	u := c.baseURL + "/drive/v3/files/" + url.PathEscape(fileID) + "/copy?fields=" + url.QueryEscape(fileFields)
	if err := c.doJSON(ctx, http.MethodPost, u, body, &raw); err != nil {
		return nil, err
	}
	return fileFromRaw(raw), nil
}

func (c *driveHTTPClient) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error) {
	u := c.baseURL + "/drive/v3/files/" + url.PathEscape(fileID) + "?alt=media"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		return nil, readGoogleError(resp)
	}
	return resp.Body, nil
}

func (c *driveHTTPClient) uploadMedia(ctx context.Context, method, endpoint string, meta map[string]any, media io.Reader, contentType string) (*File, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	metadata, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	initURL := c.uploadBase + endpoint
	req, err := http.NewRequestWithContext(ctx, method, initURL, bytes.NewReader(metadata))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Upload-Content-Type", contentType)
	if size, ok := readerSize(media); ok {
		req.Header.Set("X-Upload-Content-Length", strconv.FormatInt(size, 10))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	location := resp.Header.Get("Location")
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &googleError{Code: resp.StatusCode, Message: "could not start upload session"}
	}
	if location == "" {
		return nil, errors.New("google drive: upload session returned no location")
	}
	if size, ok := readerSize(media); ok {
		return c.uploadOneShot(ctx, location, media, size, contentType)
	}
	return c.uploadChunked(ctx, location, media, contentType)
}

func (c *driveHTTPClient) uploadOneShot(ctx context.Context, u string, media io.Reader, size int64, contentType string) (*File, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, media)
	if err != nil {
		return nil, err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", contentType)
	return c.uploadPUT(req)
}

func (c *driveHTTPClient) uploadChunked(ctx context.Context, u string, media io.Reader, contentType string) (*File, error) {
	buf := make([]byte, chunkSize)
	var start int64
	for {
		n, err := io.ReadFull(media, buf)
		if n == 0 {
			if errors.Is(err, io.EOF) {
				return nil, errors.New("google drive: cannot upload empty stream with unknown size")
			}
			return nil, err
		}
		last := errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF)
		end := start + int64(n)
		var rng string
		if last {
			rng = fmt.Sprintf("bytes %d-%d/%d", start, end-1, end)
		} else {
			rng = fmt.Sprintf("bytes %d-%d/*", start, end-1)
		}
		req, rerr := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(buf[:n]))
		if rerr != nil {
			return nil, rerr
		}
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Content-Range", rng)
		resp, derr := c.http.Do(req)
		if derr != nil {
			return nil, derr
		}
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			defer resp.Body.Close()
			return decodeUploadResponse(resp)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusPartialContent {
			return nil, readGoogleError(resp)
		}
		start = end
	}
}

func (c *driveHTTPClient) uploadPUT(req *http.Request) (*File, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, readGoogleError(resp)
	}
	return decodeUploadResponse(resp)
}

func decodeUploadResponse(resp *http.Response) (*File, error) {
	var raw fileRaw
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return fileFromRaw(raw), nil
}

type fileRaw struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	MimeType      string            `json:"mimeType"`
	Size          int64             `json:"size"`
	ModifiedTime  string            `json:"modifiedTime"`
	AppProperties map[string]string `json:"appProperties"`
	Parents       []string          `json:"parents"`
	Trashed       bool              `json:"trashed"`
}

func fileFromRaw(raw fileRaw) *File {
	f := &File{
		ID:            raw.ID,
		Name:          raw.Name,
		MimeType:      raw.MimeType,
		Size:          raw.Size,
		AppProperties: raw.AppProperties,
		Parents:       raw.Parents,
		Trashed:       raw.Trashed,
	}
	if raw.ModifiedTime != "" {
		if t, err := time.Parse(time.RFC3339, raw.ModifiedTime); err == nil {
			f.ModifiedTime = t
		}
	}
	return f
}

func readerSize(r io.Reader) (int64, bool) {
	switch v := r.(type) {
	case interface{ Size() int64 }:
		return v.Size(), true
	case interface{ Len() int }:
		return int64(v.Len()), true
	case *os.File:
		if st, err := v.Stat(); err == nil {
			return st.Size(), true
		}
	}
	return 0, false
}

func escapeQ(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `'`, `\'`)
}
