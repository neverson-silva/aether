package s3

import (
	"context"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"aether/internal/storage"
)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

type Provider struct {
	client *minio.Client
	bucket string
}

func NewProvider(config Config) (*Provider, error) {
	if config.Endpoint == "" || config.AccessKey == "" || config.SecretKey == "" || config.Bucket == "" {
		return nil, storage.ErrInvalidConfig
	}
	endpoint := strings.TrimPrefix(strings.TrimPrefix(config.Endpoint, "https://"), "http://")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
		Region: config.Region,
	})
	if err != nil {
		return nil, err
	}
	return &Provider{client: client, bucket: config.Bucket}, nil
}

func (p *Provider) Capabilities() storage.Capabilities {
	return storage.Capabilities{
		Streaming:       true,
		ResumableUpload: true,
		CopyObject:      true,
		Metadata:        true,
		RangeRequests:   true,
		Versioning:      false,
		PresignedURLs:   false,
	}
}

func (p *Provider) PutObject(ctx context.Context, input storage.PutObjectInput) (*storage.PutObjectOutput, error) {
	opts := minio.PutObjectOptions{ContentType: input.ContentType, UserMetadata: input.Metadata}
	info, err := p.client.PutObject(ctx, p.bucket, input.Key, input.Body, -1, opts)
	if err != nil {
		return nil, err
	}
	return &storage.PutObjectOutput{Key: input.Key, ETag: info.ETag, Size: info.Size}, nil
}

func (p *Provider) GetObject(ctx context.Context, input storage.GetObjectInput) (*storage.GetObjectOutput, error) {
	obj, err := p.client.GetObject(ctx, p.bucket, input.Key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, err
	}
	return &storage.GetObjectOutput{
		Key: input.Key, Body: obj, ContentType: stat.ContentType,
		ContentLength: stat.Size, ETag: stat.ETag, LastModified: stat.LastModified,
		Metadata: stat.UserMetadata,
	}, nil
}

func (p *Provider) HeadObject(ctx context.Context, input storage.HeadObjectInput) (*storage.HeadObjectOutput, error) {
	stat, err := p.client.StatObject(ctx, p.bucket, input.Key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &storage.HeadObjectOutput{
		Key: input.Key, ContentLength: stat.Size, ContentType: stat.ContentType,
		ETag: stat.ETag, LastModified: stat.LastModified, Metadata: stat.UserMetadata,
	}, nil
}

func (p *Provider) DeleteObject(ctx context.Context, input storage.DeleteObjectInput) error {
	return p.client.RemoveObject(ctx, p.bucket, input.Key, minio.RemoveObjectOptions{})
}

func (p *Provider) ListObjects(ctx context.Context, input storage.ListObjectsInput) (*storage.ListObjectsOutput, error) {
	opts := minio.ListObjectsOptions{
		Prefix: input.Prefix, MaxKeys: input.Limit,
		StartAfter: input.Cursor, UseV1: false,
	}
	out := &storage.ListObjectsOutput{Objects: []storage.ObjectInfo{}}
	ch := p.client.ListObjects(ctx, p.bucket, opts)
	for obj := range ch {
		if obj.Err != nil {
			return nil, obj.Err
		}
		out.Objects = append(out.Objects, storage.ObjectInfo{
			Key: obj.Key, Size: obj.Size, ETag: obj.ETag, LastModified: obj.LastModified,
			ContentType: obj.ContentType,
		})
		out.NextCursor = obj.Key
		if input.Limit > 0 && len(out.Objects) >= input.Limit {
			break
		}
	}
	if len(out.Objects) == 0 {
		out.NextCursor = ""
	}
	return out, nil
}

func (p *Provider) CopyObject(ctx context.Context, input storage.CopyObjectInput) (*storage.CopyObjectOutput, error) {
	dst := minio.CopyDestOptions{Bucket: p.bucket, Object: input.DestinationKey}
	src := minio.CopySrcOptions{Bucket: p.bucket, Object: input.SourceKey}
	info, err := p.client.CopyObject(ctx, dst, src)
	if err != nil {
		return nil, err
	}
	return &storage.CopyObjectOutput{
		SourceKey: input.SourceKey, DestinationKey: input.DestinationKey, ETag: info.ETag,
	}, nil
}
