package core

import "testing"

func TestCatalogCount(t *testing.T) {
	c := testCore(t)
	defer c.Stop(timeoutCtxT())
	if err := c.SeedTemplates(); err != nil {
		t.Fatal(err)
	}
	all, err := c.Store.ListTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 100 {
		t.Fatalf("catálogo deveria ter >=100 templates, tem %d", len(all))
	}
	f := TemplateFilter{EditorsChoice: true}
	ec, _ := c.ListTemplatesFiltered(f)
	if len(ec) < 5 {
		t.Fatalf("editors choice deveria ter >=5, tem %d", len(ec))
	}
	ids := map[string]bool{}
	for _, t := range all {
		ids[t.ID] = true
	}
	for _, want := range []string{"tpl-postgresql", "tpl-grafana", "tpl-wordpress", "tpl-ollama", "tpl-calcom"} {
		if !ids[want] {
			t.Fatalf("template esperado ausente: %s", want)
		}
	}
}
