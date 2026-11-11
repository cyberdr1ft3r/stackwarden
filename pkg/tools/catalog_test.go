package tools

import "testing"

func TestCatalogInstallPolicyIsExplicit(t *testing.T) {
	portainer, err := Find("portainer")
	if err != nil {
		t.Fatal(err)
	}
	if portainer.InstallAllowed {
		t.Fatal("Portainer install must be disabled")
	}
	if portainer.InstallDisabledReason == "" {
		t.Fatal("disabled install must explain why")
	}

	ddev, err := Find("ddev")
	if err != nil {
		t.Fatal(err)
	}
	if !ddev.InstallAllowed {
		t.Fatal("DDEV install behavior must remain enabled in this scoped change")
	}
}
