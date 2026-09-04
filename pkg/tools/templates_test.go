package tools

import "testing"

func TestTemplateBasePathRejectsInvalidToolIDs(t *testing.T) {
	for _, toolID := range []string{"", "..", "../templates", "portainer/../ddev", "/portainer", "Portainer", "portainer.d"} {
		if _, err := TemplateBasePath(toolID); err == nil {
			t.Fatalf("expected %q to be rejected", toolID)
		}
	}
}

func TestTemplateBasePathAcceptsCatalogTool(t *testing.T) {
	base, err := TemplateBasePath("portainer")
	if err != nil {
		t.Fatalf("expected catalog tool to be accepted: %v", err)
	}
	if base != "templates/portainer" {
		t.Fatalf("unexpected template base: %q", base)
	}
}
