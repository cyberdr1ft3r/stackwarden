//go:build !windows

package main

import (
	"context"
	"net"
	"net/http"
	"time"
)

const dockerSocketPath = "/var/run/docker.sock"

func newDockerEngineClient() (dockerDoer, error) {
	transport := &http.Transport{
		DisableCompression:    true,
		DisableKeepAlives:     true,
		MaxConnsPerHost:       1,
		ResponseHeaderTimeout: dockerRequestTimeout,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: dockerRequestTimeout}
			return dialer.DialContext(ctx, "unix", dockerSocketPath)
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   dockerRequestTimeout + time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}
