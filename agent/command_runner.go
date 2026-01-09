package main

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

type Runner interface {
	Run(ctx context.Context, cmd []string, workdir string) (stdout, stderr string, code int, err error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, cmd []string, workdir string) (string, string, int, error) {
	if len(cmd) == 0 {
		return "", "", -1, errors.New("command required")
	}

	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	if workdir != "" {
		c.Dir = workdir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	code := 0
	if err != nil {
		if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		} else {
			code = -1
		}
	}

	return truncateOutput(stdout.String()), truncateOutput(stderr.String()), code, err
}
