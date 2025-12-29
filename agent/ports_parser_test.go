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
