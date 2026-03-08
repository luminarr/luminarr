package logging

import (
	"sync"
	"time"
)

// Entry is a single log entry stored in the ring buffer.
type Entry struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// RingBuffer is a fixed-size, thread-safe circular buffer of log entries.
// When full, the oldest entry is overwritten.
type RingBuffer struct {
	mu      sync.Mutex
	entries []Entry
	head    int  // next write position
	full    bool // whether the buffer has wrapped
}

// NewRingBuffer creates a ring buffer that holds up to size entries.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		entries: make([]Entry, size),
	}
}

// Add appends an entry to the buffer, overwriting the oldest if full.
func (rb *RingBuffer) Add(e Entry) {
	rb.mu.Lock()
	rb.entries[rb.head] = e
	rb.head++
	if rb.head == len(rb.entries) {
		rb.head = 0
		rb.full = true
	}
	rb.mu.Unlock()
}

// Entries returns a copy of all buffered entries, oldest first.
func (rb *RingBuffer) Entries() []Entry {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	n := rb.Len()
	result := make([]Entry, n)
	if n == 0 {
		return result
	}

	if rb.full {
		// Oldest entries start at rb.head (the next write position wraps around).
		copied := copy(result, rb.entries[rb.head:])
		copy(result[copied:], rb.entries[:rb.head])
	} else {
		copy(result, rb.entries[:rb.head])
	}
	return result
}

// Len returns the number of entries currently in the buffer.
func (rb *RingBuffer) Len() int {
	if rb.full {
		return len(rb.entries)
	}
	return rb.head
}
