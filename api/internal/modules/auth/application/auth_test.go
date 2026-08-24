package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/auth/domain"
	"aether/internal/modules/auth/infra"
)

type testEnv struct {
	ctx   context.Context
	svc   *Auth
	store *infra.Store
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	env := &testEnv{
		ctx:   context.Background(),
		store: store,
	}
	env.svc = &Auth{
		Users: store, Orgs: store, Members: store, Keys: store, AuditLog: store,
		Tokens: infra.NewSigner("unit-test-secret-0123456789abcdef0123456789abcdef"),
		Hash:   infra.NewHasher(), TokenTTL: time.Hour,
	}
	t.Cleanup(func() { _ = store.Close() })
	return env
}

func TestRegisterLoginMe(t *testing.T) {
	env := newTestEnv(t)
	user, token, err := env.svc.Register(env.ctx, "alice@example.com", "Alice", "senha-segura-123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "alice@example.com" || user.ID == (uuid.UUID{}) {
		t.Fatalf("user inesperado: %+v", user)
	}
	if token == "" {
		t.Fatalf("token vazio")
	}

	got, token2, err := env.svc.Login(env.ctx, "alice@example.com", "senha-segura-123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if got.ID != user.ID || token2 == "" {
		t.Fatalf("login divergente")
	}

	me, orgs, err := env.svc.Me(env.ctx, user.ID)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if me.Email != "alice@example.com" || len(orgs) != 1 {
		t.Fatalf("me divergente: %+v %+v", me, orgs)
	}
	if orgs[0].Role != domain.RoleOwner {
		t.Fatalf("role do fundador deve ser owner: %s", orgs[0].Role)
	}
}

func TestRegisterInvalid(t *testing.T) {
	env := newTestEnv(t)
	cases := []struct {
		name     string
		email    string
		pass     string
		expected error
	}{
		{"email invalido", "nao-e-email", "senha-segura-123", domain.ErrValidation},
		{"senha curta", "bob@example.com", "123", domain.ErrValidation},
		{"nome vazio", "bob@example.com", "senha-segura-123", domain.ErrValidation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := env.svc.Register(env.ctx, c.email, "", c.pass)
			if err != c.expected {
				t.Fatalf("esperava %v, got %v", c.expected, err)
			}
		})
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	env := newTestEnv(t)
	if _, _, err := env.svc.Register(env.ctx, "dupe@example.com", "Dup", "senha-segura-123"); err != nil {
		t.Fatalf("primeiro register: %v", err)
	}
	if _, _, err := env.svc.Register(env.ctx, "dupe@example.com", "Outro", "senha-segura-456"); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("esperava ErrEmailTaken, got %v", err)
	}
}

func TestRegisterMultipleEmailsCreateUniqueOrganizations(t *testing.T) {
	env := newTestEnv(t)
	for i, email := range []string{"one@example.com", "two@example.com", "three@example.com"} {
		_, token, err := env.svc.Register(env.ctx, email, "User", "senha-segura-123")
		if err != nil {
			t.Fatalf("register %d: %v", i, err)
		}
		if token == "" {
			t.Fatalf("register %d: token vazio", i)
		}
	}
}

func TestRegisterAtomicity(t *testing.T) {
	env := newTestEnv(t)
	if _, _, err := env.svc.Register(env.ctx, "alice@example.com", "Alice", "senha-segura-123"); err != nil {
		t.Fatalf("primeiro register: %v", err)
	}
	if _, _, err := env.svc.Register(env.ctx, "alice2@example.com", "Alice", "senha-segura-456"); err != nil {
		t.Fatalf("segundo register não deveria colidir: %v", err)
	}
	if _, err := env.svc.Users.GetUserByEmail(env.ctx, "alice2@example.com"); err != nil {
		t.Fatalf("usuário do segundo register deveria persistir: %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	env := newTestEnv(t)
	if _, _, err := env.svc.Register(env.ctx, "carol@example.com", "Carol", "senha-segura-123"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, _, err := env.svc.Login(env.ctx, "carol@example.com", "errada"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("esperava ErrInvalidCredentials, got %v", err)
	}
	if _, _, err := env.svc.Login(env.ctx, "inexistente@example.com", "senha-segura-123"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("email inexistente deve ser credencial inválida, got %v", err)
	}
}

func TestAddMemberAndRoles(t *testing.T) {
	env := newTestEnv(t)
	owner, _, err := env.svc.Register(env.ctx, "owner@example.com", "Owner", "senha-segura-123")
	if err != nil {
		t.Fatalf("register owner: %v", err)
	}
	_, orgs, _ := env.svc.Me(env.ctx, owner.ID)
	orgID := orgs[0].ID

	if err := env.svc.AddMember(env.ctx, orgID, owner.ID, "dev@example.com", "Dev", "senha-segura-456", domain.RoleDeveloper); err != nil {
		t.Fatalf("add member: %v", err)
	}

	members, err := env.svc.ListMembers(env.ctx, orgID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("esperava 2 membros, got %d", len(members))
	}

	if err := env.svc.AddMember(env.ctx, orgID, owner.ID, "dev@example.com", "Dev2", "senha-segura-456", domain.RoleMember); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("email duplicado em org deveria falhar, got %v", err)
	}
}

func TestAPIKeyLifecycle(t *testing.T) {
	env := newTestEnv(t)
	user, _, err := env.svc.Register(env.ctx, "keys@example.com", "Keys", "senha-segura-123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_, orgs, _ := env.svc.Me(env.ctx, user.ID)
	orgID := orgs[0].ID

	key, raw, err := env.svc.CreateAPIKey(env.ctx, orgID, user.ID, "ci")
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if !strings.HasPrefix(raw, "aether_") || key.Name != "ci" {
		t.Fatalf("key inesperada: %+v", key)
	}
	keys, err := env.svc.ListAPIKeys(env.ctx, orgID)
	if err != nil || len(keys) != 1 {
		t.Fatalf("list keys: %v %d", err, len(keys))
	}
	if err := env.svc.DeleteAPIKey(env.ctx, orgID, user.ID, key.ID); err != nil {
		t.Fatalf("delete key: %v", err)
	}
	keys, _ = env.svc.ListAPIKeys(env.ctx, orgID)
	if len(keys) != 0 {
		t.Fatalf("key deveria ter sido removida")
	}
}

func TestAuditTrail(t *testing.T) {
	env := newTestEnv(t)
	user, _, err := env.svc.Register(env.ctx, "audit@example.com", "Audit", "senha-segura-123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_, orgs, _ := env.svc.Me(env.ctx, user.ID)
	orgID := orgs[0].ID

	events, err := env.svc.Audit(env.ctx, orgID, 50)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Action == "user.register" {
			found = true
		}
	}
	if !found {
		t.Fatalf("evento user.register ausente: %+v", events)
	}
}
