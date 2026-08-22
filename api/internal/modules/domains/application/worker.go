package application

import (
	"context"
	"time"

	"aether/internal/modules/domains/domain"
)

const (
	maxDomainRetries = 10
	provisionTick    = 15 * time.Second
)

// ProvisionWorker aplica a config Traefik dos domínios e resolve o estado do
// certificado, com retry e backoff exponencial. O Control Plane só persiste
// config; o Traefik do server detecta a rota sem redeploy.
type ProvisionWorker struct {
	Store       domain.Store
	Provisioner *Provisioner
}

func (w *ProvisionWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(provisionTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *ProvisionWorker) process(ctx context.Context) {
	domains, err := w.Store.ListProvisioningDomains(ctx, time.Now().UTC(), maxDomainRetries)
	if err != nil {
		return
	}
	for i := range domains {
		w.provision(ctx, &domains[i])
	}
}

func (w *ProvisionWorker) provision(ctx context.Context, d *domain.Domain) {
	alias := w.Provisioner.Alias(d.AppID, d.ServiceType)
	if err := w.Provisioner.WriteDomainConfig(d, alias, false); err != nil {
		w.scheduleRetry(ctx, d, err)
		return
	}
	if !d.HTTPS {
		_ = w.Store.UpdateDomainProvision(ctx, d.ID, d.AppID, string(domain.DomainActive), "active", "", nil, 0)
		return
	}
	if w.Provisioner.VerifyCertificate(d.Host) {
		_ = w.Provisioner.WriteDomainConfig(d, alias, true)
		_ = w.Store.UpdateDomainProvision(ctx, d.ID, d.AppID, string(domain.DomainActive), "active", "", nil, 0)
		return
	}
	_ = w.Store.UpdateDomainProvision(ctx, d.ID, d.AppID, string(domain.DomainActive), "pending", "", retryIn(d.RetryCount), d.RetryCount+1)
}

func (w *ProvisionWorker) scheduleRetry(ctx context.Context, d *domain.Domain, err error) {
	_ = w.Store.UpdateDomainProvision(ctx, d.ID, d.AppID, string(domain.DomainError), d.CertStatus, err.Error(), retryIn(d.RetryCount), d.RetryCount+1)
}

func retryIn(retries int) *time.Time {
	backoff := time.Duration(1<<min(retries, 7)) * 30 * time.Second
	t := time.Now().UTC().Add(backoff)
	return &t
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
