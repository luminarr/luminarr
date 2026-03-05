// Package command implements a Luminarr notification plugin that executes
// a user-provided script from /config/scripts/ on each event. The event
// payload is passed as JSON on stdin and key fields are set as environment
// variables.
package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/pkg/plugin"
)

// ScriptsDir is the directory from which scripts are resolved.
// Package-level var so tests can override it.
var ScriptsDir = "/config/scripts" //nolint:gochecknoglobals // injectable for tests

func init() {
	registry.Default.RegisterNotifier("command", func(settings json.RawMessage) (plugin.Notifier, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("command: invalid settings: %w", err)
		}
		if cfg.ScriptName == "" {
			return nil, fmt.Errorf("command: script_name is required")
		}
		if err := validateScriptName(cfg.ScriptName); err != nil {
			return nil, err
		}
		return New(cfg), nil
	})
	registry.Default.RegisterNotifierSanitizer("command", func(settings json.RawMessage) json.RawMessage {
		return settings // script_name is not sensitive
	})
}

// Config holds the user-supplied settings for a command notifier.
type Config struct {
	ScriptName string `json:"script_name"`
	Timeout    int    `json:"timeout,omitempty"` // seconds; default 30
}

// Notifier is a command notifier plugin instance.
type Notifier struct {
	cfg Config
}

// New creates a new Notifier from the given config.
func New(cfg Config) *Notifier {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}
	return &Notifier{cfg: cfg}
}

func (*Notifier) Name() string { return "Command" }

// Notify executes the configured script with event data.
func (n *Notifier) Notify(ctx context.Context, event plugin.NotificationEvent) error {
	scriptPath, err := resolveScript(n.cfg.ScriptName)
	if err != nil {
		return err
	}

	timeout := time.Duration(n.cfg.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("command: marshaling event: %w", err)
	}

	cmd := exec.CommandContext(ctx, scriptPath) //nolint:gosec // path validated via resolveScript
	cmd.WaitDelay = 3 * time.Second // give pipes time to drain after kill
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Env = append(os.Environ(),
		"LUMINARR_EVENT_TYPE="+string(event.Type),
		"LUMINARR_MOVIE_ID="+event.MovieID,
		"LUMINARR_MESSAGE="+event.Message,
		"LUMINARR_TIMESTAMP="+event.Timestamp.Format(time.RFC3339),
	)

	// Use a buffer for combined output instead of CombinedOutput() which
	// can hang on context cancellation waiting for pipe reads.
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	err = cmd.Run()
	if outBuf.Len() > 0 {
		slog.Info("command notifier output", "script", n.cfg.ScriptName, "output", outBuf.String())
	}
	if err != nil {
		return fmt.Errorf("command: script %q failed: %w", n.cfg.ScriptName, err)
	}
	return nil
}

// Test verifies the script file exists and is executable.
func (n *Notifier) Test(_ context.Context) error {
	scriptPath, err := resolveScript(n.cfg.ScriptName)
	if err != nil {
		return err
	}
	info, err := os.Stat(scriptPath)
	if err != nil {
		return fmt.Errorf("command: script not found: %w", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("command: script %q is not executable", n.cfg.ScriptName)
	}
	return nil
}

// validateScriptName rejects names that could escape the scripts directory.
func validateScriptName(name string) error {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("command: script_name must be a plain filename (no path separators or ..)")
	}
	return nil
}

// resolveScript builds the absolute path and verifies it stays within ScriptsDir.
func resolveScript(name string) (string, error) {
	if err := validateScriptName(name); err != nil {
		return "", err
	}
	abs := filepath.Join(ScriptsDir, name)
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(ScriptsDir, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("command: script path escapes scripts directory")
	}
	return abs, nil
}
