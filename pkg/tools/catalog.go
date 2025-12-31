package tools

import "errors"

type InstallKind string

const (
	InstallKindCompose    InstallKind = "compose"
	InstallKindLinuxCLI   InstallKind = "linux_cli"
	InstallKindBundleOnly InstallKind = "bundle_only"
)

type Tool struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Tags         []string    `json:"tags"`
	InstallKind  InstallKind `json:"install_kind"`
	ServicePorts []int       `json:"service_ports,omitempty"`
}

var Catalog = []Tool{
	{
		ID:           "portainer",
		Name:         "Portainer CE (LTS)",
		Description:  "Docker Compose stack for Portainer CE LTS.",
		Tags:         []string{"docker", "management"},
		InstallKind:  InstallKindCompose,
		ServicePorts: []int{9000, 9443},
	},
	{
		ID:          "ddev",
		Name:        "DDEV",
		Description: "Installer script and starter config for DDEV.",
		Tags:        []string{"php", "web", "local-dev"},
		InstallKind: InstallKindLinuxCLI,
	},
}

func Find(id string) (Tool, error) {
	for _, t := range Catalog {
		if t.ID == id {
			return t, nil
		}
	}
	return Tool{}, errors.New("tool not found")
}
