package storage

import "context"

type ObjectStorage interface {
	PutObject(ctx context.Context, input PutObjectInput) (*PutObjectOutput, error)
	GetObject(ctx context.Context, input GetObjectInput) (*GetObjectOutput, error)
	HeadObject(ctx context.Context, input HeadObjectInput) (*HeadObjectOutput, error)
	DeleteObject(ctx context.Context, input DeleteObjectInput) error
	ListObjects(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error)
	CopyObject(ctx context.Context, input CopyObjectInput) (*CopyObjectOutput, error)
}

type Provider interface {
	ObjectStorage
	Capabilities() Capabilities
}

type Storage struct {
	provider Provider
}

func New(provider Provider) *Storage {
	return &Storage{provider: provider}
}

func (s *Storage) PutObject(ctx context.Context, input PutObjectInput) (*PutObjectOutput, error) {
	return s.provider.PutObject(ctx, input)
}

func (s *Storage) GetObject(ctx context.Context, input GetObjectInput) (*GetObjectOutput, error) {
	return s.provider.GetObject(ctx, input)
}

func (s *Storage) HeadObject(ctx context.Context, input HeadObjectInput) (*HeadObjectOutput, error) {
	return s.provider.HeadObject(ctx, input)
}

func (s *Storage) DeleteObject(ctx context.Context, input DeleteObjectInput) error {
	return s.provider.DeleteObject(ctx, input)
}

func (s *Storage) ListObjects(ctx context.Context, input ListObjectsInput) (*ListObjectsOutput, error) {
	return s.provider.ListObjects(ctx, input)
}

func (s *Storage) CopyObject(ctx context.Context, input CopyObjectInput) (*CopyObjectOutput, error) {
	return s.provider.CopyObject(ctx, input)
}

func (s *Storage) Capabilities() Capabilities {
	return s.provider.Capabilities()
}
