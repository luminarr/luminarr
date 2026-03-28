// Package watchsync ingests watch history from media servers and stores it
// locally for watch-aware library management.
package watchsync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/luminarr/luminarr/internal/core/mediaserver"
	"github.com/luminarr/luminarr/internal/core/movie"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// WatchStatus is the derived watch state for a single movie.
type WatchStatus struct {
	Watched       bool    `json:"watched"`
	PlayCount     int64   `json:"play_count"`
	LastWatchedAt *string `json:"last_watched_at,omitempty"`
}

// WatchStats is aggregate watch statistics.
type WatchStats struct {
	WatchedCount int64   `json:"watched_count"`
	TotalCount   int64   `json:"total_count"`
	Percentage   float64 `json:"percentage"`
}

// Service manages watch history sync from media servers.
type Service struct {
	q        dbsqlite.Querier
	msSvc    *mediaserver.Service
	movieSvc *movie.Service
	reg      *registry.Registry
	logger   *slog.Logger
}

// NewService creates a new watch sync service.
func NewService(q dbsqlite.Querier, msSvc *mediaserver.Service, movieSvc *movie.Service, reg *registry.Registry, logger *slog.Logger) *Service {
	return &Service{q: q, msSvc: msSvc, movieSvc: movieSvc, reg: reg, logger: logger}
}

// Sync polls all configured media servers for new watch events and stores them.
func (s *Service) Sync(ctx context.Context) error {
	servers, err := s.msSvc.List(ctx)
	if err != nil {
		return fmt.Errorf("listing media servers: %w", err)
	}

	var syncErrors []error
	for _, cfg := range servers {
		ms, err := s.reg.NewMediaServer(cfg.Kind, cfg.Settings)
		if err != nil {
			s.logger.Warn("failed to instantiate media server for watch sync", "id", cfg.ID, "kind", cfg.Kind, "error", err)
			continue
		}

		wp, ok := ms.(plugin.WatchProvider)
		if !ok {
			continue // plugin doesn't support watch history
		}

		since := s.getLastSync(ctx, cfg.ID)
		events, err := wp.WatchHistory(ctx, since)
		if err != nil {
			s.logger.Warn("watch history fetch failed", "id", cfg.ID, "kind", cfg.Kind, "error", err)
			syncErrors = append(syncErrors, err)
			continue
		}

		inserted := 0
		for _, e := range events {
			// Match TMDB ID to a movie in our library.
			m, err := s.movieSvc.GetByTMDBID(ctx, e.TMDBID)
			if err != nil {
				continue // not in library — skip
			}

			err = s.q.InsertWatchEvent(ctx, dbsqlite.InsertWatchEventParams{
				ID:        uuid.New().String(),
				MovieID:   m.ID,
				TmdbID:    int64(e.TMDBID),
				WatchedAt: e.WatchedAt.UTC().Format(time.RFC3339),
				UserName:  e.UserName,
				Source:    cfg.Kind,
			})
			if err != nil {
				continue // duplicate or DB error — skip silently
			}
			inserted++
		}

		// Update last sync time only on success.
		now := time.Now().UTC().Format(time.RFC3339)
		_ = s.q.UpsertSyncState(ctx, dbsqlite.UpsertSyncStateParams{
			MediaServerID: cfg.ID,
			LastSyncAt:    now,
		})

		s.logger.Info("watch sync completed", "server", cfg.Kind, "events", len(events), "inserted", inserted)
	}

	if len(syncErrors) > 0 {
		return fmt.Errorf("watch sync had %d error(s)", len(syncErrors))
	}
	return nil
}

// getLastSync returns the last sync time for a media server, or epoch if never synced.
func (s *Service) getLastSync(ctx context.Context, mediaServerID string) time.Time {
	ts, err := s.q.GetSyncState(ctx, mediaServerID)
	if err != nil || ts == "" {
		return time.Time{} // epoch — fetch all history
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return t
}

// WatchStatusForMovie returns the watch status for a single movie.
func (s *Service) WatchStatusForMovie(ctx context.Context, movieID string) (*WatchStatus, error) {
	row, err := s.q.WatchStatusForMovie(ctx, movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &WatchStatus{}, nil
		}
		return nil, err
	}

	ws := &WatchStatus{
		Watched:   row.PlayCount > 0,
		PlayCount: row.PlayCount,
	}
	if s, ok := row.LastWatchedAt.(string); ok && s != "" {
		ws.LastWatchedAt = &s
	}
	return ws, nil
}

// WatchStatusBatch returns watch status for all movies that have been watched.
func (s *Service) WatchStatusBatch(ctx context.Context) (map[string]*WatchStatus, error) {
	rows, err := s.q.WatchStatusBatch(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*WatchStatus, len(rows))
	for _, r := range rows {
		ws := &WatchStatus{
			Watched:   true,
			PlayCount: r.PlayCount,
		}
		if s, ok := r.LastWatchedAt.(string); ok && s != "" {
			ws.LastWatchedAt = &s
		}
		result[r.MovieID] = ws
	}
	return result, nil
}

// Stats returns aggregate watch statistics.
func (s *Service) Stats(ctx context.Context) (*WatchStats, error) {
	row, err := s.q.WatchStats(ctx)
	if err != nil {
		return nil, err
	}

	pct := 0.0
	if row.TotalCount > 0 {
		pct = float64(row.WatchedCount) / float64(row.TotalCount) * 100
	}

	return &WatchStats{
		WatchedCount: row.WatchedCount,
		TotalCount:   row.TotalCount,
		Percentage:   pct,
	}, nil
}
