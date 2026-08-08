package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"time"

	"aether/internal/domain"
)

type ChannelConfig struct {
	WebhookURL string `json:"webhook_url,omitempty"`
	BotToken   string `json:"bot_token,omitempty"`
	ChatID     string `json:"chat_id,omitempty"`
	SMTPHost   string `json:"smtp_host,omitempty"`
	EmailFrom  string `json:"email_from,omitempty"`
}

func (c *Core) CreateNotificationChannel(orgID, name, typ string, cfg ChannelConfig) (*domain.NotificationChannel, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	enc, err := c.Secrets.EncryptString(string(raw))
	if err != nil {
		return nil, err
	}
	ch := &domain.NotificationChannel{
		ID:        domain.NewID(),
		OrgID:     orgID,
		Name:      name,
		Type:      typ,
		ConfigEnc: enc,
		Enabled:   true,
		CreatedAt: time.Now().UTC(),
	}
	if err := c.Store.CreateNotificationChannel(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

func (c *Core) NotifyOrg(orgID, title, message string) {
	channels, err := c.Store.ListNotificationChannels(orgID)
	if err != nil {
		return
	}
	for i := range channels {
		ch := &channels[i]
		if !ch.Enabled {
			continue
		}
		go c.sendNotification(ch, title, message)
	}
}

func (c *Core) channelConfig(ch *domain.NotificationChannel) (*ChannelConfig, error) {
	raw, err := c.Secrets.DecryptString(ch.ConfigEnc)
	if err != nil {
		return nil, err
	}
	var cfg ChannelConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Core) sendNotification(ch *domain.NotificationChannel, title, message string) {
	cfg, err := c.channelConfig(ch)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 15 * time.Second}
	switch ch.Type {
	case "slack":
		payload := map[string]string{"text": fmt.Sprintf("*%s*\n%s", title, message)}
		c.postJSON(client, cfg.WebhookURL, payload)
	case "discord":
		payload := map[string]string{"content": fmt.Sprintf("**%s**\n%s", title, message)}
		c.postJSON(client, cfg.WebhookURL, payload)
	case "telegram":
		payload := map[string]any{"chat_id": cfg.ChatID, "text": fmt.Sprintf("%s\n%s", title, message)}
		c.postJSON(client, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken), payload)
	case "email":
		c.sendEmail(cfg, title, message)
	case "webhook":
		payload := map[string]string{"title": title, "message": message, "event": "aether.notification"}
		c.postJSON(client, cfg.WebhookURL, payload)
	}
}

func (c *Core) postJSON(client *http.Client, url string, payload any) {
	if url == "" {
		return
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (c *Core) sendEmail(cfg *ChannelConfig, title, message string) {
	if cfg.SMTPHost == "" || cfg.EmailFrom == "" {
		return
	}
	_ = smtp.SendMail(cfg.SMTPHost, nil, cfg.EmailFrom, []string{cfg.EmailFrom},
		[]byte("Subject: "+title+"\r\n\r\n"+message))
}

var _ = context.Background
