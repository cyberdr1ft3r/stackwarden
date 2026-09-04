//go:build !windows

package main

import "testing"

func TestParseSSLtnOutput(t *testing.T) {
	sample := `
LISTEN 0      4096          0.0.0.0:22               0.0.0.0:*
LISTEN 0      4096          [::]:22                  [::]:*
LISTEN 0      4096          0.0.0.0:80               0.0.0.0:*
LISTEN 0      4096          [::]:443                 [::]:*
LISTEN 0      4096          127.0.0.1:8080           0.0.0.0:*
LISTEN 0      4096          *:9091                   *:*
`

	ports := parseSSLtnOutput(sample)
	if len(ports) != 5 {
		t.Fatalf("expected 5 unique ports, got %d", len(ports))
	}

	expected := map[int]string{
		22:   "0.0.0.0",
		80:   "0.0.0.0",
		443:  "::",
		8080: "127.0.0.1",
		9091: "*",
	}

	for port, addr := range expected {
		found := false
		for _, p := range ports {
			if p.LocalPort == port {
				found = true
				if p.LocalAddress != addr {
					t.Fatalf("port %d expected address %s, got %s", port, addr, p.LocalAddress)
				}
				if p.Protocol != "tcp" {
					t.Fatalf("port %d expected protocol tcp, got %s", port, p.Protocol)
				}
			}
		}
		if !found {
			t.Fatalf("port %d not found", port)
		}
	}
}
