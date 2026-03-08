package logging

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestTeeHandler_CapturesEntries(t *testing.T) {
	buf := NewRingBuffer(100)
	var out bytes.Buffer
	inner := slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewTeeHandler(inner, buf))

	logger.Info("hello world", "key", "value")
	logger.Warn("something bad", "code", 42)
	logger.Debug("trace detail")

	entries := buf.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries in buffer, got %d", len(entries))
	}

	if entries[0].Message != "hello world" {
		t.Errorf("first entry message = %q, want %q", entries[0].Message, "hello world")
	}
	if entries[0].Level != "INFO" {
		t.Errorf("first entry level = %q, want INFO", entries[0].Level)
	}
	if entries[0].Fields["key"] != "value" {
		t.Errorf("first entry field key = %v, want 'value'", entries[0].Fields["key"])
	}

	if entries[1].Level != "WARN" {
		t.Errorf("second entry level = %q, want WARN", entries[1].Level)
	}

	if entries[2].Level != "DEBUG" {
		t.Errorf("third entry level = %q, want DEBUG", entries[2].Level)
	}

	// Also verify the inner handler received output.
	if out.Len() == 0 {
		t.Error("inner handler should have received output")
	}
}

func TestTeeHandler_WithAttrs(t *testing.T) {
	buf := NewRingBuffer(100)
	inner := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewTeeHandler(inner, buf)).With("service", "test")

	logger.Info("with attrs")

	entries := buf.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Fields["service"] != "test" {
		t.Errorf("expected service=test in fields, got %v", entries[0].Fields)
	}
}

func TestTeeHandler_WithGroup(t *testing.T) {
	buf := NewRingBuffer(100)
	inner := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewTeeHandler(inner, buf)).WithGroup("http")

	logger.Info("request", "method", "GET")

	entries := buf.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Fields["http.method"] != "GET" {
		t.Errorf("expected http.method=GET in fields, got %v", entries[0].Fields)
	}
}

func TestTeeHandler_LevelFiltering(t *testing.T) {
	buf := NewRingBuffer(100)
	// Inner handler only accepts INFO+, but our buffer should capture what the handler enables.
	inner := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(NewTeeHandler(inner, buf))

	logger.Debug("should not appear")
	logger.Info("should appear")

	entries := buf.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (debug filtered), got %d", len(entries))
	}
	if entries[0].Message != "should appear" {
		t.Errorf("expected 'should appear', got %q", entries[0].Message)
	}
}
