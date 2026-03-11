// Package downloader manages download client configurations and orchestrates
// adding releases to the appropriate download client.
package downloader

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/luminarr/luminarr/internal/core/dbutil"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// ErrNotFound is returned when a download client config does not exist.
var ErrNotFound = errors.New("download client not found")

// ErrNoCompatibleClient is returned when no enabled client supports the release protocol.
var ErrNoCompatibleClient = errors.New("no enabled download client configured for this protocol")

// Config is the domain representation of a stored download client configuration.
type Config struct {
	ID        string
	Name      string
	Kind      string // "qbittorrent", "transmission", etc.
	Enabled   bool
	Priority  int
	Settings  json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateRequest carries the fields needed to create a download client config.
type CreateRequest struct {
	Name     string
	Kind     string
	Enabled  bool
	Priority int
	Settings json.RawMessage
}

// UpdateRequest carries the fields needed to update a download client config.
type UpdateRequest = CreateRequest

// Service manages download client configuration and release submission.
type Service struct {
	q     dbsqlite.Querier
	reg   *registry.Registry
	bus   *events.Bus
	cache sync.Map // config ID → plugin.DownloadClient
}

// NewService creates a new Service.
func NewService(q dbsqlite.Querier, reg *registry.Registry, bus *events.Bus) *Service {
	return &Service{q: q, reg: reg, bus: bus}
}

// clientFor returns a cached or freshly-created download client for the given
// config. Cached clients persist across calls, avoiding repeated auth round-trips.
func (s *Service) cachedClient(kind string, id string, settings json.RawMessage) (plugin.DownloadClient, error) {
	if v, ok := s.cache.Load(id); ok {
		return v.(plugin.DownloadClient), nil
	}
	client, err := s.reg.NewDownloader(kind, settings)
	if err != nil {
		return nil, err
	}
	// Store returns the existing value if another goroutine raced us — use that.
	actual, _ := s.cache.LoadOrStore(id, client)
	return actual.(plugin.DownloadClient), nil
}

// evictClient removes a cached client instance, forcing re-creation on next use.
func (s *Service) evictClient(id string) {
	s.cache.Delete(id)
}

// Create persists a new download client configuration.
func (s *Service) Create(ctx context.Context, req CreateRequest) (Config, error) {
	settings := req.Settings
	if len(settings) == 0 {
		settings = json.RawMessage("{}")
	}
	if _, err := s.reg.NewDownloader(req.Kind, settings); err != nil {
		return Config{}, fmt.Errorf("invalid downloader kind or settings: %w", err)
	}

	priority := req.Priority
	if priority <= 0 {
		priority = 25
	}

	now := time.Now().UTC()
	row, err := s.q.CreateDownloadClientConfig(ctx, dbsqlite.CreateDownloadClientConfigParams{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Kind:      req.Kind,
		Enabled:   dbutil.BoolToInt(req.Enabled),
		Priority:  int64(priority),
		Settings:  string(settings),
		CreatedAt: now.Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	})
	if err != nil {
		return Config{}, fmt.Errorf("inserting download client config: %w", err)
	}
	return rowToConfig(row)
}

// Get returns a download client config by ID. Returns ErrNotFound if absent.
func (s *Service) Get(ctx context.Context, id string) (Config, error) {
	row, err := s.q.GetDownloadClientConfig(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Config{}, ErrNotFound
		}
		return Config{}, fmt.Errorf("fetching download client %q: %w", id, err)
	}
	return rowToConfig(row)
}

// List returns all download client configs ordered by priority then name.
func (s *Service) List(ctx context.Context) ([]Config, error) {
	rows, err := s.q.ListDownloadClientConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing download client configs: %w", err)
	}
	configs := make([]Config, 0, len(rows))
	for _, row := range rows {
		cfg, err := rowToConfig(row)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

// Update replaces the mutable fields of a download client config.
// Returns ErrNotFound if the config does not exist.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Config, error) {
	existing, err := s.q.GetDownloadClientConfig(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Config{}, ErrNotFound
		}
		return Config{}, fmt.Errorf("fetching download client %q for update: %w", id, err)
	}

	// Merge: keys absent from req.Settings are preserved from existing settings.
	// This ensures secret fields (passwords) are not erased when omitted by the client.
	settings := dbutil.MergeSettings(json.RawMessage(existing.Settings), req.Settings)
	if len(settings) == 0 {
		settings = json.RawMessage("{}")
	}

	priority := req.Priority
	if priority <= 0 {
		priority = 25
	}

	row, err := s.q.UpdateDownloadClientConfig(ctx, dbsqlite.UpdateDownloadClientConfigParams{
		ID:        id,
		Name:      req.Name,
		Kind:      req.Kind,
		Enabled:   dbutil.BoolToInt(req.Enabled),
		Priority:  int64(priority),
		Settings:  string(settings),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return Config{}, fmt.Errorf("updating download client %q: %w", id, err)
	}
	s.evictClient(id)
	return rowToConfig(row)
}

// Delete removes a download client config. Returns ErrNotFound if absent.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.q.GetDownloadClientConfig(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("fetching download client %q for delete: %w", id, err)
	}
	if err := s.q.DeleteDownloadClientConfig(ctx, id); err != nil {
		return fmt.Errorf("deleting download client %q: %w", id, err)
	}
	s.evictClient(id)
	return nil
}

// Test instantiates the plugin and verifies connectivity to the download client.
func (s *Service) Test(ctx context.Context, id string) error {
	cfg, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	client, err := s.reg.NewDownloader(cfg.Kind, cfg.Settings)
	if err != nil {
		return fmt.Errorf("instantiating downloader plugin: %w", err)
	}
	return client.Test(ctx)
}

// Add finds the first enabled download client that supports the release's
// protocol and submits the release. Returns the download client config ID
// and the client-assigned item ID.
func (s *Service) Add(ctx context.Context, r plugin.Release) (downloadClientID, clientItemID string, err error) {
	rows, err := s.q.ListEnabledDownloadClients(ctx)
	if err != nil {
		return "", "", fmt.Errorf("listing enabled download clients: %w", err)
	}

	for _, row := range rows {
		cfg, err := rowToConfig(row)
		if err != nil {
			return "", "", fmt.Errorf("parsing download client config %q: %w", row.ID, err)
		}
		client, err := s.cachedClient(cfg.Kind, cfg.ID, cfg.Settings)
		if err != nil {
			return "", "", fmt.Errorf("initialising download client %q (%s): %w", cfg.Name, cfg.Kind, err)
		}
		if client.Protocol() != r.Protocol {
			continue
		}

		itemID, err := client.Add(ctx, r)
		if err != nil {
			return "", "", fmt.Errorf("adding release to %q (%s): %w", cfg.Name, cfg.Kind, err)
		}

		if s.bus != nil {
			s.bus.Publish(ctx, events.Event{
				Type:    events.TypeGrabStarted,
				MovieID: "",
				Data: map[string]any{
					"client":         cfg.Name,
					"client_item_id": itemID,
				},
			})
		}

		return cfg.ID, itemID, nil
	}

	return "", "", ErrNoCompatibleClient
}

// ClientFor returns a (cached) plugin.DownloadClient for the given config ID.
// Used by the queue service to communicate with a specific client.
func (s *Service) ClientFor(ctx context.Context, configID string) (plugin.DownloadClient, error) {
	cfg, err := s.Get(ctx, configID)
	if err != nil {
		return nil, err
	}
	return s.cachedClient(cfg.Kind, cfg.ID, cfg.Settings)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func rowToConfig(row dbsqlite.DownloadClientConfig) (Config, error) {
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)
	return Config{
		ID:        row.ID,
		Name:      row.Name,
		Kind:      row.Kind,
		Enabled:   row.Enabled != 0,
		Priority:  int(row.Priority),
		Settings:  json.RawMessage(row.Settings),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}
