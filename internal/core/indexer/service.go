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

	"github.com/luminarr/luminarr/internal/core/dbutil"
	"github.com/luminarr/luminarr/internal/core/quality"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/ratelimit"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/pkg/plugin"
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
	IndexerID      string
	QualityScore   int
	ScoreBreakdown plugin.ScoreBreakdown
}

// Service manages indexer configuration and search orchestration.
type Service struct {
	q     dbsqlite.Querier
	reg   *registry.Registry
	bus   *events.Bus
	rl    *ratelimit.Registry
	cache sync.Map // config ID → plugin.Indexer
}

// NewService creates a new Service.
func NewService(q dbsqlite.Querier, reg *registry.Registry, bus *events.Bus, rl *ratelimit.Registry) *Service {
	return &Service{q: q, reg: reg, bus: bus, rl: rl}
}

// cachedIndexer returns a cached or freshly-created indexer for the given config.
func (s *Service) cachedIndexer(kind, id string, settings json.RawMessage) (plugin.Indexer, error) {
	if v, ok := s.cache.Load(id); ok {
		return v.(plugin.Indexer), nil
	}
	idx, err := s.reg.NewIndexer(kind, settings)
	if err != nil {
		return nil, err
	}
	actual, _ := s.cache.LoadOrStore(id, idx)
	return actual.(plugin.Indexer), nil
}

// evictIndexer removes a cached indexer instance.
func (s *Service) evictIndexer(id string) {
	s.cache.Delete(id)
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
		Enabled:   dbutil.BoolToInt(req.Enabled),
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
	settings := dbutil.MergeSettings(json.RawMessage(existing.Settings), req.Settings)
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
		Enabled:   dbutil.BoolToInt(req.Enabled),
		Priority:  int64(priority),
		Settings:  string(settings),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return Config{}, fmt.Errorf("updating indexer %q: %w", id, err)
	}
	s.evictIndexer(id)
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
	s.rl.Remove(id)
	s.evictIndexer(id)
	return nil
}

// Test instantiates the indexer plugin and verifies connectivity.
func (s *Service) Test(ctx context.Context, id string) error {
	cfg, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.rl.Wait(ctx, cfg.ID, extractRateLimit(cfg.Settings)); err != nil {
		return fmt.Errorf("rate limit wait: %w", err)
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
			if err := s.rl.Wait(ctx, cfg.ID, extractRateLimit(cfg.Settings)); err != nil {
				resultsCh <- indexerResult{indexerID: cfg.ID, indexerName: cfg.Name, err: err}
				return
			}
			idx, err := s.cachedIndexer(cfg.Kind, cfg.ID, cfg.Settings)
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
			// Prefer the per-item indexer name from Prowlarr/Jackett; fall back
			// to the user-configured name from the database.
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
			if err := s.rl.Wait(ctx, cfg.ID, extractRateLimit(cfg.Settings)); err != nil {
				resultsCh <- indexerResult{indexerID: cfg.ID, indexerName: cfg.Name, err: err}
				return
			}
			idx, err := s.cachedIndexer(cfg.Kind, cfg.ID, cfg.Settings)
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
// when no client is involved. scoreBreakdownJSON is a pre-serialized
// plugin.ScoreBreakdown; pass "" when not available.
func (s *Service) Grab(ctx context.Context, movieID, indexerID string, r plugin.Release, downloadClientID, clientItemID, scoreBreakdownJSON string) (dbsqlite.GrabHistory, error) {
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
		ScoreBreakdown:    scoreBreakdownJSON,
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

// ListHistory returns the most recent grab history entries across all movies,
// optionally filtered by download status and/or protocol.
func (s *Service) ListHistory(ctx context.Context, limit int, status, protocol string) ([]dbsqlite.GrabHistory, error) {
	switch {
	case status != "" && protocol != "":
		return s.q.ListGrabHistoryByStatusAndProtocol(ctx, dbsqlite.ListGrabHistoryByStatusAndProtocolParams{
			DownloadStatus: status,
			Protocol:       protocol,
			Limit:          int64(limit),
		})
	case status != "":
		return s.q.ListGrabHistoryByStatus(ctx, dbsqlite.ListGrabHistoryByStatusParams{
			DownloadStatus: status,
			Limit:          int64(limit),
		})
	case protocol != "":
		return s.q.ListGrabHistoryByProtocol(ctx, dbsqlite.ListGrabHistoryByProtocolParams{
			Protocol: protocol,
			Limit:    int64(limit),
		})
	default:
		return s.q.ListGrabHistory(ctx, int64(limit))
	}
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

// extractRateLimit reads the rate_limit field from an indexer's settings JSON.
// Returns 0 (unlimited) if the field is absent or unparseable.
func extractRateLimit(settings json.RawMessage) int {
	var s struct {
		RateLimit int `json:"rate_limit"`
	}
	_ = json.Unmarshal(settings, &s)
	return s.RateLimit
}

// SeedCriteria holds the per-indexer seeding requirements applied to torrents
// after import. Zero values mean "use download client default".
type SeedCriteria struct {
	SeedRatio       float64 // 0 = use client default
	SeedTimeMinutes int     // 0 = no limit
}

// GetSeedCriteria loads the seed criteria from an indexer's settings JSON.
func (s *Service) GetSeedCriteria(ctx context.Context, indexerID string) (SeedCriteria, error) {
	cfg, err := s.Get(ctx, indexerID)
	if err != nil {
		return SeedCriteria{}, err
	}
	return ExtractSeedCriteria(cfg.Settings), nil
}

// ExtractSeedCriteria parses seed_ratio and seed_time_minutes from an indexer's
// settings JSON. Returns zero values if the fields are absent.
func ExtractSeedCriteria(settings json.RawMessage) SeedCriteria {
	var s struct {
		SeedRatio       float64 `json:"seed_ratio"`
		SeedTimeMinutes int     `json:"seed_time_minutes"`
	}
	_ = json.Unmarshal(settings, &s)
	return SeedCriteria{SeedRatio: s.SeedRatio, SeedTimeMinutes: s.SeedTimeMinutes}
}
