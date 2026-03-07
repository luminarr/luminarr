// Package queue tracks active downloads and synchronises their status with
// the download clients.
package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// Item is the queue-service view of an active download, enriching the raw
// plugin.QueueItem with grab_history context.
type Item struct {
	GrabID           string
	MovieID          string
	ReleaseTitle     string
	Protocol         string
	Size             int64
	DownloadedBytes  int64
	Status           string
	ClientItemID     string
	DownloadClientID string
	GrabbedAt        time.Time
}

// DownloaderClient is the minimal interface the queue service needs from the
// downloader service — just enough to talk to specific clients.
type DownloaderClient interface {
	ClientFor(ctx context.Context, configID string) (plugin.DownloadClient, error)
}

// Service polls download clients and keeps grab_history status up to date.
type Service struct {
	q          dbsqlite.Querier
	downloader DownloaderClient
	bus        *events.Bus
	logger     *slog.Logger
}

// NewService creates a new queue Service.
func NewService(q dbsqlite.Querier, downloader DownloaderClient, bus *events.Bus, logger *slog.Logger) *Service {
	return &Service{q: q, downloader: downloader, bus: bus, logger: logger}
}

// GetQueue returns all active downloads (those with a client_item_id that are
// not yet completed, failed, or removed) using the status cached in grab_history.
func (s *Service) GetQueue(ctx context.Context) ([]Item, error) {
	grabs, err := s.q.ListActiveGrabs(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing active grabs: %w", err)
	}

	items := make([]Item, 0, len(grabs))
	for _, g := range grabs {
		grabbedAt, _ := time.Parse(time.RFC3339, g.GrabbedAt)
		item := Item{
			GrabID:          g.ID,
			MovieID:         g.MovieID,
			ReleaseTitle:    g.ReleaseTitle,
			Protocol:        g.Protocol,
			Size:            g.Size,
			DownloadedBytes: g.DownloadedBytes,
			Status:          g.DownloadStatus,
			GrabbedAt:       grabbedAt,
		}
		if g.ClientItemID != nil {
			item.ClientItemID = *g.ClientItemID
		}
		if g.DownloadClientID != nil {
			item.DownloadClientID = *g.DownloadClientID
		}
		items = append(items, item)
	}
	return items, nil
}

// GetQueueItem returns a single active queue item by its grab ID.
// Returns an error if the item is not found.
func (s *Service) GetQueueItem(ctx context.Context, grabID string) (Item, error) {
	g, err := s.q.GetGrabByID(ctx, grabID)
	if err != nil {
		return Item{}, fmt.Errorf("grab %q not found: %w", grabID, err)
	}
	grabbedAt, _ := time.Parse(time.RFC3339, g.GrabbedAt)
	item := Item{
		GrabID:       g.ID,
		MovieID:      g.MovieID,
		ReleaseTitle: g.ReleaseTitle,
		Protocol:     g.Protocol,
		Size:         g.Size,
		Status:       g.DownloadStatus,
		GrabbedAt:    grabbedAt,
	}
	if g.ClientItemID != nil {
		item.ClientItemID = *g.ClientItemID
	}
	if g.DownloadClientID != nil {
		item.DownloadClientID = *g.DownloadClientID
	}
	return item, nil
}

// RemoveFromQueue removes a download from the client and marks the grab as removed.
// If deleteFiles is true the downloaded data is also deleted on disk.
func (s *Service) RemoveFromQueue(ctx context.Context, grabID string, deleteFiles bool) error {
	grab, err := s.q.GetGrabByID(ctx, grabID)
	if err != nil {
		return fmt.Errorf("grab %q not found: %w", grabID, err)
	}
	target := &grab
	if target.DownloadClientID == nil || target.ClientItemID == nil {
		return errors.New("grab has no associated download client")
	}

	client, err := s.downloader.ClientFor(ctx, *target.DownloadClientID)
	if err != nil {
		return fmt.Errorf("getting download client for grab: %w", err)
	}

	if err := client.Remove(ctx, *target.ClientItemID, deleteFiles); err != nil {
		return fmt.Errorf("removing from download client: %w", err)
	}

	if err := s.q.MarkGrabRemoved(ctx, grabID); err != nil {
		return fmt.Errorf("marking grab as removed: %w", err)
	}
	return nil
}

// PollAndUpdate fetches the current status of every active download from its
// download client and updates grab_history. Fires events for transitions to
// completed or failed.
func (s *Service) PollAndUpdate(ctx context.Context) error {
	grabs, err := s.q.ListActiveGrabs(ctx)
	if err != nil {
		return fmt.Errorf("listing active grabs: %w", err)
	}
	if len(grabs) == 0 {
		return nil
	}

	// Group grabs by download client to minimize API calls.
	byClient := make(map[string][]dbsqlite.GrabHistory)
	for _, g := range grabs {
		if g.DownloadClientID == nil || g.ClientItemID == nil {
			continue
		}
		byClient[*g.DownloadClientID] = append(byClient[*g.DownloadClientID], g)
	}

	for clientID, clientGrabs := range byClient {
		if err := s.pollClient(ctx, clientID, clientGrabs); err != nil {
			s.logger.Warn("queue poll failed for client",
				"client_id", clientID,
				"error", err,
			)
		}
	}
	return nil
}

func (s *Service) pollClient(ctx context.Context, clientID string, grabs []dbsqlite.GrabHistory) error {
	client, err := s.downloader.ClientFor(ctx, clientID)
	if err != nil {
		return fmt.Errorf("getting client %q: %w", clientID, err)
	}

	for _, g := range grabs {
		if g.ClientItemID == nil {
			continue
		}

		item, err := client.Status(ctx, *g.ClientItemID)
		if err != nil {
			s.logger.Debug("could not get status for item",
				"client_id", clientID,
				"client_item_id", *g.ClientItemID,
				"error", err,
			)
			continue
		}

		newStatus := string(item.Status)
		if newStatus == g.DownloadStatus && item.Downloaded == g.DownloadedBytes {
			continue // no change
		}

		if err := s.q.UpdateGrabStatus(ctx, dbsqlite.UpdateGrabStatusParams{
			ID:              g.ID,
			DownloadStatus:  newStatus,
			DownloadedBytes: item.Downloaded,
		}); err != nil {
			s.logger.Warn("failed to update grab status",
				"grab_id", g.ID,
				"error", err,
			)
			continue
		}

		// Fire events on terminal state transitions.
		if g.DownloadStatus != newStatus {
			s.fireTransitionEvent(ctx, g, newStatus, item.ContentPath)
		}
	}
	return nil
}

func (s *Service) fireTransitionEvent(ctx context.Context, g dbsqlite.GrabHistory, newStatus, contentPath string) {
	if s.bus == nil {
		return
	}
	switch plugin.DownloadStatus(newStatus) {
	case plugin.StatusCompleted:
		s.bus.Publish(ctx, events.Event{
			Type:    events.TypeDownloadDone,
			MovieID: g.MovieID,
			Data: map[string]any{
				"grab_id":      g.ID,
				"title":        g.ReleaseTitle,
				"content_path": contentPath,
			},
		})
		s.logger.Info("download completed",
			"movie_id", g.MovieID,
			"release", g.ReleaseTitle,
		)
	case plugin.StatusFailed:
		s.logger.Warn("download failed",
			"movie_id", g.MovieID,
			"release", g.ReleaseTitle,
		)
	default:
		// StatusQueued, StatusDownloading, StatusPaused — no event to fire.
	}
}
