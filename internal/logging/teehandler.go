package logging

import (
	"context"
	"log/slog"
)

// TeeHandler wraps an existing slog.Handler and also writes each record to a
// RingBuffer for in-memory log viewing via the API.
type TeeHandler struct {
	inner  slog.Handler
	buf    *RingBuffer
	groups []string
	attrs  []slog.Attr
}

// NewTeeHandler creates a handler that forwards to inner and captures to buf.
func NewTeeHandler(inner slog.Handler, buf *RingBuffer) *TeeHandler {
	return &TeeHandler{inner: inner, buf: buf}
}

func (h *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	// Capture to ring buffer.
	fields := make(map[string]any)
	// Add handler-level attrs first (from WithAttrs).
	for _, a := range h.attrs {
		fields[a.Key] = a.Value.Any()
	}
	// Add record-level attrs.
	r.Attrs(func(a slog.Attr) bool {
		key := a.Key
		// Prefix with group names if any.
		for i := len(h.groups) - 1; i >= 0; i-- {
			key = h.groups[i] + "." + key
		}
		fields[key] = a.Value.Any()
		return true
	})

	h.buf.Add(Entry{
		Time:    r.Time,
		Level:   r.Level.String(),
		Message: r.Message,
		Fields:  fields,
	})

	// Forward to the underlying handler.
	return h.inner.Handle(ctx, r)
}

func (h *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TeeHandler{
		inner:  h.inner.WithAttrs(attrs),
		buf:    h.buf,
		groups: h.groups,
		attrs:  append(append([]slog.Attr{}, h.attrs...), attrs...),
	}
}

func (h *TeeHandler) WithGroup(name string) slog.Handler {
	return &TeeHandler{
		inner:  h.inner.WithGroup(name),
		buf:    h.buf,
		groups: append(append([]string{}, h.groups...), name),
		attrs:  h.attrs,
	}
}
