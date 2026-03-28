package plugin

import (
	"context"
	"time"
)

// WatchProvider is an optional interface for media servers that can report
// watch history. Plugins that don't support it simply don't implement it.
// Check with a type assertion at runtime:
//
//	if wp, ok := server.(WatchProvider); ok { ... }
type WatchProvider interface {
	// WatchHistory returns watch events since the given timestamp.
	// Each event represents one completed playback (>= 90% watched).
	WatchHistory(ctx context.Context, since time.Time) ([]WatchEvent, error)
}

// WatchEvent represents a single completed playback of a movie.
type WatchEvent struct {
	TMDBID    int       // movie identifier
	Title     string    // for display/logging
	WatchedAt time.Time // when playback completed
	UserName  string    // media server user (for multi-user setups)
}
