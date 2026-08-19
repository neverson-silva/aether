package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"aether/internal/settings/domain"
	"aether/internal/storage"
	"aether/internal/storage/s3"
	"aether/internal/storage/gdrive"
)

var ErrGoogleNotConfigured = errors.New("google oauth is not configured")

// Settings orchestrates destinations and OAuth flows.
type Settings struct {
	Store     domain.Store
	Passwords domain.PasswordCipher
	OIDC      domain.OIDCDiscoverer

	GoogleRedirectURI  string
	PublicURL          string
	HTTPClient         *http.Client

	oauthStates *oauthStateStore
}

func (s *Settings) GetBranding(ctx context.Context, orgID uuid.UUID) (*domain.Branding, error) {
	branding, err := s.Store.GetBranding(ctx, orgID)
	if errors.Is(err, domain.ErrNotFound) {
		return &domain.Branding{OrgID: orgID}, nil
	}
	return branding, err
}

func (s *Settings) SaveBranding(ctx context.Context, orgID uuid.UUID, branding *domain.Branding) (*domain.Branding, error) {
	branding.OrgID = orgID
	return s.Store.SaveBranding(ctx, branding)
}

func (s *Settings) CreateOIDC(ctx context.Context, orgID uuid.UUID, name, issuer, clientID, clientSecret, scopes string) (*domain.OIDCProvider, error) {
	name = strings.TrimSpace(name)
	if name == "" || issuer == "" || clientID == "" {
		return nil, domain.ErrValidation
	}
	if scopes == "" {
		scopes = "openid email profile"
	}
	secretEnc, err := s.Passwords.Encrypt(clientSecret)
	if err != nil {
		return nil, err
	}
	return s.Store.CreateOIDC(ctx, &domain.OIDCProvider{
		OrgID: orgID, Name: name, Issuer: issuer, ClientID: clientID,
		ClientSecretEnc: secretEnc, Scopes: scopes, Enabled: true,
	})
}

func (s *Settings) ListOIDC(ctx context.Context, orgID uuid.UUID) ([]domain.OIDCProvider, error) {
	return s.Store.ListOIDCByOrg(ctx, orgID)
}

func (s *Settings) DeleteOIDC(ctx context.Context, id, orgID uuid.UUID) error {
	return s.Store.DeleteOIDC(ctx, id, orgID)
}

func (s *Settings) PublicOIDC(ctx context.Context) ([]domain.OIDCProvider, error) {
	return s.Store.ListEnabledOIDC(ctx)
}

func (s *Settings) OIDCAuthURL(ctx context.Context, id uuid.UUID) (string, error) {
	provider, err := s.Store.GetOIDC(ctx, id)
	if err != nil {
		return "", err
	}
	if !provider.Enabled {
		return "", domain.ErrForbidden
	}
	return s.OIDC.AuthURL(ctx, provider.Issuer, provider.ClientID, provider.Scopes, provider.ID.String())
}

func (s *Settings) OIDCCallback(ctx context.Context, id uuid.UUID, code string) (*domain.OIDCUser, error) {
	provider, err := s.Store.GetOIDC(ctx, id)
	if err != nil {
		return nil, err
	}
	if !provider.Enabled {
		return nil, domain.ErrForbidden
	}
	secret, err := s.Passwords.Decrypt(provider.ClientSecretEnc)
	if err != nil {
		return nil, err
	}
	return s.OIDC.Exchange(ctx, provider.Issuer, provider.ClientID, secret, provider.ID.String(), code)
}

type oauthState struct {
	DestID  uuid.UUID
	OrgID   uuid.UUID
	Expires time.Time
}

type oauthStateStore struct {
	mu     sync.Mutex
	states map[string]oauthState
}

func newOAuthStateStore() *oauthStateStore {
	return &oauthStateStore{states: map[string]oauthState{}}
}

func (s *oauthStateStore) create(destID, orgID uuid.UUID) (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	state := hex.EncodeToString(raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[state] = oauthState{DestID: destID, OrgID: orgID, Expires: time.Now().Add(10 * time.Minute)}
	return state, nil
}

func (s *oauthStateStore) consume(state string) (oauthState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.states[state]
	if !ok {
		return oauthState{}, false
	}
	delete(s.states, state)
	if time.Now().After(entry.Expires) {
		return oauthState{}, false
	}
	return entry, true
}

// Destination types that require a user-supplied endpoint.
func requiresEndpoint(typ domain.DestinationType) bool {
	switch typ {
	case domain.TypeMinIO, domain.TypeCustomS3:
		return true
	default:
		return false
	}
}

// ResolveEndpoint derives the endpoint for known providers; user endpoints
// are the source of truth for MinIO and custom S3 destinations.
func ResolveEndpoint(typ domain.DestinationType, region, accountID, userEndpoint string) string {
	switch typ {
	case domain.TypeAWS:
		if region == "" {
			region = "us-east-1"
		}
		return "https://s3." + region + ".amazonaws.com"
	case domain.TypeCloudflareR2:
		if accountID == "" {
			return ""
		}
		return "https://" + accountID + ".r2.cloudflarestorage.com"
	case domain.TypeMinIO, domain.TypeCustomS3:
		return strings.TrimSpace(userEndpoint)
	default:
		return ""
	}
}

func validType(typ domain.DestinationType) bool {
	switch typ {
	case domain.TypeAWS, domain.TypeCloudflareR2, domain.TypeMinIO, domain.TypeCustomS3, domain.TypeGoogleDrive:
		return true
	default:
		return false
	}
}

func validateDestination(dest *domain.S3Destination) error {
	if strings.TrimSpace(dest.Name) == "" {
		return domain.ErrValidation
	}
	switch dest.Type {
	case domain.TypeAWS:
		if dest.Bucket == "" || dest.Region == "" || dest.AccessKeyEnc == "" || dest.SecretKeyEnc == "" {
			return domain.ErrValidation
		}
	case domain.TypeCloudflareR2:
		if dest.Bucket == "" || dest.AccountID == "" || dest.AccessKeyEnc == "" || dest.SecretKeyEnc == "" {
			return domain.ErrValidation
		}
	case domain.TypeMinIO, domain.TypeCustomS3:
		if dest.Endpoint == "" || dest.Bucket == "" || dest.AccessKeyEnc == "" || dest.SecretKeyEnc == "" {
			return domain.ErrValidation
		}
	case domain.TypeGoogleDrive:
	default:
		return domain.ErrValidation
	}
	return nil
}

// CreateS3 creates a destination with an explicit provider type. Endpoints
// for known providers are resolved here; credentials are encrypted.
func (s *Settings) CreateS3(ctx context.Context, orgID uuid.UUID, dest *domain.S3Destination) (*domain.S3Destination, error) {
	dest.OrgID = orgID
	dest.Type = domain.DestinationType(strings.TrimSpace(string(dest.Type)))
	if !validType(dest.Type) {
		return nil, domain.ErrValidation
	}
	if dest.UsesCredentials() {
		if strings.TrimSpace(dest.Endpoint) == "" && !requiresEndpoint(dest.Type) {
			dest.Endpoint = ResolveEndpoint(dest.Type, dest.Region, dest.AccountID, dest.Endpoint)
		}
		if dest.Endpoint == "" {
			return nil, domain.ErrValidation
		}
		accessEnc, err := s.Passwords.Encrypt(dest.AccessKeyEnc)
		if err != nil {
			return nil, err
		}
		secretEnc, err := s.Passwords.Encrypt(dest.SecretKeyEnc)
		if err != nil {
			return nil, err
		}
		dest.AccessKeyEnc = accessEnc
		dest.SecretKeyEnc = secretEnc
		if dest.Region == "" {
			dest.Region = "us-east-1"
		}
	} else {
		dest.Endpoint = ""
		dest.AccessKeyEnc = ""
		dest.SecretKeyEnc = ""
		dest.OAuthStatus = domain.OAuthNone
		if dest.GoogleClientSecretEnc != "" {
			enc, err := s.Passwords.Encrypt(dest.GoogleClientSecretEnc)
			if err != nil {
				return nil, err
			}
			dest.GoogleClientSecretEnc = enc
		}
	}
	if err := validateDestination(dest); err != nil {
		return nil, err
	}
	return s.Store.CreateS3(ctx, dest)
}

// UpdateS3 edits a destination. OAuth credentials are preserved for Google
// Drive destinations; S3 credentials are re-encrypted on change.
func (s *Settings) UpdateS3(ctx context.Context, orgID uuid.UUID, dest *domain.S3Destination) (*domain.S3Destination, error) {
	existing, err := s.Store.GetS3(ctx, dest.ID, orgID)
	if err != nil {
		return nil, err
	}
	if dest.Type == "" {
		dest.Type = existing.Type
	}
	if existing.IsOAuth() && dest.Type == domain.TypeGoogleDrive {
		dest.AccessTokenEnc = existing.AccessTokenEnc
		dest.RefreshTokenEnc = existing.RefreshTokenEnc
		dest.OAuthStatus = existing.OAuthStatus
		dest.OAuthEmail = existing.OAuthEmail
		if dest.GoogleClientSecretEnc == "" {
			dest.GoogleClientSecretEnc = existing.GoogleClientSecretEnc
		} else if !strings.HasPrefix(dest.GoogleClientSecretEnc, "enc:v1:") {
			enc, err := s.Passwords.Encrypt(dest.GoogleClientSecretEnc)
			if err != nil {
				return nil, err
			}
			dest.GoogleClientSecretEnc = enc
		}
	} else {
		dest.AccessTokenEnc = ""
		dest.RefreshTokenEnc = ""
		dest.OAuthStatus = domain.OAuthNone
		dest.OAuthEmail = ""
	}
	if !dest.IsOAuth() && dest.Type != existing.Type {
		dest.Endpoint = ResolveEndpoint(dest.Type, dest.Region, dest.AccountID, dest.Endpoint)
	}
	if dest.UsesCredentials() {
		if dest.AccessKeyEnc == "" && dest.SecretKeyEnc == "" {
			dest.AccessKeyEnc = existing.AccessKeyEnc
			dest.SecretKeyEnc = existing.SecretKeyEnc
		} else if !strings.HasPrefix(dest.AccessKeyEnc, "enc:v1:") {
			accessEnc, err := s.Passwords.Encrypt(dest.AccessKeyEnc)
			if err != nil {
				return nil, err
			}
			secretEnc, err := s.Passwords.Encrypt(dest.SecretKeyEnc)
			if err != nil {
				return nil, err
			}
			dest.AccessKeyEnc = accessEnc
			dest.SecretKeyEnc = secretEnc
		}
	}
	dest.OrgID = orgID
	if err := validateDestination(dest); err != nil {
		return nil, err
	}
	return s.Store.UpdateS3(ctx, dest)
}

func (s *Settings) ListS3(ctx context.Context, orgID uuid.UUID) ([]domain.S3Destination, error) {
	return s.Store.ListS3ByOrg(ctx, orgID)
}

func (s *Settings) GetS3(ctx context.Context, id, orgID uuid.UUID) (*domain.S3Destination, error) {
	return s.Store.GetS3(ctx, id, orgID)
}

func (s *Settings) DeleteS3(ctx context.Context, id, orgID uuid.UUID) error {
	return s.Store.DeleteS3(ctx, id, orgID)
}

// S3Provider builds the storage provider for a credentials-based destination.
func (s *Settings) S3Provider(dest *domain.S3Destination) (storage.Provider, error) {
	accessKey, err := s.Passwords.Decrypt(dest.AccessKeyEnc)
	if err != nil {
		return nil, err
	}
	secretKey, err := s.Passwords.Decrypt(dest.SecretKeyEnc)
	if err != nil {
		return nil, err
	}
	return s3.NewProvider(s3.Config{
		Endpoint: dest.Endpoint, AccessKey: accessKey, SecretKey: secretKey,
		Bucket: dest.Bucket, Region: dest.Region, UseSSL: strings.HasPrefix(dest.Endpoint, "https://"),
	})
}

// GoogleProvider builds the Google Drive provider with OAuth token handling.
func (s *Settings) GoogleProvider(dest *domain.S3Destination) (storage.Provider, error) {
	if dest.OAuthStatus != domain.OAuthConnected {
		return nil, domain.ErrReauthRequired
	}
	client, err := s.googleOAuthClient(dest)
	if err != nil {
		return nil, err
	}
	return gdrive.NewProvider(gdrive.Config{
		Client: client, RootFolderID: "root", RootFolderName: dest.Bucket,
	})
}

// TestConnection verifies the destination against its provider.
func (s *Settings) TestConnection(ctx context.Context, orgID uuid.UUID, destID uuid.UUID) error {
	dest, err := s.Store.GetS3(ctx, destID, orgID)
	if err != nil {
		return err
	}
	if dest.IsOAuth() {
		_, err := s.GoogleProvider(dest)
		return err
	}
	provider, err := s.S3Provider(dest)
	if err != nil {
		return err
	}
	testKey := fmt.Sprintf(".aether-test-%d", time.Now().UnixNano())
	if _, err := provider.PutObject(ctx, storage.PutObjectInput{Key: testKey, Body: strings.NewReader("ok")}); err != nil {
		return err
	}
	return provider.DeleteObject(ctx, storage.DeleteObjectInput{Key: testKey})
}
