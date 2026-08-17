package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"aether/internal/webhooks/domain"
)

type Webhooks struct {
	Store     domain.Store
	Passwords domain.PasswordCipher
	Client    *http.Client
}

func (w *Webhooks) Create(ctx context.Context, orgID uuid.UUID, name, url, secret string, events []string) (*domain.OutWebhook, error) {
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	if name == "" || url == "" || len(events) == 0 {
		return nil, domain.ErrValidation
	}
	secretEnc, err := w.Passwords.Encrypt(secret)
	if err != nil {
		return nil, err
	}
	return w.Store.CreateOutWebhook(ctx, &domain.OutWebhook{
		OrgID: orgID, Name: name, URL: url, SecretEnc: secretEnc, Events: events, Enabled: true,
	})
}

func (w *Webhooks) List(ctx context.Context, orgID uuid.UUID) ([]domain.OutWebhook, error) {
	return w.Store.ListByOrg(ctx, orgID)
}

func (w *Webhooks) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	return w.Store.DeleteOutWebhook(ctx, id, orgID)
}

func (w *Webhooks) Deliver(ctx context.Context, event string, payload any) error {
	hooks, err := w.Store.ListByEvent(ctx, event)
	if err != nil || len(hooks) == 0 {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := w.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	for i := range hooks {
		hook := &hooks[i]
		if !hook.Enabled {
			continue
		}
		secret, err := w.Passwords.Decrypt(hook.SecretEnc)
		if err != nil {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Aether-Event", event)
		req.Header.Set("X-Aether-Signature", sign(body, secret))
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}
	return nil
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
