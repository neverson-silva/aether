package http

import (
	"testing"

	"github.com/google/uuid"

	"aether/internal/modules/templates/domain"
)

func TestTemplateDTOUsesCatalogLogoIDForLocalTemplate(t *testing.T) {
	localID := uuid.New()
	logoID := uuid.New()
	result := templateDTO(&domain.Template{ID: localID, Name: "Casdoor", LogoTemplateID: logoID})

	want := "/api/v1/templates/" + logoID.String() + "/logo"
	if got := result["logo_url"]; got != want {
		t.Fatalf("logo_url = %v, want %s", got, want)
	}
}
