package core

import (
	"errors"
	"time"

	"aether/internal/domain"
	"aether/internal/security"
)

var ErrAlreadyRegistered = errors.New("platform already registered")

func (c *Core) Registered() (bool, error) {
	return c.Store.HasUsers()
}

func (c *Core) Register(email, name, password string) (*domain.User, string, error) {
	has, err := c.Store.HasUsers()
	if err != nil {
		return nil, "", err
	}
	if has {
		return nil, "", ErrAlreadyRegistered
	}
	user, org, err := c.CreateUserAndOrg(email, name, password)
	if err != nil {
		return nil, "", err
	}
	token, err := c.Tokens.Sign(security.Claims{
		Subject: user.ID,
		OrgID:   org.ID,
		Role:    string(domain.RoleOwner),
	}, 24*time.Hour)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (c *Core) AuthStatus() map[string]any {
	registered, err := c.Store.HasUsers()
	if err != nil {
		registered = false
	}
	providers, err := c.Store.ListOIDCProviders("")
	if err != nil {
		providers = nil
	}
	return map[string]any{
		"registered": registered,
		"sso":        len(providers) > 0,
	}
}
