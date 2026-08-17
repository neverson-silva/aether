package gdrive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type fakeDriveClient struct {
	mu           sync.Mutex
	nextID       int
	files        map[string]*fakeFile
	rootID       string
	listPageSize int
	seq          int

	createFile   func(ctx context.Context, input CreateFileInput) (*File, error)
	getFile      func(ctx context.Context, fileID string) (*File, error)
	updateFile   func(ctx context.Context, fileID string, input UpdateFileInput) (*File, error)
	deleteFile   func(ctx context.Context, fileID string) error
	listFiles    func(ctx context.Context, input ListFilesInput) (*ListFilesOutput, error)
	copyFile     func(ctx context.Context, fileID string, input CopyFileInput) (*File, error)
	downloadFile func(ctx context.Context, fileID string) (io.ReadCloser, error)
}

type fakeFile struct {
	ID            string
	Name          string
	MimeType      string
	ParentID      string
	Content       []byte
	AppProperties map[string]string
	ModifiedTime  time.Time
	Deleted       bool
}

func newFakeDriveClient(rootID string) *fakeDriveClient {
	c := &fakeDriveClient{
		files: map[string]*fakeFile{}, rootID: rootID, listPageSize: 1000,
	}
	c.mustAdd(rootID, "", folderMIME, "")
	c.createFile = c.realCreateFile
	c.getFile = c.realGetFile
	c.updateFile = c.realUpdateFile
	c.deleteFile = c.realDeleteFile
	c.listFiles = c.realListFiles
	c.copyFile = c.realCopyFile
	c.downloadFile = c.realDownloadFile
	return c
}

func (c *fakeDriveClient) mustAdd(id, name, mime, parentID string) *fakeFile {
	f := &fakeFile{
		ID: id, Name: name, MimeType: mime, ParentID: parentID,
		ModifiedTime: time.Now(),
	}
	c.files[id] = f
	return f
}

func (c *fakeDriveClient) allocate() string {
	c.seq++
	return "file" + strconv.Itoa(c.seq)
}

func (c *fakeDriveClient) realCreateFile(_ context.Context, input CreateFileInput) (*File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if input.ParentID != "" {
		if _, ok := c.files[input.ParentID]; !ok {
			return nil, &googleError{Code: 404, Message: "parent not found"}
		}
	}
	id := c.allocate()
	f := &fakeFile{
		ID: id, Name: input.Name, MimeType: input.MimeType, ParentID: input.ParentID,
		AppProperties: input.Metadata, ModifiedTime: time.Now(),
	}
	if input.Body != nil {
		data, _ := io.ReadAll(input.Body)
		f.Content = data
	}
	c.files[id] = f
	return fakeToFile(f), nil
}

func (c *fakeDriveClient) realGetFile(_ context.Context, fileID string) (*File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.files[fileID]
	if !ok || f.Deleted {
		return nil, &googleError{Code: 404, Message: "file not found"}
	}
	return fakeToFile(f), nil
}

func (c *fakeDriveClient) realUpdateFile(_ context.Context, fileID string, input UpdateFileInput) (*File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.files[fileID]
	if !ok || f.Deleted {
		return nil, &googleError{Code: 404, Message: "file not found"}
	}
	if input.MimeType != "" {
		f.MimeType = input.MimeType
	}
	if input.Metadata != nil {
		f.AppProperties = input.Metadata
	}
	if input.Body != nil {
		data, _ := io.ReadAll(input.Body)
		f.Content = data
	}
	f.ModifiedTime = time.Now()
	return fakeToFile(f), nil
}

func (c *fakeDriveClient) realDeleteFile(_ context.Context, fileID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.files[fileID]
	if !ok || f.Deleted {
		return &googleError{Code: 404, Message: "file not found"}
	}
	f.Deleted = true
	return nil
}

func (c *fakeDriveClient) realListFiles(_ context.Context, input ListFilesInput) (*ListFilesOutput, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var children []*fakeFile
	for _, f := range c.files {
		if f.Deleted || f.ParentID != input.ParentID {
			continue
		}
		if input.Name != "" && f.Name != input.Name {
			continue
		}
		if input.MimeType != "" && f.MimeType != input.MimeType {
			continue
		}
		children = append(children, f)
	}
	sort.Slice(children, func(i, j int) bool { return children[i].Name < children[j].Name })

	out := &ListFilesOutput{}
	offset := 0
	if input.PageToken != "" {
		n, err := strconv.Atoi(input.PageToken)
		if err != nil {
			return nil, &googleError{Code: 400, Message: "bad token"}
		}
		offset = n
	}
	pageSize := c.listPageSize
	if input.PageSize > 0 && input.PageSize < int64(pageSize) {
		pageSize = int(input.PageSize)
	}
	for i := offset; i < len(children) && len(out.Files) < pageSize; i++ {
		out.Files = append(out.Files, fakeToFile(children[i]))
	}
	if offset+len(out.Files) < len(children) {
		out.NextPageToken = strconv.Itoa(offset + len(out.Files))
	}
	return out, nil
}

func (c *fakeDriveClient) realCopyFile(_ context.Context, fileID string, input CopyFileInput) (*File, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	src, ok := c.files[fileID]
	if !ok || src.Deleted {
		return nil, &googleError{Code: 404, Message: "file not found"}
	}
	id := c.allocate()
	f := &fakeFile{
		ID: id, Name: input.Name, MimeType: src.MimeType, ParentID: input.ParentID,
		Content: append([]byte(nil), src.Content...), AppProperties: src.AppProperties,
		ModifiedTime: time.Now(),
	}
	c.files[id] = f
	return fakeToFile(f), nil
}

func (c *fakeDriveClient) realDownloadFile(_ context.Context, fileID string) (io.ReadCloser, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.files[fileID]
	if !ok || f.Deleted {
		return nil, &googleError{Code: 404, Message: "file not found"}
	}
	return io.NopCloser(bytes.NewReader(f.Content)), nil
}

func (c *fakeDriveClient) CreateFile(ctx context.Context, input CreateFileInput) (*File, error) {
	return c.createFile(ctx, input)
}

func (c *fakeDriveClient) GetFile(ctx context.Context, fileID string) (*File, error) {
	return c.getFile(ctx, fileID)
}

func (c *fakeDriveClient) UpdateFile(ctx context.Context, fileID string, input UpdateFileInput) (*File, error) {
	return c.updateFile(ctx, fileID, input)
}

func (c *fakeDriveClient) DeleteFile(ctx context.Context, fileID string) error {
	return c.deleteFile(ctx, fileID)
}

func (c *fakeDriveClient) ListFiles(ctx context.Context, input ListFilesInput) (*ListFilesOutput, error) {
	return c.listFiles(ctx, input)
}

func (c *fakeDriveClient) CopyFile(ctx context.Context, fileID string, input CopyFileInput) (*File, error) {
	return c.copyFile(ctx, fileID, input)
}

func (c *fakeDriveClient) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error) {
	return c.downloadFile(ctx, fileID)
}

func fakeToFile(f *fakeFile) *File {
	return &File{
		ID: f.ID, Name: f.Name, MimeType: f.MimeType, Size: int64(len(f.Content)),
		ModifiedTime: f.ModifiedTime, AppProperties: f.AppProperties, Parents: []string{f.ParentID},
	}
}

// root returns the ID of the fake root folder for assertions.
func (c *fakeDriveClient) root() string { return c.rootID }

func (c *fakeDriveClient) hasFile(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	parts := strings.Split(key, "/")
	parent := c.rootID
	for i, part := range parts {
		last := i == len(parts)-1
		for _, f := range c.files {
			if f.Deleted || f.ParentID != parent || f.Name != part {
				continue
			}
			if last && f.MimeType == folderMIME {
				continue
			}
			parent = f.ID
			if last {
				return true
			}
			break
		}
	}
	return false
}

func (c *fakeDriveClient) content(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	parts := strings.Split(key, "/")
	parent := c.rootID
	for i, part := range parts {
		last := i == len(parts)-1
		for _, f := range c.files {
			if f.Deleted || f.ParentID != parent || f.Name != part {
				continue
			}
			if last {
				return string(f.Content)
			}
			parent = f.ID
			break
		}
	}
	return ""
}

func (c *fakeDriveClient) debugState() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var sb strings.Builder
	fmt.Fprintf(&sb, "root=%s\n", c.rootID)
	for _, f := range c.files {
		if f.Deleted {
			continue
		}
		fmt.Fprintf(&sb, "  %s name=%q mime=%s parent=%q size=%d\n", f.ID, f.Name, f.MimeType, f.ParentID, len(f.Content))
	}
	return sb.String()
}
