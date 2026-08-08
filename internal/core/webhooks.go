package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"aether/internal/domain"
)

const (
	EvDeployStarted = "deployment.started"
	EvDeployReady   = "deployment.ready"
	EvDeployFailed  = "deployment.failed"
	EvBackupStarted = "backup.started"
	EvBackupDone    = "backup.finished"
	EvBackupFailed  = "backup.failed"
)

func (c *Core) CreateOutWebhook(orgID, name, url, secret string, events []string) (domain.OutWebhook, error) {
	w := domain.OutWebhook{
		ID:        "wh-" + domain.NewID(),
		OrgID:     orgID,
		Name:      name,
		URL:       url,
		Events:    events,
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	}
	if secret != "" {
		enc, err := c.Secrets.EncryptString(secret)
		if err != nil {
			return w, err
		}
		w.SecretEnc = enc
	}
	err := c.Store.CreateOutWebhook(&w)
	return w, err
}

func (c *Core) FireWebhookEvent(ctx context.Context, orgID, event string, payload map[string]any) {
	hooks, err := c.Store.AllOutWebhooks(orgID)
	if err != nil {
		return
	}
	for _, h := range hooks {
		if !h.Enabled || !containsString(h.Events, event) {
			continue
		}
		hook := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[webhook] panic %s: %v", hook.Name, r)
				}
			}()
			c.deliverWebhook(ctx, &hook, event, payload)
		}()
	}
}

func (c *Core) deliverWebhook(ctx context.Context, h *domain.OutWebhook, event string, payload map[string]any) {
	body, err := json.Marshal(map[string]any{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"payload":   payload,
	})
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", h.URL, strings.NewReader(string(body)))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "aether-platform/1.0")
	req.Header.Set("X-Aether-Event", event)
	if h.SecretEnc != "" {
		if secret, serr := c.Secrets.DecryptString(h.SecretEnc); serr == nil && secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(body)
			req.Header.Set("X-Aether-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		}
	}
	if resp, err := client.Do(req); err == nil {
		resp.Body.Close()
	}
}

func (c *Core) FireDeployStarted(app *domain.App, dep *domain.Deployment) {
	c.FireWebhookEvent(context.Background(), app.OrgID, EvDeployStarted, map[string]any{
		"app":     app.Name,
		"app_id":  app.ID,
		"project": app.ProjectID,
		"build":   dep.Number,
	})
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (c *Core) RegistryPushAfterBuild(imageRef string) {
	cfg, err := c.Store.GetRegistrySettings()
	if err != nil || !cfg.Enabled || cfg.Status != "running" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := c.RegistryPush(ctx, imageRef); err != nil {
			log.Printf("[registry] push %s: %v", imageRef, err)
		}
	}()
}
