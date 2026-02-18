package main

import "testing"

func TestResolveAPIBind_PortFlagPreservesHostFromEnv(t *testing.T) {
	t.Setenv("API_BIND", "127.0.0.1:8080")

	bind, err := resolveAPIBind(6000, true)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "127.0.0.1:6000" {
		t.Fatalf("expected bind 127.0.0.1:6000, got %q", bind)
	}
}

func TestResolveAPIBind_WithoutPortFlagUsesEnvBind(t *testing.T) {
	t.Setenv("API_BIND", "127.0.0.1:8080")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "127.0.0.1:8080" {
		t.Fatalf("expected bind 127.0.0.1:8080, got %q", bind)
	}
}

func TestResolveAPIBind_DefaultWhenNoEnvAndNoPortFlag(t *testing.T) {
	t.Setenv("API_BIND", "")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != ":8080" {
		t.Fatalf("expected bind :8080, got %q", bind)
	}
}

func TestResolveAPIBind_WithoutPortFlagAllowsEphemeralPortOnAnyHost(t *testing.T) {
	t.Setenv("API_BIND", ":0")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != ":0" {
		t.Fatalf("expected bind :0, got %q", bind)
	}
}

func TestResolveAPIBind_WithoutPortFlagAllowsEphemeralPortOnSpecificHost(t *testing.T) {
	t.Setenv("API_BIND", "127.0.0.1:0")

	bind, err := resolveAPIBind(6000, false)
	if err != nil {
		t.Fatalf("resolveAPIBind returned error: %v", err)
	}
	if bind != "127.0.0.1:0" {
		t.Fatalf("expected bind 127.0.0.1:0, got %q", bind)
	}
}

func TestResolveAPIBind_WithoutPortFlagRejectsOutOfRangePort(t *testing.T) {
	t.Setenv("API_BIND", ":65536")

	if _, err := resolveAPIBind(6000, false); err == nil {
		t.Fatal("expected error for API_BIND=:65536")
	}
}

func TestResolveAPIBind_WithoutPortFlagRejectsNegativePort(t *testing.T) {
	t.Setenv("API_BIND", ":-1")

	if _, err := resolveAPIBind(6000, false); err == nil {
		t.Fatal("expected error for API_BIND=:-1")
	}
}

func TestResolveAPIBind_WithoutPortFlagRejectsNonNumericPort(t *testing.T) {
	t.Setenv("API_BIND", ":abc")

	if _, err := resolveAPIBind(6000, false); err == nil {
		t.Fatal("expected error for API_BIND=:abc")
	}
}
