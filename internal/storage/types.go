package storage

import (
	"io"
	"time"
)

type PutObjectInput struct {
	Key         string
	Body        io.Reader
	ContentType string
	Metadata    map[string]string
}

type PutObjectOutput struct {
	Key  string
	ETag string
	Size int64
}

type GetObjectInput struct {
	Key string
}

type GetObjectOutput struct {
	Key           string
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
	ETag          string
	LastModified  time.Time
	Metadata      map[string]string
}

type HeadObjectInput struct {
	Key string
}

type HeadObjectOutput struct {
	Key           string
	ContentLength int64
	ContentType   string
	ETag          string
	LastModified  time.Time
	Metadata      map[string]string
}

type DeleteObjectInput struct {
	Key string
}

type ListObjectsInput struct {
	Prefix string
	Limit  int
	Cursor string
}

type ListObjectsOutput struct {
	Objects    []ObjectInfo
	NextCursor string
}

type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

type CopyObjectInput struct {
	SourceKey      string
	DestinationKey string
}

type CopyObjectOutput struct {
	SourceKey      string
	DestinationKey string
	ETag           string
}
