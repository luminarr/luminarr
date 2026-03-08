package logging

import (
	"testing"
	"time"
)

func TestRingBuffer_Basic(t *testing.T) {
	rb := NewRingBuffer(5)

	if rb.Len() != 0 {
		t.Fatalf("empty buffer should have len 0, got %d", rb.Len())
	}

	for i := range 3 {
		rb.Add(Entry{Message: string(rune('A' + i))})
	}

	if rb.Len() != 3 {
		t.Fatalf("expected len 3, got %d", rb.Len())
	}

	entries := rb.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Message != "A" || entries[2].Message != "C" {
		t.Errorf("entries should be A,B,C; got %s,%s,%s", entries[0].Message, entries[1].Message, entries[2].Message)
	}
}

func TestRingBuffer_Wrap(t *testing.T) {
	rb := NewRingBuffer(3)

	// Add 5 entries to a buffer of size 3.
	for i := range 5 {
		rb.Add(Entry{Message: string(rune('A' + i))})
	}

	if rb.Len() != 3 {
		t.Fatalf("expected len 3 after wrap, got %d", rb.Len())
	}

	entries := rb.Entries()
	// Should contain C, D, E (oldest first).
	if entries[0].Message != "C" || entries[1].Message != "D" || entries[2].Message != "E" {
		t.Errorf("expected C,D,E after wrap; got %s,%s,%s", entries[0].Message, entries[1].Message, entries[2].Message)
	}
}

func TestRingBuffer_EntriesReturnsCopy(t *testing.T) {
	rb := NewRingBuffer(5)
	rb.Add(Entry{Message: "original"})

	entries := rb.Entries()
	entries[0].Message = "mutated"

	// The buffer should still have the original.
	fresh := rb.Entries()
	if fresh[0].Message != "original" {
		t.Error("Entries() should return a copy, but original was mutated")
	}
}

func TestRingBuffer_PreservesFields(t *testing.T) {
	rb := NewRingBuffer(5)
	rb.Add(Entry{
		Time:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Level:   "INFO",
		Message: "test message",
		Fields:  map[string]any{"key": "value"},
	})

	entries := rb.Entries()
	if entries[0].Level != "INFO" {
		t.Errorf("level should be INFO, got %s", entries[0].Level)
	}
	if entries[0].Fields["key"] != "value" {
		t.Errorf("field key should be value, got %v", entries[0].Fields["key"])
	}
}
