// Package plexsync provides bidirectional library comparison between Luminarr
// and a configured Plex media server. It identifies movies that exist in one
// system but not the other, and supports importing Plex-only movies into Luminarr.
package plexsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/davidfic/luminarr/internal/core/mediaserver"
	"github.com/davidfic/luminarr/internal/core/movie"
	dbsqlite "github.com/davidfic/luminarr/internal/db/generated/sqlite"
	plexpkg "github.com/davidfic/luminarr/plugins/mediaservers/plex"
)

// ── Result types ─────────────────────────────────────────────────────────────

// SyncMovie is a movie found only in Plex (not in Luminarr).
type SyncMovie struct {
	Title  string `json:"title"`
	Year   int    `json:"year"`
	TmdbID int    `json:"tmdb_id"`
}

// LuminarrMovie is a movie found only in Luminarr (not in Plex).
type LuminarrMovie struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Year   int    `json:"year"`
	TmdbID int    `json:"tmdb_id"`
	Status string `json:"status"`
}

// SyncPreview summarises the library diff between Plex and Luminarr.
type SyncPreview struct {
	PlexTotal      int             `json:"plex_total"`
	InPlexOnly     []SyncMovie     `json:"in_plex_only"`
	InLuminarrOnly []LuminarrMovie `json:"in_luminarr_only"`
	AlreadySynced  int             `json:"already_synced"`
	Unmatched      int             `json:"unmatched"`
}

// SyncImportOptions controls what gets imported from Plex.
type SyncImportOptions struct {
	TmdbIDs          []int  `json:"tmdb_ids"`
	LibraryID        string `json:"library_id"`
	QualityProfileID string `json:"quality_profile_id"`
	Monitored        bool   `json:"monitored"`
}

// SyncImportResult reports what happened during import.
type SyncImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors"`
}

// ── Service ──────────────────────────────────────────────────────────────────

// Service orchestrates the bidirectional Plex library sync.
type Service struct {
	ms     *mediaserver.Service
	movies *movie.Service
	q      dbsqlite.Querier
}

// NewService creates a new plexsync Service.
func NewService(ms *mediaserver.Service, movies *movie.Service, q dbsqlite.Querier) *Service {
	return &Service{ms: ms, movies: movies, q: q}
}

// Sections returns the movie library sections from a Plex media server.
func (s *Service) Sections(ctx context.Context, mediaServerID string) ([]plexpkg.Section, error) {
	srv, err := s.instantiatePlex(ctx, mediaServerID)
	if err != nil {
		return nil, err
	}
	return srv.ListSections(ctx)
}

// Preview fetches movies from a Plex section and compares against Luminarr's library.
func (s *Service) Preview(ctx context.Context, mediaServerID, sectionKey string) (*SyncPreview, error) {
	srv, err := s.instantiatePlex(ctx, mediaServerID)
	if err != nil {
		return nil, err
	}

	plexMovies, err := srv.ListMovies(ctx, sectionKey)
	if err != nil {
		return nil, fmt.Errorf("listing plex movies: %w", err)
	}

	// Build set of Luminarr TMDB IDs.
	summaries, err := s.q.ListMovieSummaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing luminarr movies: %w", err)
	}
	luminarrByTmdb := make(map[int]dbsqlite.ListMovieSummariesRow, len(summaries))
	for _, m := range summaries {
		luminarrByTmdb[int(m.TmdbID)] = m
	}

	// Build set of Plex TMDB IDs.
	plexByTmdb := make(map[int]plexpkg.Movie, len(plexMovies))
	var unmatched int
	for _, pm := range plexMovies {
		if pm.TmdbID == 0 {
			unmatched++
			continue
		}
		plexByTmdb[pm.TmdbID] = pm
	}

	// Compute diff.
	var inPlexOnly []SyncMovie
	var alreadySynced int
	for tmdbID, pm := range plexByTmdb {
		if _, inLuminarr := luminarrByTmdb[tmdbID]; inLuminarr {
			alreadySynced++
		} else {
			inPlexOnly = append(inPlexOnly, SyncMovie{
				Title:  pm.Title,
				Year:   pm.Year,
				TmdbID: pm.TmdbID,
			})
		}
	}

	var inLuminarrOnly []LuminarrMovie
	for tmdbID, lm := range luminarrByTmdb {
		if _, inPlex := plexByTmdb[tmdbID]; !inPlex {
			inLuminarrOnly = append(inLuminarrOnly, LuminarrMovie{
				ID:     lm.ID,
				Title:  lm.Title,
				Year:   int(lm.Year),
				TmdbID: int(lm.TmdbID),
				Status: lm.Status,
			})
		}
	}

	return &SyncPreview{
		PlexTotal:      len(plexMovies),
		InPlexOnly:     inPlexOnly,
		InLuminarrOnly: inLuminarrOnly,
		AlreadySynced:  alreadySynced,
		Unmatched:      unmatched,
	}, nil
}

// Import adds selected Plex movies to Luminarr.
func (s *Service) Import(ctx context.Context, opts SyncImportOptions) (*SyncImportResult, error) {
	result := &SyncImportResult{Errors: []string{}}
	for _, tmdbID := range opts.TmdbIDs {
		req := movie.AddRequest{
			TMDBID:           tmdbID,
			LibraryID:        opts.LibraryID,
			QualityProfileID: opts.QualityProfileID,
			Monitored:        opts.Monitored,
		}
		if _, err := s.movies.Add(ctx, req); err != nil {
			if errors.Is(err, movie.ErrAlreadyExists) {
				result.Skipped++
			} else {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("tmdb:%d: %v", tmdbID, err))
			}
			continue
		}
		result.Imported++
	}
	return result, nil
}

// instantiatePlex loads a media server config, verifies it's Plex, and returns
// a *plex.Server ready for API calls.
func (s *Service) instantiatePlex(ctx context.Context, mediaServerID string) (*plexpkg.Server, error) {
	cfg, err := s.ms.Get(ctx, mediaServerID)
	if err != nil {
		return nil, err
	}
	if cfg.Kind != "plex" {
		return nil, fmt.Errorf("media server %q is %s, not plex", mediaServerID, cfg.Kind)
	}
	var plexCfg plexpkg.Config
	if err := json.Unmarshal(cfg.Settings, &plexCfg); err != nil {
		return nil, fmt.Errorf("parsing plex settings: %w", err)
	}
	return plexpkg.New(plexCfg), nil
}
