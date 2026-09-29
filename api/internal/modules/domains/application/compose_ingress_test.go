package application

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/domains/domain"
)

type composeIngressStore struct {
	domain.Store
}

func (composeIngressStore) UpdateDomainProvision(context.Context, uuid.UUID, uuid.UUID, string, string, string, *time.Time, int) error {
	return nil
}

func TestComposeDomainProvisionUsesPrivateServiceAliasAndHTTPS(t *testing.T) {
	root := t.TempDir()
	serviceID := uuid.New()
	domainID := uuid.New()
	worker := &ProvisionWorker{
		Store:       composeIngressStore{},
		Provisioner: &Provisioner{TraefikDir: root},
	}
	configuredDomain := &domain.Domain{
		ID: domainID, AppID: uuid.New(), ServiceID: serviceID, ServiceType: ServiceTypeCompose,
		Host: "gdrive.example.test", HTTPS: true, Path: "/", InternalPath: "/",
		ContainerPort: 9000, ComposeServiceName: "gdrive-s3",
	}
	worker.provision(context.Background(), configuredDomain)

	path := filepath.Join(root, "dynamic", "domain-"+domainID.String()+".yml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated Traefik route: %v", err)
	}
	configuration := string(content)
	expectedAlias := "app-" + serviceID.String()[:8] + "-gdrive-s3"
	for _, expected := range []string{
		"Host(`gdrive.example.test`)",
		"entryPoints:\n      - websecure",
		"certResolver: letsencrypt",
		"url: \"http://" + expectedAlias + ":9000/\"",
	} {
		if !strings.Contains(configuration, expected) {
			t.Fatalf("Traefik route is missing %q: %s", expected, configuration)
		}
	}
}
