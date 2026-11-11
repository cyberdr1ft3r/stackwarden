package pkg

import (
	"strings"
	"testing"
)

func TestValidateDockerDiscoveryRejectsUnboundedOrOwnershipData(t *testing.T) {
	valid := DockerDiscovery{
		Status: DockerStatusAvailable,
		Containers: []DockerContainer{{
			ID:    "docker:0123456789abcdef",
			State: "running",
			Running: func() *bool {
				value := true
				return &value
			}(),
		}},
	}
	if err := ValidateDockerDiscovery(valid); err != nil {
		t.Fatalf("valid discovery rejected: %v", err)
	}

	oversized := valid
	oversized.Containers[0].ImageReference = strings.Repeat("x", MaxDockerImageBytes+1)
	if err := ValidateDockerDiscovery(oversized); err == nil {
		t.Fatal("expected oversized image reference to be rejected")
	}

	ownership := valid
	ownership.Containers[0].ImageReference = ""
	ownership.Containers[0].ComposeHint = &DockerComposeHint{Source: "ownership", Project: "site"}
	if err := ValidateDockerDiscovery(ownership); err == nil {
		t.Fatal("expected ownership-like hint source to be rejected")
	}
}
