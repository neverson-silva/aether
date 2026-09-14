package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authapplication "aether/internal/modules/auth/application"
	authdomain "aether/internal/modules/auth/domain"
	authhttp "aether/internal/modules/auth/http"
)

type routeUserStore struct{}

func (routeUserStore) CreateUser(context.Context, string, string, string, string) (*authdomain.User, error) {
	return nil, authdomain.ErrNotFound
}

func (routeUserStore) GetUserByEmail(context.Context, string) (*authdomain.User, error) {
	return nil, authdomain.ErrNotFound
}

func (routeUserStore) GetUserByID(_ context.Context, id uuid.UUID) (*authdomain.User, error) {
	return &authdomain.User{ID: id}, nil
}

func (routeUserStore) GetUserWithSecret(context.Context, uuid.UUID) (*authdomain.User, error) {
	return nil, authdomain.ErrNotFound
}

func (routeUserStore) HasUsers(context.Context) (bool, error) { return true, nil }
func (routeUserStore) ListByOrg(context.Context, uuid.UUID) ([]authdomain.Member, error) {
	return nil, nil
}
func (routeUserStore) Register(context.Context, string, string, string, string, string, string) (*authdomain.User, *authdomain.Org, error) {
	return nil, nil, authdomain.ErrNotFound
}
func (routeUserStore) AddMemberUser(context.Context, uuid.UUID, string, string, string, authdomain.Role) (*authdomain.User, error) {
	return nil, authdomain.ErrNotFound
}
func (routeUserStore) SetTOTP(context.Context, uuid.UUID, []byte) error        { return nil }
func (routeUserStore) DisableTOTP(context.Context, uuid.UUID) error            { return nil }
func (routeUserStore) UpdatePassword(context.Context, uuid.UUID, string) error { return nil }

type routeMemberStore struct{}

func (routeMemberStore) CreateMember(context.Context, uuid.UUID, uuid.UUID, authdomain.Role) error {
	return nil
}
func (routeMemberStore) GetMember(_ context.Context, orgID, userID uuid.UUID) (*authdomain.Member, error) {
	return &authdomain.Member{OrgID: orgID, UserID: userID, Role: authdomain.RoleOwner}, nil
}
func (routeMemberStore) UpdateRole(context.Context, uuid.UUID, uuid.UUID, authdomain.Role) error {
	return nil
}
func (routeMemberStore) DeleteMember(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type routeTokenSigner struct{}

func (routeTokenSigner) Sign(context.Context, uuid.UUID, uuid.UUID, authdomain.Role, string, time.Duration) (string, error) {
	return "", nil
}

func (routeTokenSigner) SignRefresh(context.Context, uuid.UUID, uuid.UUID, authdomain.Role, string, time.Duration) (string, error) {
	return "", nil
}

func (routeTokenSigner) SignRefreshUntil(context.Context, uuid.UUID, uuid.UUID, authdomain.Role, string, time.Time) (string, error) {
	return "", nil
}

func (routeTokenSigner) Verify(context.Context, string) (*authdomain.AuthToken, error) {
	return &authdomain.AuthToken{Subject: uuid.New(), OrgID: uuid.New(), Role: authdomain.RoleOwner, Kind: "access"}, nil
}

func TestGlobalRoutesRejectTenantTokens(t *testing.T) {
	router := &Router{
		engine: gin.New(),
		auth: authhttp.New(&authapplication.Auth{
			Users: routeUserStore{}, Members: routeMemberStore{}, Tokens: routeTokenSigner{},
		}, true),
	}
	router.routes()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/servers"},
		{http.MethodPost, "/api/v1/servers/token"},
		{http.MethodDelete, "/api/v1/servers/server-id"},
		{http.MethodGet, "/api/v1/registry"},
		{http.MethodPost, "/api/v1/registry"},
		{http.MethodGet, "/api/v1/registry/images"},
		{http.MethodDelete, "/api/v1/registry/images/repository/tag"},
	}
	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			req.Header.Set("Authorization", "Bearer tenant-token")
			response := httptest.NewRecorder()
			router.engine.ServeHTTP(response, req)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
			}
		})
	}
}
