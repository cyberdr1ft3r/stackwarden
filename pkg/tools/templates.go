package tools

import (
	"embed"
	"errors"
	"io/fs"
	"path"
	"strings"
)

//go:embed templates/**
var templateFS embed.FS

func TemplateFS() fs.FS {
	return templateFS
}

func TemplateBasePath(toolID string) (string, error) {
	if strings.TrimSpace(toolID) == "" {
		return "", errors.New("tool id required")
	}
	base := path.Join("templates", toolID)
	if _, err := fs.Stat(templateFS, base); err != nil {
		return "", err
	}
	return base, nil
}
