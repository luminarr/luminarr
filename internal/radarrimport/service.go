// Package radarrimport orchestrates a one-time import from a running Radarr
// instance into Luminarr's database using the existing service layer.
package radarrimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/davidfic/luminarr/internal/core/downloader"
	"github.com/davidfic/luminarr/internal/core/indexer"
	"github.com/davidfic/luminarr/internal/core/library"
	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/core/quality"
	"github.com/davidfic/luminarr/pkg/plugin"
)

// ── Result types ──────────────────────────────────────────────────────────────

// PreviewResult summarises what would be imported from a Radarr instance.
type PreviewResult struct {
	Version         string           `json:"version"`
	MovieCount      int              `json:"movie_count"`
	QualityProfiles []ProfilePreview `json:"quality_profiles"`
	RootFolders     []FolderPreview  `json:"root_folders"`
	Indexers        []IndexerPreview `json:"indexers"`
	DownloadClients []ClientPreview  `json:"download_clients"`
}

// ProfilePreview is a summary of a Radarr quality profile.
type ProfilePreview struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// FolderPreview is a summary of a Radarr root folder.
type FolderPreview struct {
	Path        string `json:"path"`
	FreeSpaceGB int    `json:"free_space_gb"`
}

// IndexerPreview is a summary of a Radarr indexer.
type IndexerPreview struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"` // "torznab", "newznab", or "" if unsupported
}

// ClientPreview is a summary of a Radarr download client.
type ClientPreview struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"` // "qbittorrent", "deluge", or "" if unsupported
}

// ImportOptions controls which categories are imported.
type ImportOptions struct {
	QualityProfiles bool `json:"quality_profiles"`
	Libraries       bool `json:"libraries"`
	Indexers        bool `json:"indexers"`
	DownloadClients bool `json:"download_clients"`
	Movies          bool `json:"movies"`
}

// ImportResult reports how many records were created per category.
type ImportResult struct {
	QualityProfiles CategoryResult `json:"quality_profiles"`
	Libraries       CategoryResult `json:"libraries"`
	Indexers        CategoryResult `json:"indexers"`
	DownloadClients CategoryResult `json:"download_clients"`
	Movies          CategoryResult `json:"movies"`
	Errors          []string       `json:"errors"`
}

// CategoryResult holds import statistics for a single category.
type CategoryResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}

// ── Service ───────────────────────────────────────────────────────────────────

// Service orchestrates the one-time import from a Radarr instance.
type Service struct {
	movies      *movie.Service
	qualities   *quality.Service
	libraries   *library.Service
	indexers    *indexer.Service
	downloaders *downloader.Service
}

// NewService creates an import Service wired to the given core services.
func NewService(
	movies *movie.Service,
	qualities *quality.Service,
	libraries *library.Service,
	indexers *indexer.Service,
	downloaders *downloader.Service,
) *Service {
	return &Service{
		movies:      movies,
		qualities:   qualities,
		libraries:   libraries,
		indexers:    indexers,
		downloaders: downloaders,
	}
}

// Preview connects to a Radarr instance and returns a summary of what would be
// imported, without making any changes to the Luminarr database.
func (s *Service) Preview(ctx context.Context, radarrURL, apiKey string) (*PreviewResult, error) {
	c := NewClient(radarrURL, apiKey)

	status, err := c.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	profiles, err := c.GetQualityProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching quality profiles: %w", err)
	}

	folders, err := c.GetRootFolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching root folders: %w", err)
	}

	rdxIndexers, err := c.GetIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching indexers: %w", err)
	}

	rdxClients, err := c.GetDownloadClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching download clients: %w", err)
	}

	movies, err := c.GetMovies(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching movies: %w", err)
	}

	result := &PreviewResult{
		Version:         status.Version,
		MovieCount:      len(movies),
		QualityProfiles: []ProfilePreview{},
		RootFolders:     []FolderPreview{},
		Indexers:        []IndexerPreview{},
		DownloadClients: []ClientPreview{},
	}

	for _, p := range profiles {
		result.QualityProfiles = append(result.QualityProfiles, ProfilePreview{ID: p.ID, Name: p.Name})
	}
	for _, f := range folders {
		result.RootFolders = append(result.RootFolders, FolderPreview{
			Path:        f.Path,
			FreeSpaceGB: int(f.FreeSpace / (1024 * 1024 * 1024)),
		})
	}
	for _, idx := range rdxIndexers {
		result.Indexers = append(result.Indexers, IndexerPreview{
			ID:   idx.ID,
			Name: idx.Name,
			Kind: mapIndexerKind(idx.ConfigContract),
		})
	}
	for _, cl := range rdxClients {
		result.DownloadClients = append(result.DownloadClients, ClientPreview{
			ID:   cl.ID,
			Name: cl.Name,
			Kind: mapClientKind(cl.ConfigContract),
		})
	}

	return result, nil
}

// Execute imports data from Radarr into Luminarr according to opts.
// It proceeds best-effort: individual failures are collected in ImportResult.Errors.
func (s *Service) Execute(ctx context.Context, radarrURL, apiKey string, opts ImportOptions) (*ImportResult, error) {
	c := NewClient(radarrURL, apiKey)

	if _, err := c.GetStatus(ctx); err != nil {
		return nil, err
	}

	result := &ImportResult{
		Errors: []string{},
	}

	// profileIDMap: Radarr int ID → Luminarr UUID
	profileIDMap := map[int]string{}
	// libraryPathMap: Radarr root folder path → Luminarr library UUID
	libraryPathMap := map[string]string{}
	// firstProfileID: used as default for libraries that don't get a matched profile
	var firstProfileID string

	// ── 1. Quality profiles ────────────────────────────────────────────────────
	if opts.QualityProfiles {
		profiles, err := c.GetQualityProfiles(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("fetch quality profiles: %v", err))
		} else {
			for _, p := range profiles {
				req := mapProfile(p)
				created, err := s.qualities.Create(ctx, req)
				if err != nil {
					result.QualityProfiles.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("quality profile %q: %v", p.Name, err))
					continue
				}
				profileIDMap[p.ID] = created.ID
				if firstProfileID == "" {
					firstProfileID = created.ID
				}
				result.QualityProfiles.Imported++
			}
		}
	}

	// ── 2. Libraries (from root folders) ──────────────────────────────────────
	if opts.Libraries {
		folders, err := c.GetRootFolders(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("fetch root folders: %v", err))
		} else {
			for _, f := range folders {
				name := filepath.Base(strings.TrimRight(f.Path, "/\\"))
				if name == "" || name == "." {
					name = f.Path
				}
				req := library.CreateRequest{
					Name:                    name,
					RootPath:                f.Path,
					DefaultQualityProfileID: firstProfileID,
					MinFreeSpaceGB:          0,
					Tags:                    []string{},
				}
				created, err := s.libraries.Create(ctx, req)
				if err != nil {
					result.Libraries.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("library %q: %v", f.Path, err))
					continue
				}
				libraryPathMap[f.Path] = created.ID
				result.Libraries.Imported++
			}
		}
	}

	// ── 3. Indexers ────────────────────────────────────────────────────────────
	if opts.Indexers {
		rdxIndexers, err := c.GetIndexers(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("fetch indexers: %v", err))
		} else {
			for _, idx := range rdxIndexers {
				kind := mapIndexerKind(idx.ConfigContract)
				if kind == "" {
					result.Indexers.Skipped++
					continue
				}
				settings := map[string]string{
					"url":     fieldString(idx.Fields, "baseUrl"),
					"api_key": fieldString(idx.Fields, "apiKey"),
				}
				settingsJSON, _ := json.Marshal(settings)
				req := indexer.CreateRequest{
					Name:     idx.Name,
					Kind:     kind,
					Enabled:  idx.EnableRss,
					Priority: 25,
					Settings: settingsJSON,
				}
				if _, err := s.indexers.Create(ctx, req); err != nil {
					result.Indexers.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("indexer %q: %v", idx.Name, err))
					continue
				}
				result.Indexers.Imported++
			}
		}
	}

	// ── 4. Download clients ────────────────────────────────────────────────────
	if opts.DownloadClients {
		rdxClients, err := c.GetDownloadClients(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("fetch download clients: %v", err))
		} else {
			for _, cl := range rdxClients {
				kind := mapClientKind(cl.ConfigContract)
				if kind == "" {
					result.DownloadClients.Skipped++
					continue
				}
				settingsJSON, err := buildClientSettings(kind, cl.Fields)
				if err != nil {
					result.DownloadClients.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("download client %q settings: %v", cl.Name, err))
					continue
				}
				req := downloader.CreateRequest{
					Name:     cl.Name,
					Kind:     kind,
					Enabled:  cl.Enable,
					Priority: 25,
					Settings: settingsJSON,
				}
				if _, err := s.downloaders.Create(ctx, req); err != nil {
					result.DownloadClients.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("download client %q: %v", cl.Name, err))
					continue
				}
				result.DownloadClients.Imported++
			}
		}
	}

	// ── 5. Movies ──────────────────────────────────────────────────────────────
	if opts.Movies {
		rdxMovies, err := c.GetMovies(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("fetch movies: %v", err))
		} else {
			for _, m := range rdxMovies {
				if m.TmdbID == 0 {
					result.Movies.Skipped++
					continue
				}
				// Map Radarr quality profile ID → Luminarr UUID.
				profileID := profileIDMap[m.QualityProfileID]
				if profileID == "" {
					profileID = firstProfileID
				}
				// Map Radarr root folder path → Luminarr library UUID.
				libID := libraryPathMap[m.RootFolderPath]
				if libID == "" {
					// Fall back to any available library.
					for _, id := range libraryPathMap {
						libID = id
						break
					}
				}
				if libID == "" {
					result.Movies.Skipped++
					result.Errors = append(result.Errors, fmt.Sprintf("movie %q (tmdb:%d): no library available", m.Title, m.TmdbID))
					continue
				}
				req := movie.AddRequest{
					TMDBID:           m.TmdbID,
					LibraryID:        libID,
					QualityProfileID: profileID,
					Monitored:        m.Monitored,
				}
				if _, err := s.movies.Add(ctx, req); err != nil {
					if errors.Is(err, movie.ErrAlreadyExists) {
						result.Movies.Skipped++
					} else {
						result.Movies.Failed++
						result.Errors = append(result.Errors, fmt.Sprintf("movie %q (tmdb:%d): %v", m.Title, m.TmdbID, err))
					}
					continue
				}
				result.Movies.Imported++
			}
		}
	}

	return result, nil
}

// ── Mapping helpers ────────────────────────────────────────────────────────────

// mapIndexerKind maps a Radarr configContract to a Luminarr indexer kind.
// Returns "" for unsupported contracts.
func mapIndexerKind(contract string) string {
	switch contract {
	case "NewznabSettings":
		return "newznab"
	case "TorznabSettings":
		return "torznab"
	default:
		return ""
	}
}

// mapClientKind maps a Radarr configContract to a Luminarr downloader kind.
// Returns "" for unsupported contracts.
func mapClientKind(contract string) string {
	switch contract {
	case "QBittorrentSettings":
		return "qbittorrent"
	case "DelugeSettings":
		return "deluge"
	default:
		return ""
	}
}

// buildClientSettings builds the settings JSON for a download client from
// Radarr's field list.
func buildClientSettings(kind string, fields []radarrField) (json.RawMessage, error) {
	host := fieldString(fields, "host")
	port := fieldInt(fields, "port")
	useSsl := fieldBool(fields, "useSsl")
	url := buildURL(host, port, useSsl, fieldString(fields, "urlBase"))

	var settings map[string]string
	switch kind {
	case "qbittorrent":
		settings = map[string]string{
			"url":      url,
			"username": fieldString(fields, "username"),
			"password": fieldString(fields, "password"),
			"category": fieldString(fields, "movieCategory"),
		}
	case "deluge":
		settings = map[string]string{
			"url":      url,
			"password": fieldString(fields, "password"),
			"label":    fieldString(fields, "movieCategory"),
		}
	default:
		return nil, fmt.Errorf("unsupported client kind %q", kind)
	}
	return json.Marshal(settings)
}

// cutoffName finds the quality name matching the given cutoff ID by searching
// the profile items recursively. Returns "" if not found.
func cutoffName(items []radarrProfileItem, cutoffID int) string {
	for _, item := range items {
		if len(item.Items) == 0 {
			if item.Quality.ID == cutoffID {
				return item.Quality.Name
			}
		} else {
			for _, child := range item.Items {
				if child.Quality.ID == cutoffID {
					return child.Quality.Name
				}
			}
		}
	}
	return ""
}

// mapProfile converts a Radarr quality profile into a Luminarr CreateRequest.
func mapProfile(p radarrProfile) quality.CreateRequest {
	var qualities []plugin.Quality
	for _, item := range p.Items {
		for _, q := range item.qualities() {
			qualities = append(qualities, mapRadarrQuality(q.Name))
		}
	}
	cutoff := mapRadarrQuality(cutoffName(p.Items, p.Cutoff))
	return quality.CreateRequest{
		Name:           p.Name,
		Cutoff:         cutoff,
		Qualities:      qualities,
		UpgradeAllowed: p.UpgradeAllowed,
	}
}

// mapRadarrQuality translates a Radarr quality name into a Luminarr Quality.
// Codec defaults to "unknown" and HDR defaults to "none" since Radarr encodes
// those via Custom Formats, not quality profiles.
func mapRadarrQuality(name string) plugin.Quality {
	type entry struct {
		resolution plugin.Resolution
		source     plugin.Source
	}

	table := map[string]entry{
		"SDTV":               {plugin.ResolutionSD, plugin.SourceHDTV},
		"DVDR":               {plugin.ResolutionSD, plugin.SourceDVD},
		"DVD":                {plugin.ResolutionSD, plugin.SourceDVD},
		"HDTV-720p":          {plugin.Resolution720p, plugin.SourceHDTV},
		"HDTV-1080p":         {plugin.Resolution1080p, plugin.SourceHDTV},
		"WEBRip-480p":        {plugin.ResolutionSD, plugin.SourceWEBRip},
		"WEBRip-720p":        {plugin.Resolution720p, plugin.SourceWEBRip},
		"WEBRip-1080p":       {plugin.Resolution1080p, plugin.SourceWEBRip},
		"WEBRip-2160p":       {plugin.Resolution2160p, plugin.SourceWEBRip},
		"WEBDL-480p":         {plugin.ResolutionSD, plugin.SourceWEBDL},
		"WEBDL-720p":         {plugin.Resolution720p, plugin.SourceWEBDL},
		"WEBDL-1080p":        {plugin.Resolution1080p, plugin.SourceWEBDL},
		"WEBDL-2160p":        {plugin.Resolution2160p, plugin.SourceWEBDL},
		"Bluray-480p":        {plugin.ResolutionSD, plugin.SourceBluRay},
		"Bluray-720p":        {plugin.Resolution720p, plugin.SourceBluRay},
		"Bluray-1080p":       {plugin.Resolution1080p, plugin.SourceBluRay},
		"Bluray-2160p":       {plugin.Resolution2160p, plugin.SourceBluRay},
		"Bluray-720p Remux":  {plugin.Resolution720p, plugin.SourceRemux},
		"Bluray-1080p Remux": {plugin.Resolution1080p, plugin.SourceRemux},
		"Bluray-2160p Remux": {plugin.Resolution2160p, plugin.SourceRemux},
		"Raw-HD":             {plugin.Resolution1080p, plugin.SourceRemux},
	}

	e, ok := table[name]
	if !ok {
		return plugin.Quality{
			Resolution: plugin.ResolutionUnknown,
			Source:     plugin.SourceUnknown,
			Codec:      plugin.CodecUnknown,
			HDR:        plugin.HDRNone,
			Name:       name,
		}
	}
	return plugin.Quality{
		Resolution: e.resolution,
		Source:     e.source,
		Codec:      plugin.CodecUnknown,
		HDR:        plugin.HDRNone,
		Name:       name,
	}
}
