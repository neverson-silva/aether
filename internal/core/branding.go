package core

import (
	"time"

	"aether/internal/domain"
)

func (c *Core) GetBranding(orgID string) (domain.Branding, error) {
	return c.Store.GetBranding(orgID)
}

func (c *Core) SaveBranding(b *domain.Branding) error {
	b.UpdatedAt = time.Now().UTC()
	return c.Store.SaveBranding(b)
}
