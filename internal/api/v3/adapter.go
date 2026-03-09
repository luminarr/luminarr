// Package v3 provides a Radarr v3 API compatibility layer.
//
// External tools like Overseerr, Jellyseerr, Homepage, and others have a
// built-in "Radarr" integration. This package exposes the subset of the Radarr
// v3 API surface that those tools actually call, translating between Radarr's
// response shapes and Luminarr's existing service layer.
package v3

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/luminarr/luminarr/internal/core/library"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/internal/metadata/tmdb"
)

// ── ROWID helpers ────────────────────────────────────────────────────────────
//
// sqlc doesn't understand SQLite's implicit ROWID column, so these use raw
// database/sql. Each table with a TEXT PRIMARY KEY still has a stable integer
// ROWID that we surface as the Radarr-compatible integer ID.

// rowIDMap builds a bidirectional UUID ↔ ROWID mapping for a table.
type rowIDMap struct {
	uuidToRow map[string]int64
	rowToUUID map[int64]string
}

func buildRowIDMap(ctx context.Context, db *sql.DB, table string) (rowIDMap, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT rowid, id FROM %s", table)) //nolint:gosec // table name is a constant from internal callers
	if err != nil {
		return rowIDMap{}, err
	}
	defer rows.Close()

	m := rowIDMap{
		uuidToRow: make(map[string]int64),
		rowToUUID: make(map[int64]string),
	}
	for rows.Next() {
		var rowid int64
		var uuid string
		if err := rows.Scan(&rowid, &uuid); err != nil {
			return rowIDMap{}, err
		}
		m.uuidToRow[uuid] = rowid
		m.rowToUUID[rowid] = uuid
	}
	return m, rows.Err()
}

// getUUIDByRowID returns the UUID for a given ROWID, or "" if not found.
func getUUIDByRowID(ctx context.Context, db *sql.DB, table string, rowid int64) (string, error) {
	var uuid string
	err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT id FROM %s WHERE rowid = ?", table), rowid).Scan(&uuid) //nolint:gosec
	if err == sql.ErrNoRows {
		return "", nil
	}
	return uuid, err
}

// getRowIDByUUID returns the ROWID for a given UUID, or 0 if not found.
func getRowIDByUUID(ctx context.Context, db *sql.DB, table string, uuid string) (int64, error) {
	var rowid int64
	err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT rowid FROM %s WHERE id = ?", table), uuid).Scan(&rowid) //nolint:gosec
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return rowid, err
}

// ── Radarr v3 response types ────────────────────────────────────────────────

// RadarrMovie is the Radarr v3 movie response shape.
type RadarrMovie struct {
	ID                  int64         `json:"id"`
	Title               string        `json:"title"`
	OriginalTitle       string        `json:"originalTitle"`
	SortTitle           string        `json:"sortTitle"`
	TmdbID              int           `json:"tmdbId"`
	ImdbID              string        `json:"imdbId"`
	Year                int           `json:"year"`
	Path                string        `json:"path"`
	RootFolderPath      string        `json:"rootFolderPath"`
	QualityProfileID    int64         `json:"qualityProfileId"`
	Monitored           bool          `json:"monitored"`
	MinimumAvailability string        `json:"minimumAvailability"`
	IsAvailable         bool          `json:"isAvailable"`
	HasFile             bool          `json:"hasFile"`
	SizeOnDisk          int64         `json:"sizeOnDisk"`
	Status              string        `json:"status"`
	Overview            string        `json:"overview"`
	Images              []RadarrImage `json:"images"`
	Genres              []string      `json:"genres"`
	Tags                []int64       `json:"tags"`
	Added               string        `json:"added"`
	Ratings             struct{}      `json:"ratings"`
	TitleSlug           string        `json:"titleSlug"`
	Runtime             int           `json:"runtime"`
	FolderName          string        `json:"folderName,omitempty"`
}

// RadarrImage is a poster/fanart image entry.
type RadarrImage struct {
	CoverType string `json:"coverType"`
	RemoteURL string `json:"remoteUrl"`
}

// RadarrQualityProfile is the Radarr v3 quality profile response shape.
type RadarrQualityProfile struct {
	ID             int64                      `json:"id"`
	Name           string                     `json:"name"`
	UpgradeAllowed bool                       `json:"upgradeAllowed"`
	Cutoff         int                        `json:"cutoff"`
	Items          []RadarrQualityProfileItem `json:"items"`
}

// RadarrQualityProfileItem is a single allowed quality in a profile.
type RadarrQualityProfileItem struct {
	Quality RadarrQualityRef `json:"quality"`
	Allowed bool             `json:"allowed"`
}

// RadarrQualityRef is a quality ID + name pair.
type RadarrQualityRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// RadarrRootFolder is the Radarr v3 root folder response.
type RadarrRootFolder struct {
	ID         int64  `json:"id"`
	Path       string `json:"path"`
	Accessible bool   `json:"accessible"`
	FreeSpace  int64  `json:"freeSpace"`
}

// RadarrTag is the Radarr v3 tag response.
type RadarrTag struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
}

// RadarrSystemStatus is the Radarr v3 system/status response.
type RadarrSystemStatus struct {
	AppName        string `json:"appName"`
	InstanceName   string `json:"instanceName"`
	Version        string `json:"version"`
	Branch         string `json:"branch"`
	Authentication string `json:"authentication"`
	URLBase        string `json:"urlBase"`
	RuntimeName    string `json:"runtimeName"`
	RuntimeVersion string `json:"runtimeVersion"`
	StartupPath    string `json:"startupPath"`
	AppData        string `json:"appData"`
	IsDocker       bool   `json:"isDocker"`
	IsLinux        bool   `json:"isLinux"`
	IsWindows      bool   `json:"isWindows"`
	IsOsx          bool   `json:"isOsx"`
}

// RadarrQueueStatus is the Radarr v3 queue/status response.
type RadarrQueueStatus struct {
	TotalCount           int  `json:"totalCount"`
	Count                int  `json:"count"`
	UnknownCount         int  `json:"unknownCount"`
	Errors               bool `json:"errors"`
	Warnings             bool `json:"warnings"`
	UnknownErrors        bool `json:"unknownErrors"`
	UnknownWarnings      bool `json:"unknownWarnings"`
	HasEnabledDownloader bool `json:"hasEnabledDownloader"` //nolint:revive
}

// RadarrCommand is the request body for POST /api/v3/command.
type RadarrCommand struct {
	Name     string  `json:"name"`
	MovieIDs []int64 `json:"movieIds,omitempty"`
}

// RadarrCommandResponse is the response for POST /api/v3/command.
type RadarrCommandResponse struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Started string `json:"started"`
}

// ── Conversion functions ────────────────────────────────────────────────────

const tmdbImageBase = "https://image.tmdb.org/t/p/original"

func movieToRadarr(m movie.Movie, rowid int64, files []movie.FileInfo, libRootPath string, qpRowID int64) RadarrMovie {
	var sizeOnDisk int64
	for _, f := range files {
		sizeOnDisk += f.SizeBytes
	}

	images := make([]RadarrImage, 0, 2)
	if m.PosterURL != "" {
		url := m.PosterURL
		if !strings.HasPrefix(url, "http") {
			url = tmdbImageBase + url
		}
		images = append(images, RadarrImage{CoverType: "poster", RemoteURL: url})
	}
	if m.FanartURL != "" {
		url := m.FanartURL
		if !strings.HasPrefix(url, "http") {
			url = tmdbImageBase + url
		}
		images = append(images, RadarrImage{CoverType: "fanart", RemoteURL: url})
	}

	return RadarrMovie{
		ID:                  rowid,
		Title:               m.Title,
		OriginalTitle:       m.OriginalTitle,
		SortTitle:           strings.ToLower(m.Title),
		TmdbID:              m.TMDBID,
		ImdbID:              m.IMDBID,
		Year:                m.Year,
		Path:                m.Path,
		RootFolderPath:      libRootPath,
		QualityProfileID:    qpRowID,
		Monitored:           m.Monitored,
		MinimumAvailability: m.MinimumAvailability,
		IsAvailable:         m.Status == "released" || m.Status == "downloaded",
		HasFile:             len(files) > 0,
		SizeOnDisk:          sizeOnDisk,
		Status:              m.Status,
		Overview:            m.Overview,
		Images:              images,
		Genres:              m.Genres,
		Tags:                []int64{},
		Added:               m.AddedAt.Format(time.RFC3339),
		TitleSlug:           fmt.Sprintf("%s-%d", slugify(m.Title), m.TMDBID),
		Runtime:             m.RuntimeMinutes,
	}
}

func qualityProfileToRadarr(p quality.Profile, rowid int64) RadarrQualityProfile {
	items := make([]RadarrQualityProfileItem, len(p.Qualities))
	for i, q := range p.Qualities {
		items[i] = RadarrQualityProfileItem{
			Quality: RadarrQualityRef{
				ID:   q.Score(),
				Name: q.Name,
			},
			Allowed: true,
		}
	}
	return RadarrQualityProfile{
		ID:             rowid,
		Name:           p.Name,
		UpgradeAllowed: p.UpgradeAllowed,
		Cutoff:         p.Cutoff.Score(),
		Items:          items,
	}
}

func libraryToRadarrRootFolder(lib library.Library, rowid int64, freeSpace int64) RadarrRootFolder {
	return RadarrRootFolder{
		ID:         rowid,
		Path:       lib.RootPath,
		Accessible: true,
		FreeSpace:  freeSpace,
	}
}

func tmdbResultToRadarrMovie(r tmdb.SearchResult) RadarrMovie {
	images := make([]RadarrImage, 0, 2)
	if r.PosterPath != "" {
		images = append(images, RadarrImage{CoverType: "poster", RemoteURL: tmdbImageBase + r.PosterPath})
	}
	if r.BackdropPath != "" {
		images = append(images, RadarrImage{CoverType: "fanart", RemoteURL: tmdbImageBase + r.BackdropPath})
	}
	return RadarrMovie{
		TmdbID:              r.ID,
		Title:               r.Title,
		OriginalTitle:       r.OriginalTitle,
		SortTitle:           strings.ToLower(r.Title),
		Year:                r.Year,
		Overview:            r.Overview,
		Images:              images,
		Tags:                []int64{},
		Genres:              []string{},
		Status:              "released",
		MinimumAvailability: "released",
		TitleSlug:           fmt.Sprintf("%s-%d", slugify(r.Title), r.ID),
	}
}

// slugify produces a URL-safe slug from a title.
func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == ' ' || r == '-' {
			return '-'
		}
		return -1
	}, s)
	// collapse repeated dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}
