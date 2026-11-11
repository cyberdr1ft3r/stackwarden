package tools

import "errors"

type InstallKind string

const (
	InstallKindCompose    InstallKind = "compose"
	InstallKindLinuxCLI   InstallKind = "linux_cli"
	InstallKindBundleOnly InstallKind = "bundle_only"
)

type Tool struct {
	ID                    string      `json:"id"`
	Name                  string      `json:"name"`
	Description           string      `json:"description"`
	Tags                  []string    `json:"tags"`
	InstallKind           InstallKind `json:"install_kind"`
	InstallAllowed        bool        `json:"install_allowed"`
	InstallDisabledReason string      `json:"install_disabled_reason,omitempty"`
	ServicePorts          []int       `json:"service_ports,omitempty"`
	Status                StatusSpec  `json:"status,omitempty"`
	Uninstall             Uninstall   `json:"uninstall,omitempty"`
}

type StatusSpec struct {
	Binary       string   `json:"binary,omitempty"`
	CheckCmd     []string `json:"check_cmd,omitempty"`
	VersionCmd   []string `json:"version_cmd,omitempty"`
	VersionRegex string   `json:"version_regex,omitempty"`
}

type Uninstall struct {
	UninstallCmds [][]string `json:"uninstall_cmds,omitempty"`
}

var Catalog = []Tool{
	{
		ID:                    "portainer",
		Name:                  "Portainer CE (LTS)",
		Description:           "Existing Portainer deployments can be inspected or explicitly uninstalled; new installs are disabled.",
		Tags:                  []string{"docker", "management"},
		InstallKind:           InstallKindCompose,
		InstallAllowed:        false,
		InstallDisabledReason: "new Portainer installations are disabled because the prior template exposed a Docker-socket management service",
		ServicePorts:          []int{9000, 9443},
	},
	{
		ID:             "ddev",
		Name:           "DDEV",
		Description:    "Installer script and starter config for DDEV.",
		Tags:           []string{"php", "web", "local-dev"},
		InstallKind:    InstallKindLinuxCLI,
		InstallAllowed: true,
		Status: StatusSpec{
			Binary:       "ddev",
			VersionCmd:   []string{"ddev", "version"},
			VersionRegex: `v?(\d+\.\d+\.\d+)`,
		},
		Uninstall: Uninstall{
			UninstallCmds: [][]string{
				{"apt-get", "remove", "-y", "ddev", "mkcert"},
			},
		},
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
