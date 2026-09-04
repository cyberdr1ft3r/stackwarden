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
	if !validToolID(toolID) {
		return "", errors.New("invalid tool id")
	}
	base := path.Join("templates", toolID)
	if path.Dir(base) != "templates" {
		return "", errors.New("template path escapes base")
	}
	if _, err := fs.Stat(templateFS, base); err != nil {
		return "", err
	}
	return base, nil
}

func validToolID(toolID string) bool {
	if toolID == "" || strings.Contains(toolID, "..") || path.IsAbs(toolID) {
		return false
	}
	for _, char := range toolID {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}
