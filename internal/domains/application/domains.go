package application

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	appsdomain "aether/internal/apps/domain"
	"aether/internal/domains/domain"
)

type Domains struct {
	Store       domain.Store
	Apps        AppStore
	Provisioner *Provisioner
}

type AppStore interface {
	GetApp(ctx context.Context, id, orgID uuid.UUID) (*appsdomain.App, error)
}

type AddDomainInput struct {
	Host          string
	HTTPS         bool
	Path          string
	InternalPath  string
	StripPath     bool
	ContainerPort int
	ServerID      uuid.UUID
}

func (d *Domains) Add(ctx context.Context, appID, orgID uuid.UUID, in AddDomainInput) (*domain.Domain, error) {
	app, err := d.Apps.GetApp(ctx, appID, orgID)
	if err != nil {
		return nil, err
	}
	in.Host = strings.ToLower(strings.TrimSpace(in.Host))
	if err := d.Provisioner.ValidateHost(in.Host); err != nil {
		return nil, err
	}
	if in.ContainerPort <= 0 {
		if app.BuildType == "custom" {
			in.ContainerPort = 80
		} else {
			in.ContainerPort = app.Port
		}
	}
	dom, err := d.Store.CreateDomain(ctx, &domain.Domain{
		AppID: appID, ServerID: in.ServerID, Host: in.Host, HTTPS: in.HTTPS,
		Path: orDefault(in.Path, "/"), InternalPath: orDefault(in.InternalPath, "/"),
		StripPath: in.StripPath, ContainerPort: in.ContainerPort,
		Status: string(domain.DomainProvisioning),
	})
	if err != nil {
		return nil, err
	}
	return dom, nil
}

func (d *Domains) GenerateFreeDomain(ctx context.Context, appID, orgID uuid.UUID, https bool) (*domain.Domain, error) {
	app, err := d.Apps.GetApp(ctx, appID, orgID)
	if err != nil {
		return nil, err
	}
	in := AddDomainInput{
		Host: d.Provisioner.GenerateFreeDomain(app.Name), HTTPS: https,
		Path: "/", InternalPath: "/",
	}
	return d.Add(ctx, appID, orgID, in)
}

func (d *Domains) List(ctx context.Context, appID, orgID uuid.UUID) ([]domain.Domain, error) {
	if _, err := d.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	return d.Store.ListDomains(ctx, appID)
}

func (d *Domains) Remove(ctx context.Context, appID, orgID uuid.UUID, host string) error {
	if _, err := d.Apps.GetApp(ctx, appID, orgID); err != nil {
		return err
	}
	dom, err := d.Store.GetDomainByHost(ctx, appID, strings.ToLower(host))
	if err != nil {
		return err
	}
	_ = d.Store.UpdateDomainStatus(ctx, dom.ID, appID, string(domain.DomainRemoving), dom.CertStatus)
	if err := d.Provisioner.RemoveDomainConfig(dom); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = d.Store.UpdateDomainStatus(ctx, dom.ID, appID, string(domain.DomainRemoved), dom.CertStatus)
	return d.Store.DeleteDomain(ctx, dom.ID, appID)
}

func (d *Domains) Reprovision(ctx context.Context, appID, orgID uuid.UUID, domainID uuid.UUID) error {
	dom, err := d.getAppDomain(ctx, appID, orgID, domainID)
	if err != nil {
		return err
	}
	_ = d.Store.UpdateDomainProvision(ctx, dom.ID, appID, string(domain.DomainProvisioning), dom.CertStatus, "", nil, 0)
	return nil
}

func (d *Domains) UpdateDomain(ctx context.Context, appID, orgID uuid.UUID, domainID uuid.UUID, in AddDomainInput) error {
	dom, err := d.getAppDomain(ctx, appID, orgID, domainID)
	if err != nil {
		return err
	}
	in.Host = strings.ToLower(strings.TrimSpace(in.Host))
	if err := d.Provisioner.ValidateHost(in.Host); err != nil {
		return err
	}
	if in.ContainerPort <= 0 {
		in.ContainerPort = dom.ContainerPort
	}
	_ = d.Store.UpdateDomainFields(ctx, dom.ID, appID, in.Host, in.HTTPS,
		orDefault(in.Path, "/"), orDefault(in.InternalPath, "/"), in.StripPath, in.ContainerPort)
	return nil
}

func (d *Domains) GetDomain(ctx context.Context, appID, orgID uuid.UUID, domainID uuid.UUID) (*domain.Domain, error) {
	return d.getAppDomain(ctx, appID, orgID, domainID)
}

func (d *Domains) Verify(ctx context.Context, appID, orgID uuid.UUID, domainID uuid.UUID) error {
	dom, err := d.getAppDomain(ctx, appID, orgID, domainID)
	if err != nil {
		return err
	}
	_ = d.Store.UpdateDomainProvision(ctx, dom.ID, appID, string(domain.DomainProvisioning), dom.CertStatus, "", nil, 0)
	return nil
}

func (d *Domains) getAppDomain(ctx context.Context, appID, orgID, domainID uuid.UUID) (*domain.Domain, error) {
	if _, err := d.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	dom, err := d.Store.GetDomainByID(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if dom.AppID != appID {
		return nil, domain.ErrNotFound
	}
	return dom, nil
}

func (d *Domains) CreatePreview(ctx context.Context, appID, orgID uuid.UUID, branch string) (*domain.Preview, error) {
	app, err := d.Apps.GetApp(ctx, appID, orgID)
	if err != nil {
		return nil, err
	}
	if app.SourceType != "git" {
		return nil, domain.ErrValidation
	}
	if branch == "" {
		branch = "preview"
	}
	branch = sanitizeBranch(branch)
	previewDomain := app.PreviewDomain
	if previewDomain == "" {
		previewDomain = "preview.aether.local"
	}
	return d.Store.CreatePreview(ctx, &domain.Preview{
		AppID: appID, Branch: branch, Status: "active",
		Domain: fmt.Sprintf("%s-%s.%s", app.Name, branch, previewDomain),
	})
}

func (d *Domains) ListPreviews(ctx context.Context, appID, orgID uuid.UUID) ([]domain.Preview, error) {
	if _, err := d.Apps.GetApp(ctx, appID, orgID); err != nil {
		return nil, err
	}
	return d.Store.ListPreviews(ctx, appID)
}

func (d *Domains) DeletePreview(ctx context.Context, previewID, orgID uuid.UUID) error {
	preview, err := d.Store.GetPreviewByID(ctx, previewID)
	if err != nil {
		return err
	}
	if _, err := d.Apps.GetApp(ctx, preview.AppID, orgID); err != nil {
		return err
	}
	return d.Store.DeletePreview(ctx, preview.ID, preview.AppID)
}

func (d *Domains) Certificates(ctx context.Context, orgID uuid.UUID) ([]domain.Certificate, error) {
	return d.Store.ListCertificatesByOrg(ctx, orgID)
}

func sanitizeBranch(branch string) string {
	replacer := strings.NewReplacer("/", "-", ".", "-", "_", "-")
	return replacer.Replace(branch)
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
