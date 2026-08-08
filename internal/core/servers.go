package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"net"
	"strings"
	"time"

	"aether/internal/config"
	"aether/internal/db"
	"aether/internal/domain"
)

func NewServerToken(name string) (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
	cfg := cfgFromEnv()
	sum := sha256.Sum256([]byte(token))
	store, err := storeFrom(cfg)
	if err != nil {
		return "", err
	}
	if err := store.SaveServerToken(hex.EncodeToString(sum[:]), "", 24*time.Hour); err != nil {
		return "", err
	}
	return token, nil
}

func ListServersLocal() ([]domain.Server, error) {
	store, err := storeFrom(cfgFromEnv())
	if err != nil {
		return nil, err
	}
	return store.ListServers()
}

func DeleteServerLocal(id string) error {
	store, err := storeFrom(cfgFromEnv())
	if err != nil {
		return err
	}
	return store.DeleteServer(id)
}

func cfgFromEnv() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{StateDir: config.DefaultStateDir()}
	}
	_ = cfg.EnsureDirs()
	return cfg
}

func storeFrom(cfg *config.Config) (*db.Store, error) {
	sqldb, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}
	return db.NewStore(sqldb), nil
}

func (c *Core) NewServerTokenFor(name string) (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
	sum := sha256.Sum256([]byte(token))
	if err := c.Store.SaveServerToken(hex.EncodeToString(sum[:]), "", 24*time.Hour); err != nil {
		return "", err
	}
	return token, nil
}

func (c *Core) AgentURL() string {
	if c.agentAddr != "" {
		return "https://" + c.agentAddr
	}
	host, port, err := net.SplitHostPort(c.Cfg.AgentAddr)
	if err != nil {
		return "https://127.0.0.1:9443"
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "https://" + host + ":" + port
}
