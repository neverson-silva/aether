package application

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"aether/internal/modules/webhooks/domain"
	"aether/internal/modules/webhooks/infra"
)

type env struct {
	ctx   context.Context
	svc   *Webhooks
	orgID uuid.UUID
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testPool(t)
	store := infra.NewStore(pool)
	cipher, err := infra.NewPasswordCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	e := &env{ctx: context.Background(), svc: &Webhooks{Store: store, Passwords: cipher}, orgID: uuid.New()}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := pool.Exec(e.ctx, `INSERT INTO orgs (id, name, slug, owner_user_id) VALUES ($1, $2, $3, NULL)`, e.orgID, "Whk Org", "whk-org"); err != nil {
		t.Fatalf("criar org: %v", err)
	}
	return e
}

func TestWebhookLifecycle(t *testing.T) {
	e := newEnv(t)
	hook, err := e.svc.Create(e.ctx, e.orgID, "deploys", "https://hooks.example.com/deploy", "s3cr3t", []string{"deploy.ready"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if hook.SecretEnc == "s3cr3t" || hook.SecretEnc == "" {
		t.Fatalf("secret deveria estar encriptada")
	}
	if !hook.Enabled {
		t.Fatalf("hook deveria estar enabled")
	}
	if _, err := e.svc.Create(e.ctx, e.orgID, "", "x", "", nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("name vazio deveria falhar: %v", err)
	}
	list, err := e.svc.List(e.ctx, e.orgID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if err := e.svc.Delete(e.ctx, hook.ID, e.orgID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestWebhookDeliver(t *testing.T) {
	e := newEnv(t)
	var gotSig, gotEvent string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-Aether-Signature")
		gotEvent = r.Header.Get("X-Aether-Event")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, _ = e.svc.Create(e.ctx, e.orgID, "deploys", srv.URL, "s3cr3t", []string{"deploy.ready"})
	e.svc.Client = srv.Client()

	if err := e.svc.Deliver(e.ctx, "deploy.ready", map[string]any{"app": "web"}); err != nil {
		t.Fatalf("deliver: %v", err)
	}
	if !strings.HasPrefix(gotSig, "sha256=") {
		t.Fatalf("assinatura ausente: %q", gotSig)
	}
	if gotEvent != "deploy.ready" {
		t.Fatalf("event header inesperado: %q", gotEvent)
	}
	if !strings.Contains(string(gotBody), `"app":"web"`) {
		t.Fatalf("payload inesperado: %q", string(gotBody))
	}
}
