package command

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidfic/luminarr/pkg/plugin"
)

func TestValidateScriptName_Valid(t *testing.T) {
	for _, name := range []string{"on-import.sh", "my_script.py", "notify", "test.sh"} {
		if err := validateScriptName(name); err != nil {
			t.Errorf("validateScriptName(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateScriptName_PathTraversal(t *testing.T) {
	for _, name := range []string{"../etc/passwd", "foo/bar.sh", "..\\evil", "..", "a/b"} {
		if err := validateScriptName(name); err == nil {
			t.Errorf("validateScriptName(%q) = nil, want error", name)
		}
	}
}

func TestFactory_MissingScriptName(t *testing.T) {
	// The factory (init) checks for empty script_name before calling New.
	// Here we verify Test() on an empty-name notifier returns an error
	// from resolveScript since the path is empty.
	n := New(Config{ScriptName: ""})
	if err := n.Test(context.Background()); err == nil {
		t.Fatal("expected error for empty script name")
	}
}

func TestFactory_InvalidScriptName(t *testing.T) {
	if err := validateScriptName("../bad"); err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestNotify_ScriptExecution(t *testing.T) {
	tmp := t.TempDir()
	orig := ScriptsDir
	ScriptsDir = tmp
	defer func() { ScriptsDir = orig }()

	marker := filepath.Join(tmp, "marker.txt")
	script := filepath.Join(tmp, "test.sh")
	// Script writes env vars to marker file and copies stdin to marker.stdin.
	content := "#!/bin/sh\necho \"TYPE=$LUMINARR_EVENT_TYPE\" > " + marker + "\ncat > " + marker + ".stdin\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	n := New(Config{ScriptName: "test.sh", Timeout: 10})
	event := plugin.NotificationEvent{
		Type:      "grab_started",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		MovieID:   "abc-123",
		Message:   "Grabbing release: Test Movie",
	}

	if err := n.Notify(context.Background(), event); err != nil {
		t.Fatalf("Notify() = %v", err)
	}

	// Verify env vars were written.
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("reading marker: %v", err)
	}
	if got := string(data); got != "TYPE=grab_started\n" {
		t.Errorf("marker = %q, want TYPE=grab_started\\n", got)
	}

	// Verify stdin was piped.
	stdin, err := os.ReadFile(marker + ".stdin")
	if err != nil {
		t.Fatalf("reading stdin marker: %v", err)
	}
	if len(stdin) == 0 {
		t.Error("stdin was empty, expected JSON payload")
	}
}

func TestNotify_ScriptTimeout(t *testing.T) {
	tmp := t.TempDir()
	orig := ScriptsDir
	ScriptsDir = tmp
	defer func() { ScriptsDir = orig }()

	script := filepath.Join(tmp, "slow.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	n := New(Config{ScriptName: "slow.sh", Timeout: 1})
	err := n.Notify(context.Background(), plugin.NotificationEvent{Type: "test"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestTest_ScriptExists(t *testing.T) {
	tmp := t.TempDir()
	orig := ScriptsDir
	ScriptsDir = tmp
	defer func() { ScriptsDir = orig }()

	script := filepath.Join(tmp, "ok.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	n := New(Config{ScriptName: "ok.sh"})
	if err := n.Test(context.Background()); err != nil {
		t.Fatalf("Test() = %v", err)
	}
}

func TestTest_ScriptMissing(t *testing.T) {
	tmp := t.TempDir()
	orig := ScriptsDir
	ScriptsDir = tmp
	defer func() { ScriptsDir = orig }()

	n := New(Config{ScriptName: "nope.sh"})
	if err := n.Test(context.Background()); err == nil {
		t.Fatal("expected error for missing script")
	}
}

func TestTest_ScriptNotExecutable(t *testing.T) {
	tmp := t.TempDir()
	orig := ScriptsDir
	ScriptsDir = tmp
	defer func() { ScriptsDir = orig }()

	script := filepath.Join(tmp, "noexec.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	n := New(Config{ScriptName: "noexec.sh"})
	if err := n.Test(context.Background()); err == nil {
		t.Fatal("expected error for non-executable script")
	}
}
