// Package pathutil provides shared path validation for file-accepting endpoints.
package pathutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sensitivePathPrefixes lists path prefixes that must never be accessed
// by file-accepting API endpoints.
var sensitivePathPrefixes = []string{
	"/config",
	"/etc",
	"/proc",
	"/sys",
	"/dev",
	"/run",
	"/var",
}

// ValidateContentPath rejects paths that are not absolute, contain traversal
// components after cleaning, or overlap with sensitive system directories.
func ValidateContentPath(p string) error {
	if p == "" {
		return errors.New("empty content_path")
	}
	if !filepath.IsAbs(p) {
		return fmt.Errorf("content_path must be absolute, got %q", p)
	}
	clean := filepath.Clean(p)
	for _, prefix := range sensitivePathPrefixes {
		if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
			return fmt.Errorf("content_path %q is within a restricted directory", clean)
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		configDir := filepath.Join(home, ".config", "luminarr")
		if clean == configDir || strings.HasPrefix(clean, configDir+"/") {
			return fmt.Errorf("content_path %q is within the Luminarr config directory", clean)
		}
	}
	return nil
}
