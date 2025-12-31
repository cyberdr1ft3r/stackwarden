package main

import "testing"

func TestParseWindowsNetstatOutput(t *testing.T) {
	sample := `
Proto  Local Address          Foreign Address        State           PID
TCP    0.0.0.0:135            0.0.0.0:0              LISTENING       1000
TCP    127.0.0.1:8080         0.0.0.0:0              LISTENING       2000
TCP    [::]:443               [::]:0                 LISTENING       3000
TCP    [fe80::1%12]:9000      [::]:0                 ESTABLISHED     4000
UDP    0.0.0.0:68             *:*                                    5000
UDP    [::]:546               *:*                                    6000
`

	ports := parseWindowsNetstatOutput(sample)
	if len(ports) != 5 {
		t.Fatalf("expected 5 listening entries, got %d", len(ports))
	}

	find := func(addr string, port int) Port {
		for _, p := range ports {
			if p.LocalAddress == addr && p.LocalPort == port {
				return p
			}
		}
		t.Fatalf("did not find %s:%d", addr, port)
		return Port{}
	}

	p := find("0.0.0.0", 135)
	if p.Protocol != "tcp" || p.State != "LISTENING" || p.PID != 1000 {
		t.Fatalf("unexpected TCP entry: %+v", p)
	}

	p = find("127.0.0.1", 8080)
	if p.Protocol != "tcp" || p.PID != 2000 {
		t.Fatalf("unexpected loopback entry: %+v", p)
	}

	p = find("::", 443)
	if p.Protocol != "tcp" || p.PID != 3000 {
		t.Fatalf("unexpected IPv6 entry: %+v", p)
	}

	p = find("0.0.0.0", 68)
	if p.Protocol != "udp" || p.State != "" || p.PID != 5000 {
		t.Fatalf("unexpected UDP entry: %+v", p)
	}

	p = find("::", 546)
	if p.Protocol != "udp" || p.PID != 6000 {
		t.Fatalf("unexpected IPv6 UDP entry: %+v", p)
	}
}

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
