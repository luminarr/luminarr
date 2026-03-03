// Package indexer manages indexer configurations and orchestrates release searches.
package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/davidfic/luminarr/internal/core/quality"
	dbsqlite "github.com/davidfic/luminarr/internal/db/generated/sqlite"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/pkg/plugin"
)

// ErrNotFound is returned when an indexer config does not exist.
var ErrNotFound = errors.New("indexer not found")

// Config is the domain representation of a stored indexer configuration.
type Config struct {
	ID        string
	Name      string
	Kind      string // "torznab", "newznab"
	Enabled   bool
	Priority  int
	Settings  json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateRequest carries the fields needed to create an indexer config.
type CreateRequest struct {
	Name     string
	Kind     string
	Enabled  bool
	Priority int
	Settings json.RawMessage
}

// UpdateRequest carries the fields needed to update an indexer config.
type UpdateRequest = CreateRequest

// SearchResult pairs a plugin release with its parsed quality score.
type SearchResult struct {
	plugin.Release
	// IndexerID is the DB UUID of the indexer that returned this release.
	IndexerID    string
	QualityScore int
}

// Service manages indexer configuration and search orchestration.
type Service struct {
	q   dbsqlite.Querier
	reg *registry.Registry
	bus *events.Bus
}

// NewService creates a new Service.
func NewService(q dbsqlite.Querier, reg *registry.Registry, bus *events.Bus) *Service {
	return &Service{q: q, reg: reg, bus: bus}
}

// Create persists a new indexer configuration.
func (s *Service) Create(ctx context.Context, req CreateRequest) (Config, error) {
	settings := req.Settings
	if len(settings) == 0 {
		settings = json.RawMessage("{}")
	}
	// Validate that the kind is registered.
	if _, err := s.reg.NewIndexer(req.Kind, settings); err != nil {
		return Config{}, fmt.Errorf("invalid indexer kind or settings: %w", err)
	}

	priority := req.Priority
	if priority <= 0 {
		priority = 25
	}

	now := time.Now().UTC()
	row, err := s.q.CreateIndexerConfig(ctx, dbsqlite.CreateIndexerConfigParams{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Kind:      req.Kind,
		Enabled:   boolToInt(req.Enabled),
		Priority:  int64(priority),
		Settings:  string(settings),
		CreatedAt: now.Format(time.RFC3339),
		UpdatedAt: now.Format(time.RFC3339),
	})
	if err != nil {
		return Config{}, fmt.Errorf("inserting indexer config: %w", err)
	}

	return rowToConfig(row)
}

// Get returns an indexer config by ID. Returns ErrNotFound if absent.
func (s *Service) Get(ctx context.Context, id string) (Config, error) {
	row, err := s.q.GetIndexerConfig(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Config{}, ErrNotFound
		}
		return Config{}, fmt.Errorf("fetching indexer %q: %w", id, err)
	}
	return rowToConfig(row)
}

// List returns all indexer configs, ordered by priority then name.
func (s *Service) List(ctx context.Context) ([]Config, error) {
	rows, err := s.q.ListIndexerConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing indexer configs: %w", err)
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

// Update replaces the mutable fields of an indexer config.
// Returns ErrNotFound if the indexer does not exist.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Config, error) {
	existing, err := s.q.GetIndexerConfig(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Config{}, ErrNotFound
		}
		return Config{}, fmt.Errorf("fetching indexer %q for update: %w", id, err)
	}

	// Merge: keys absent from req.Settings are preserved from existing settings.
	// This ensures secret fields (API keys) are not erased when omitted by the client.
	settings := mergeSettings(json.RawMessage(existing.Settings), req.Settings)
	if len(settings) == 0 {
		settings = json.RawMessage("{}")
	}

	priority := req.Priority
	if priority <= 0 {
		priority = 25
	}

	row, err := s.q.UpdateIndexerConfig(ctx, dbsqlite.UpdateIndexerConfigParams{
		ID:        id,
		Name:      req.Name,
		Kind:      req.Kind,
		Enabled:   boolToInt(req.Enabled),
		Priority:  int64(priority),
		Settings:  string(settings),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return Config{}, fmt.Errorf("updating indexer %q: %w", id, err)
	}
	return rowToConfig(row)
}

// Delete removes an indexer config. Returns ErrNotFound if absent.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.q.GetIndexerConfig(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("fetching indexer %q for delete: %w", id, err)
	}
	if err := s.q.DeleteIndexerConfig(ctx, id); err != nil {
		return fmt.Errorf("deleting indexer %q: %w", id, err)
	}
	return nil
}

// Test instantiates the indexer plugin and verifies connectivity.
func (s *Service) Test(ctx context.Context, id string) error {
	cfg, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	idx, err := s.reg.NewIndexer(cfg.Kind, cfg.Settings)
	if err != nil {
		return fmt.Errorf("instantiating indexer plugin: %w", err)
	}
	return idx.Test(ctx)
}

// Search queries all enabled indexers for the given search query and returns
// scored, sorted results. Results from all indexers are merged; errors from
// individual indexers are collected and returned alongside whatever results
// were gathered.
func (s *Service) Search(ctx context.Context, query plugin.SearchQuery) ([]SearchResult, error) {
	rows, err := s.q.ListEnabledIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing enabled indexers: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	type indexerResult struct {
		indexerID   string
		indexerName string
		releases    []plugin.Release
		err         error
	}

	resultsCh := make(chan indexerResult, len(rows))
	var wg sync.WaitGroup

	for _, row := range rows {
		wg.Add(1)
		go func(row dbsqlite.IndexerConfig) {
			defer wg.Done()
			cfg, _ := rowToConfig(row)
			idx, err := s.reg.NewIndexer(cfg.Kind, cfg.Settings)
			if err != nil {
				resultsCh <- indexerResult{indexerID: cfg.ID, indexerName: cfg.Name, err: err}
				return
			}
			releases, err := idx.Search(ctx, query)
			resultsCh <- indexerResult{
				indexerID:   cfg.ID,
				indexerName: cfg.Name,
				releases:    releases,
				err:         err,
			}
		}(row)
	}

	wg.Wait()
	close(resultsCh)

	var allResults []SearchResult
	var errs []error

	for res := range resultsCh {
		if res.err != nil {
			errs = append(errs, fmt.Errorf("indexer %q: %w", res.indexerName, res.err))
			continue
		}
		for _, r := range res.releases {
			// Fill in indexer name (plugin may have already set it, but ensure it).
			if r.Indexer == "" {
				r.Indexer = res.indexerName
			}
			// Parse quality from title if not set.
			if r.Quality.Source == "" || r.Quality.Source == plugin.SourceUnknown {
				if q, err := quality.Parse(r.Title); err == nil {
					r.Quality = q
				}
			}
			allResults = append(allResults, SearchResult{
				Release:      r,
				IndexerID:    res.indexerID,
				QualityScore: r.Quality.Score(),
			})
		}
	}

	// Sort by quality score descending, then by seeds descending.
	sort.Slice(allResults, func(i, j int) bool {
		si, sj := allResults[i].QualityScore, allResults[j].QualityScore
		if si != sj {
			return si > sj
		}
		return allResults[i].Seeds > allResults[j].Seeds
	})

	var combinedErr error
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		combinedErr = fmt.Errorf("%d indexer(s) failed: %v", len(errs), msgs)
	}

	return allResults, combinedErr
}

// GetRecent fetches the most recent releases from all enabled indexers and
// returns them merged and sorted by quality score descending. Errors from
// individual indexers are collected and returned alongside any results gathered.
func (s *Service) GetRecent(ctx context.Context) ([]SearchResult, error) {
	rows, err := s.q.ListEnabledIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing enabled indexers: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	type indexerResult struct {
		indexerID   string
		indexerName string
		releases    []plugin.Release
		err         error
	}

	resultsCh := make(chan indexerResult, len(rows))
	var wg sync.WaitGroup

	for _, row := range rows {
		wg.Add(1)
		go func(row dbsqlite.IndexerConfig) {
			defer wg.Done()
			cfg, _ := rowToConfig(row)
			idx, err := s.reg.NewIndexer(cfg.Kind, cfg.Settings)
			if err != nil {
				resultsCh <- indexerResult{indexerID: cfg.ID, indexerName: cfg.Name, err: err}
				return
			}
			releases, err := idx.GetRecent(ctx)
			resultsCh <- indexerResult{
				indexerID:   cfg.ID,
				indexerName: cfg.Name,
				releases:    releases,
				err:         err,
			}
		}(row)
	}

	wg.Wait()
	close(resultsCh)

	var allResults []SearchResult
	var errs []error

	for res := range resultsCh {
		if res.err != nil {
			errs = append(errs, fmt.Errorf("indexer %q: %w", res.indexerName, res.err))
			continue
		}
		for _, r := range res.releases {
			if r.Indexer == "" {
				r.Indexer = res.indexerName
			}
			if r.Quality.Source == "" || r.Quality.Source == plugin.SourceUnknown {
				if q, err := quality.Parse(r.Title); err == nil {
					r.Quality = q
				}
			}
			allResults = append(allResults, SearchResult{
				Release:      r,
				IndexerID:    res.indexerID,
				QualityScore: r.Quality.Score(),
			})
		}
	}

	sort.Slice(allResults, func(i, j int) bool {
		si, sj := allResults[i].QualityScore, allResults[j].QualityScore
		if si != sj {
			return si > sj
		}
		return allResults[i].Seeds > allResults[j].Seeds
	})

	var combinedErr error
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		combinedErr = fmt.Errorf("%d indexer(s) failed: %v", len(errs), msgs)
	}

	return allResults, combinedErr
}

// Grab records a grab to history and links it to the download client that
// accepted the release. Pass empty strings for downloadClientID / clientItemID
// when no client is involved.
func (s *Service) Grab(ctx context.Context, movieID, indexerID string, r plugin.Release, downloadClientID, clientItemID string) (dbsqlite.GrabHistory, error) {
	idxID := &indexerID
	if indexerID == "" {
		idxID = nil
	}
	var dcID *string
	if downloadClientID != "" {
		dcID = &downloadClientID
	}
	var itemID *string
	if clientItemID != "" {
		itemID = &clientItemID
	}

	now := time.Now().UTC().Format(time.RFC3339)
	row, err := s.q.CreateGrabHistory(ctx, dbsqlite.CreateGrabHistoryParams{
		ID:                uuid.New().String(),
		MovieID:           movieID,
		IndexerID:         idxID,
		ReleaseGuid:       r.GUID,
		ReleaseTitle:      r.Title,
		ReleaseSource:     string(r.Quality.Source),
		ReleaseResolution: string(r.Quality.Resolution),
		ReleaseCodec:      string(r.Quality.Codec),
		ReleaseHdr:        string(r.Quality.HDR),
		Protocol:          string(r.Protocol),
		Size:              r.Size,
		DownloadClientID:  dcID,
		ClientItemID:      itemID,
		GrabbedAt:         now,
		DownloadStatus:    "queued",
		DownloadedBytes:   0,
	})
	if err != nil {
		return dbsqlite.GrabHistory{}, fmt.Errorf("recording grab history: %w", err)
	}

	if s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type:    events.TypeGrabStarted,
			MovieID: movieID,
			Data:    map[string]any{"title": r.Title},
		})
	}

	return row, nil
}

// GrabHistory returns the grab history for a movie, newest first.
func (s *Service) GrabHistory(ctx context.Context, movieID string) ([]dbsqlite.GrabHistory, error) {
	return s.q.ListGrabHistoryByMovie(ctx, movieID)
}

func rowToConfig(row dbsqlite.IndexerConfig) (Config, error) {
	createdAt, err := time.Parse(time.RFC3339, row.CreatedAt)
	if err != nil {
		createdAt = time.Time{}
	}
	updatedAt, err := time.Parse(time.RFC3339, row.UpdatedAt)
	if err != nil {
		updatedAt = time.Time{}
	}
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

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// mergeSettings returns newSettings with any keys absent from newSettings
// filled in from existingSettings. Keys present in newSettings always win.
func mergeSettings(existing, newSettings json.RawMessage) json.RawMessage {
	if len(newSettings) == 0 {
		return existing
	}
	var existingMap, newMap map[string]json.RawMessage
	if json.Unmarshal(existing, &existingMap) != nil {
		return newSettings
	}
	if json.Unmarshal(newSettings, &newMap) != nil {
		return newSettings
	}
	for k, v := range existingMap {
		if _, ok := newMap[k]; !ok {
			newMap[k] = v
		}
	}
	merged, err := json.Marshal(newMap)
	if err != nil {
		return newSettings
	}
	return merged
}
