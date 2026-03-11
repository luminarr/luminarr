// Package seedenforcer applies per-indexer seed limits to torrents after import.
// It subscribes to TypeImportComplete events and, when the source indexer has
// seed_ratio or seed_time_minutes configured, tells the download client to
// enforce those limits on the torrent.
package seedenforcer

import (
	"context"
	"log/slog"

	"github.com/luminarr/luminarr/internal/core/indexer"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// SeedCriteriaProvider loads seed criteria for an indexer by ID.
type SeedCriteriaProvider interface {
	GetSeedCriteria(ctx context.Context, indexerID string) (indexer.SeedCriteria, error)
}

// ClientProvider instantiates a download client by config ID.
type ClientProvider interface {
	ClientFor(ctx context.Context, configID string) (plugin.DownloadClient, error)
}

// Service listens for import-complete events and applies per-indexer seed limits.
type Service struct {
	q          dbsqlite.Querier
	indexerSvc SeedCriteriaProvider
	dlSvc      ClientProvider
	bus        *events.Bus
	logger     *slog.Logger
}

// NewService creates a new seed enforcer service.
func NewService(q dbsqlite.Querier, indexerSvc SeedCriteriaProvider, dlSvc ClientProvider, bus *events.Bus, logger *slog.Logger) *Service {
	return &Service{q: q, indexerSvc: indexerSvc, dlSvc: dlSvc, bus: bus, logger: logger}
}

// Subscribe registers the event handler on the bus.
func (s *Service) Subscribe() {
	s.bus.Subscribe(func(ctx context.Context, e events.Event) {
		if e.Type != events.TypeImportComplete {
			return
		}
		s.handle(ctx, e)
	})
}

func (s *Service) handle(ctx context.Context, e events.Event) {
	grabID, _ := e.Data["grab_id"].(string)
	if grabID == "" {
		return
	}

	grab, err := s.q.GetGrabByID(ctx, grabID)
	if err != nil {
		s.logger.Warn("seedenforcer: could not load grab", "grab_id", grabID, "error", err)
		return
	}

	// Only applies to torrent protocol.
	if grab.Protocol != string(plugin.ProtocolTorrent) {
		return
	}

	// Need indexer_id, download_client_id, and client_item_id.
	if grab.IndexerID == nil || grab.DownloadClientID == nil || grab.ClientItemID == nil {
		return
	}

	criteria, err := s.indexerSvc.GetSeedCriteria(ctx, *grab.IndexerID)
	if err != nil {
		s.logger.Debug("seedenforcer: could not load indexer seed criteria",
			"indexer_id", *grab.IndexerID, "error", err)
		return
	}

	// Nothing to enforce if both are zero (use client defaults).
	if criteria.SeedRatio <= 0 && criteria.SeedTimeMinutes <= 0 {
		return
	}

	client, err := s.dlSvc.ClientFor(ctx, *grab.DownloadClientID)
	if err != nil {
		s.logger.Warn("seedenforcer: could not get download client",
			"client_id", *grab.DownloadClientID, "error", err)
		return
	}

	limiter, ok := client.(plugin.SeedLimiter)
	if !ok {
		s.logger.Debug("seedenforcer: download client does not support seed limits",
			"client_id", *grab.DownloadClientID)
		return
	}

	seedTimeSecs := criteria.SeedTimeMinutes * 60
	if err := limiter.SetSeedLimits(ctx, *grab.ClientItemID, criteria.SeedRatio, seedTimeSecs); err != nil {
		s.logger.Warn("seedenforcer: SetSeedLimits failed",
			"client_item_id", *grab.ClientItemID,
			"ratio", criteria.SeedRatio,
			"time_minutes", criteria.SeedTimeMinutes,
			"error", err,
		)
		return
	}

	s.logger.Info("seedenforcer: seed limits applied",
		"client_item_id", *grab.ClientItemID,
		"ratio", criteria.SeedRatio,
		"time_minutes", criteria.SeedTimeMinutes,
	)
}
