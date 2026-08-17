package gdrive

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"aether/internal/storage"
)

type Config struct {
	Client       *http.Client
	RootFolderID string
	BaseURL      string
	UploadBase   string
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
	return &Provider{
		client: newDriveHTTPClient(config.Client, config.BaseURL, config.UploadBase),
		rootID: config.RootFolderID,
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
