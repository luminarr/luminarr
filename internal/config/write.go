package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// writeAtomic writes data to path atomically: it writes to a temp file in the
// same directory, then renames it over the target. On POSIX filesystems rename
// is atomic, so the target is never left in a partially-written state.
func writeAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".luminarr-config-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("setting temp file permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}

// WriteConfigKey writes a single dot-notation key (e.g. "tmdb.api_key") to the
// given YAML config file, creating the file and parent directories if needed.
// If configFile is empty, the default path (~/.config/luminarr/config.yaml, or
// /config/config.yaml in Docker) is used.
//
// Existing keys in the file are preserved; only the target key is updated.
// Returns the actual file path that was written.
func WriteConfigKey(configFile, key, value string) (writePath string, err error) {
	path := configFile
	if path == "" {
		// Mirror the search order used by Load(): prefer /config (Docker volume
		// mount point) when that directory exists, then fall back to the
		// per-user path. This ensures the written key is found on the next
		// startup even when $HOME is set (as it usually is inside containers).
		if _, err := os.Stat("/config"); err == nil {
			path = "/config/config.yaml"
		} else {
			home, _ := os.UserHomeDir()
			if home != "" {
				path = filepath.Join(home, ".config", "luminarr", "config.yaml")
			} else {
				path = "/config/config.yaml"
			}
		}
	}

	// Read existing config into a generic map to preserve all other keys.
	data := map[string]interface{}{}
	if raw, readErr := os.ReadFile(path); readErr == nil {
		_ = yaml.Unmarshal(raw, &data)
	}

	// Set the key using dot-notation (supports one level of nesting only:
	// "parent.child"). Values nested deeper than one level are not needed today.
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 2 {
		sub, ok := data[parts[0]].(map[string]interface{})
		if !ok {
			sub = map[string]interface{}{}
		}
		sub[parts[1]] = value
		data[parts[0]] = sub
	} else {
		data[key] = value
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}

	raw, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}

	if err := writeAtomic(path, raw, 0o600); err != nil {
		return "", fmt.Errorf("writing config file: %w", err)
	}

	return path, nil
}
