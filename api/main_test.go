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
