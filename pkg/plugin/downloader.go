package plugin

import "context"

// DownloadStatus is the state of an item in the download client.
type DownloadStatus string

const (
	StatusQueued      DownloadStatus = "queued"
	StatusDownloading DownloadStatus = "downloading"
	StatusCompleted   DownloadStatus = "completed"
	StatusPaused      DownloadStatus = "paused"
	StatusFailed      DownloadStatus = "failed"
)

// QueueItem represents an item tracked in the download client.
type QueueItem struct {
	ClientItemID string
	Title        string
	Status       DownloadStatus
	Size         int64
	Downloaded   int64
	SeedRatio    float64 // torrent only; 0 for NZB
	Error        string
	// ContentPath is the absolute filesystem path to the downloaded content.
	// For single-file downloads this is the file path; for multi-file downloads
	// it is the root directory. Empty until the download client reports it.
	ContentPath string
	// AddedAt is the Unix timestamp when the item was added to the client.
	// Zero if the client does not report this field.
	AddedAt int64
}

// DownloadClient is the plugin interface for download clients.
type DownloadClient interface {
	Name() string
	Protocol() Protocol

	// Add submits a release to the download client.
	// Returns the client-assigned item ID for future status queries.
	Add(ctx context.Context, r Release) (clientItemID string, err error)

	// Status returns the current state of a download client item.
	Status(ctx context.Context, clientItemID string) (QueueItem, error)

	// GetQueue returns all items currently in the download client.
	GetQueue(ctx context.Context) ([]QueueItem, error)

	// Remove deletes an item from the download client.
	// If deleteFiles is true, the downloaded data is also deleted.
	Remove(ctx context.Context, clientItemID string, deleteFiles bool) error

	// Test validates that the connection to the download client works.
	Test(ctx context.Context) error
}

// SeedLimiter is an optional interface for download clients that support
// configuring per-torrent seed ratio and seed time limits.
// Implementations should treat ratioLimit <= 0 as "use client default"
// and seedTimeSecs <= 0 as "no time limit".
type SeedLimiter interface {
	SetSeedLimits(ctx context.Context, clientItemID string, ratioLimit float64, seedTimeSecs int) error
}
