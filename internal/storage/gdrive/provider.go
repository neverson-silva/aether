package gdrive

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"aether/internal/storage"
)

type Config struct {
	Client         *http.Client
	RootFolderID   string
	RootFolderName string
	BaseURL        string
}

type Provider struct {
	client DriveClient
	rootID string
	caps   storage.Capabilities
}

func NewProvider(config Config) (*Provider, error) {
	if config.Client == nil {
		return nil, errors.New("gdrive: client is required")
	}
	if strings.TrimSpace(config.RootFolderID) == "" {
		return nil, errors.New("gdrive: root folder id is required")
	}
	client, err := newDriveServiceClient(config.Client, config.BaseURL)
	if err != nil {
		return nil, err
	}
	rootID := config.RootFolderID
	if name := strings.TrimSpace(config.RootFolderName); name != "" && name != "root" {
		id, err := resolveRootFolder(context.Background(), client, name)
		if err != nil {
			return nil, err
		}
		rootID = id
	}
	return &Provider{
		client: client,
		rootID: rootID,
		caps: storage.Capabilities{
			Streaming:       true,
			ResumableUpload: true,
			CopyObject:      true,
			Metadata:        true,
			RangeRequests:   false,
			Versioning:      false,
			PresignedURLs:   false,
		},
	}, nil
}

func (p *Provider) Capabilities() storage.Capabilities {
	return p.caps
}

func providerWithClient(client DriveClient, rootID string) *Provider {
	return &Provider{
		client: client,
		rootID: rootID,
		caps: storage.Capabilities{
			Streaming:       true,
			ResumableUpload: true,
			CopyObject:      true,
			Metadata:        true,
			RangeRequests:   false,
			Versioning:      false,
			PresignedURLs:   false,
		},
	}
}

func (p *Provider) normalizeKey(key string) (string, error) {
	nk, err := storage.NormalizeKey(key)
	if err != nil {
		return "", fmt.Errorf("%w: %q", storage.ErrInvalidObjectKey, key)
	}
	return nk, nil
}

// EnsureRootFolder resolves or creates a top-level folder with the given name.
func EnsureRootFolder(ctx context.Context, client DriveClient, name string) (string, error) {
	return resolveRootFolder(ctx, client, name)
}

func resolveRootFolder(ctx context.Context, client DriveClient, name string) (string, error) {
	out, err := client.ListFiles(ctx, ListFilesInput{ParentID: "root", Name: name, MimeType: folderMIME, PageSize: 100})
	if err != nil {
		return "", err
	}
	for _, f := range out.Files {
		if f.Name == name && f.MimeType == folderMIME {
			return f.ID, nil
		}
	}
	f, err := client.CreateFile(ctx, CreateFileInput{Name: name, MimeType: folderMIME, ParentID: "root"})
	if err != nil {
		return "", err
	}
	return f.ID, nil
}
