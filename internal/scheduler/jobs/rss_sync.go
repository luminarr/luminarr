package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/quality"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/scheduler"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// RSSSync returns a Job that polls all enabled indexers for recent releases,
// matches them against monitored movies, and automatically grabs releases that
// satisfy each movie's quality profile. Runs every 15 minutes.
func RSSSync(
	idxSvc *indexer.Service,
	dlSvc *downloader.Service,
	qualSvc *quality.Service,
	q dbsqlite.Querier,
	logger *slog.Logger,
) scheduler.Job {
	return scheduler.Job{
		Name:     "rss_sync",
		Interval: 15 * time.Minute,
		Fn: func(ctx context.Context) {
			logger.Info("task started", "task", "rss_sync")
			start := time.Now()

			grabbed, err := runRSSSync(ctx, idxSvc, dlSvc, qualSvc, q, logger)
			if err != nil {
				logger.Warn("task failed",
					"task", "rss_sync",
					"error", err,
					"duration_ms", time.Since(start).Milliseconds(),
				)
				return
			}

			logger.Info("task finished",
				"task", "rss_sync",
				"grabbed", grabbed,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		},
	}
}

func runRSSSync(
	ctx context.Context,
	idxSvc *indexer.Service,
	dlSvc *downloader.Service,
	qualSvc *quality.Service,
	q dbsqlite.Querier,
	logger *slog.Logger,
) (int, error) {
	// 1. Fetch recent releases from all enabled indexers.
	releases, fetchErr := idxSvc.GetRecent(ctx)
	if fetchErr != nil {
		// Non-fatal: partial results from other indexers may still be useful.
		logger.Warn("some indexers failed during RSS fetch", "error", fetchErr)
	}
	if len(releases) == 0 {
		return 0, nil
	}

	// 2. List all monitored movies.
	movies, err := q.ListMonitoredMovies(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing monitored movies: %w", err)
	}
	if len(movies) == 0 {
		return 0, nil
	}

	// 3. Build a set of movie IDs that already have an active grab so we
	//    don't queue duplicate downloads.
	activeGrabs, err := q.ListActiveGrabs(ctx)
	if err != nil {
		return 0, fmt.Errorf("listing active grabs: %w", err)
	}
	activeMovies := make(map[string]bool, len(activeGrabs))
	for _, g := range activeGrabs {
		activeMovies[g.MovieID] = true
	}

	// 4. Process each monitored movie.
	var grabbed int
	for _, m := range movies {
		if activeMovies[m.ID] {
			continue // already downloading something for this movie
		}

		// Skip movies that haven't reached their minimum availability threshold.
		if !movieEligibleForGrab(m.MinimumAvailability, m.Status) {
			continue
		}

		// Load the quality profile.
		prof, err := qualSvc.Get(ctx, m.QualityProfileID)
		if err != nil {
			logger.Warn("skipping movie: could not load quality profile",
				"movie_id", m.ID,
				"profile_id", m.QualityProfileID,
				"error", err,
			)
			continue
		}

		// Determine current file quality (nil = no file on disk).
		var currentQuality *plugin.Quality
		if files, _ := q.ListMovieFiles(ctx, m.ID); len(files) > 0 {
			best := bestFileQuality(files)
			currentQuality = &best
		}

		// Find the best matching, wanted release.
		best, ok := bestMatchingRelease(releases, m.Title, int(m.Year), prof, currentQuality)
		if !ok {
			continue
		}

		// Submit to a download client.
		dcID, itemID, err := dlSvc.Add(ctx, best.Release, nil)
		if err != nil {
			if errors.Is(err, downloader.ErrNoCompatibleClient) {
				logger.Warn("no download client configured for protocol",
					"movie_id", m.ID,
					"protocol", best.Protocol,
				)
			} else {
				logger.Warn("could not submit release to download client",
					"movie_id", m.ID,
					"release", best.Title,
					"error", err,
				)
			}
			continue
		}

		// Compute score breakdown for history storage.
		_, breakdown := prof.ScoreWithBreakdown(best.Release.Quality)
		breakdownJSON, _ := json.Marshal(breakdown)

		// Record the grab in history.
		if _, err := idxSvc.Grab(ctx, m.ID, best.IndexerID, best.Release, dcID, itemID, string(breakdownJSON)); err != nil {
			logger.Warn("could not record grab history",
				"movie_id", m.ID,
				"release", best.Title,
				"error", err,
			)
			continue
		}

		logger.Info("auto-grabbed release",
			"movie_id", m.ID,
			"movie_title", m.Title,
			"release", best.Title,
			"quality_score", best.QualityScore,
		)
		grabbed++
		activeMovies[m.ID] = true // prevent a second grab if movie appears again
	}

	return grabbed, nil
}

// bestMatchingRelease returns the first release from rs (ordered best→worst
// quality) that matches the movie by title+year and is wanted by the profile.
func bestMatchingRelease(
	rs []indexer.SearchResult,
	title string,
	year int,
	prof quality.Profile,
	currentQuality *plugin.Quality,
) (indexer.SearchResult, bool) {
	for _, r := range rs {
		if !releaseMatchesMovie(r.Title, title, year) {
			continue
		}
		if prof.WantRelease(r.Quality, currentQuality) {
			return r, true
		}
	}
	return indexer.SearchResult{}, false
}

// bestFileQuality returns the highest-scoring quality among the given files.
func bestFileQuality(files []dbsqlite.MovieFile) plugin.Quality {
	var best plugin.Quality
	for _, f := range files {
		var q plugin.Quality
		if err := json.Unmarshal([]byte(f.QualityJson), &q); err != nil {
			continue
		}
		if q.BetterThan(best) {
			best = q
		}
	}
	return best
}

// releaseMatchesMovie reports whether a release title is a plausible match for
// a movie. The normalised movie title must appear as a word-boundary-aligned
// substring and the release year must also appear in the normalised release.
func releaseMatchesMovie(releaseTitle, movieTitle string, year int) bool {
	normRelease := normalizeTitle(releaseTitle)
	normMovie := normalizeTitle(movieTitle)
	if normMovie == "" {
		return false
	}
	if !containsWordAligned(normRelease, normMovie) {
		return false
	}
	return containsWordAligned(normRelease, strconv.Itoa(year))
}

// containsWordAligned reports whether haystack contains needle aligned on word
// (space) boundaries. This prevents a movie titled "it" from matching every
// release that incidentally contains the substring "it".
func containsWordAligned(haystack, needle string) bool {
	idx := 0
	for {
		pos := strings.Index(haystack[idx:], needle)
		if pos < 0 {
			return false
		}
		abs := idx + pos
		atStart := abs == 0 || haystack[abs-1] == ' '
		end := abs + len(needle)
		atEnd := end == len(haystack) || haystack[end] == ' '
		if atStart && atEnd {
			return true
		}
		idx = abs + 1
	}
}

// movieEligibleForGrab reports whether a movie's TMDB status has reached the
// user-configured minimum availability threshold. The four thresholds map to
// TMDB status strings as follows:
//
//	tba / announced → always eligible (grab as soon as monitored)
//	in_cinemas      → "In Production", "Post Production", or "Released"
//	released        → "Released" only
func movieEligibleForGrab(minAvail, tmdbStatus string) bool {
	switch minAvail {
	case "tba", "announced", "":
		return true
	case "in_cinemas":
		switch tmdbStatus {
		case "In Production", "Post Production", "Released":
			return true
		}
		return false
	case "released":
		return tmdbStatus == "Released"
	default:
		return true
	}
}

// normalizeTitle lowercases a string, converts common separators (dots,
// underscores, hyphens) to spaces, strips other non-alphanumeric characters,
// and collapses whitespace.
func normalizeTitle(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '.' || r == '_' || r == '-':
			b.WriteRune(' ')
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ':
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
