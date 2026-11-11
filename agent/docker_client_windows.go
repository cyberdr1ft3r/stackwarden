//go:build windows

package main

import "errors"

func newDockerEngineClient() (dockerDoer, error) {
	return nil, errors.New("Docker Engine socket discovery is unavailable on Windows")
}
