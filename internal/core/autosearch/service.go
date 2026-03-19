// Package autosearch implements on-demand automatic search: given a movie,
// search all indexers, pick the best release that satisfies the quality profile,
// and submit it to a download client. Used by both the single-movie search
// button and the bulk "Search All" action on the Wanted page.
package autosearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/luminarr/luminarr/internal/core/blocklist"
	"github.com/luminarr/luminarr/internal/core/conflict"
	"github.com/luminarr/luminarr/internal/core/customformat"
	"github.com/luminarr/luminarr/internal/core/downloader"
	"github.com/luminarr/luminarr/internal/core/edition"
	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/internal/core/tag"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// Result status constants.
const (
	StatusGrabbed = "grabbed"
	StatusNoMatch = "no_match"
	StatusSkipped = "skipped"
)

// Result describes the outcome of an auto-search for a single movie.
type Result struct {
	MovieID string    `json:"movie_id"`
	Status  string    `json:"result"` // "grabbed", "no_match", "skipped"
	Reason  string    `json:"reason,omitempty"`
	Grab    *GrabInfo `json:"grab,omitempty"`
}

// GrabInfo is the subset of grab history returned to the caller.
type GrabInfo struct {
	ID           string          `json:"id"`
	MovieID      string          `json:"movie_id"`
	ReleaseTitle string          `json:"release_title"`
	Protocol     string          `json:"protocol"`
	Size         int64           `json:"size"`
	GrabbedAt    string          `json:"grabbed_at"`
	Breakdown    json.RawMessage `json:"score_breakdown,omitempty"`
}

// BulkResult summarises a bulk auto-search operation.
type BulkResult struct {
	Searched int       `json:"searched"`
	Grabbed  int       `json:"grabbed"`
	Results  []*Result `json:"results"`
}

// MaxBulkMovies is the maximum number of movie IDs accepted in a single bulk
// search request.
const MaxBulkMovies = 100

// bulkConcurrency is the maximum number of movies searched concurrently during
// a bulk operation.
const bulkConcurrency = 2

// bulkStagger is the delay inserted between successive movie search starts
// to spread indexer load.
const bulkStagger = 2 * time.Second

// Service orchestrates automatic search and grab for movies.
type Service struct {
	indexerSvc    *indexer.Service
	movieSvc      *movie.Service
	downloaderSvc *downloader.Service
	blocklistSvc  *blocklist.Service
	qualitySvc    *quality.Service
	cfSvc         *customformat.Service
	tagSvc        *tag.Service
	bus           *events.Bus
	logger        *slog.Logger
}

// NewService creates a new auto-search Service.
func NewService(
	indexerSvc *indexer.Service,
	movieSvc *movie.Service,
	downloaderSvc *downloader.Service,
	blocklistSvc *blocklist.Service,
	qualitySvc *quality.Service,
	cfSvc *customformat.Service,
	tagSvc *tag.Service,
	bus *events.Bus,
	logger *slog.Logger,
) *Service {
	return &Service{
		indexerSvc:    indexerSvc,
		movieSvc:      movieSvc,
		downloaderSvc: downloaderSvc,
		blocklistSvc:  blocklistSvc,
		qualitySvc:    qualitySvc,
		cfSvc:         cfSvc,
		tagSvc:        tagSvc,
		bus:           bus,
		logger:        logger,
	}
}

// SearchMovie performs a full indexer search for a single movie, picks the best
// release satisfying the quality profile, and submits it to a download client.
// Works on both monitored and unmonitored movies (explicit user action).
func (s *Service) SearchMovie(ctx context.Context, movieID string) (*Result, error) {
	// 1. Fetch the movie.
	mov, err := s.movieSvc.Get(ctx, movieID)
	if err != nil {
		if errors.Is(err, movie.ErrNotFound) {
			return nil, movie.ErrNotFound
		}
		return nil, fmt.Errorf("fetching movie: %w", err)
	}

	// 2. Compute tag-filtered indexer and download client IDs.
	allowedIndexerIDs, allowedClientIDs := s.allowedEntityIDs(ctx, movieID)

	// 3. Full indexer search (filtered by tags).
	query := plugin.SearchQuery{
		TMDBID: mov.TMDBID,
		IMDBID: mov.IMDBID,
		Query:  mov.Title,
		Year:   mov.Year,
	}
	results, searchErr := s.indexerSvc.Search(ctx, query, allowedIndexerIDs)
	if len(results) == 0 {
		if searchErr != nil {
			return nil, fmt.Errorf("all indexers failed: %w", searchErr)
		}
		return &Result{
			MovieID: movieID,
			Status:  StatusNoMatch,
			Reason:  "no releases found from any indexer",
		}, nil
	}

	// 3b. Apply edition bonus and re-sort.
	// When the movie has a preferred edition, releases matching it get a
	// +30 bonus added to their effective score. This influences sort order
	// (which release gets grabbed first) without affecting whether a release
	// passes the quality profile gate.
	if mov.PreferredEdition != "" {
		for i := range results {
			bonus := edition.Bonus(mov.PreferredEdition, results[i].Edition)
			results[i].QualityScore += bonus
		}
		sort.SliceStable(results, func(i, j int) bool {
			si, sj := results[i].QualityScore, results[j].QualityScore
			if si != sj {
				return si > sj
			}
			return results[i].Seeds > results[j].Seeds
		})
	}

	// 4. Load quality profile.
	prof, err := s.qualitySvc.Get(ctx, mov.QualityProfileID)
	if err != nil {
		return nil, fmt.Errorf("loading quality profile: %w", err)
	}

	// 4b. Load custom formats and profile CF scores (non-fatal if unavailable).
	var allCFs []customformat.CustomFormat
	var profileCFScores map[string]int
	if s.cfSvc != nil {
		if cfs, cfErr := s.cfSvc.List(ctx); cfErr == nil {
			allCFs = cfs
		} else {
			s.logger.Warn("auto-search: failed to load custom formats", "error", cfErr)
		}
		if scores, scErr := s.cfSvc.ListScores(ctx, prof.ID); scErr == nil {
			profileCFScores = scores
		}
	}

	// 5. Determine current file quality and edition on disk (nil = no file).
	var currentQuality *plugin.Quality
	var currentEdition string
	if files, fErr := s.movieSvc.ListFiles(ctx, movieID); fErr == nil && len(files) > 0 {
		best := bestFileQuality(files)
		currentQuality = &best
		currentEdition = bestFileEdition(files)
	}

	// 6. Iterate candidates (sorted best→worst), try each.
	for _, r := range results {
		// Skip blocklisted releases.
		if s.blocklistSvc != nil {
			blocked, blErr := s.blocklistSvc.IsBlocklisted(ctx, r.GUID)
			if blErr != nil {
				s.logger.Warn("auto-search: blocklist check failed", "guid", r.GUID, "error", blErr)
			} else if blocked {
				continue
			}
		}

		// Evaluate custom format score for this release.
		var cfScore int
		var matchedCFIDs []string
		if len(allCFs) > 0 {
			cfRel := buildCFReleaseInfo(r.Release)
			matchedCFIDs = customformat.MatchRelease(allCFs, cfRel)
			cfScore = customformat.ScoreRelease(matchedCFIDs, profileCFScores)
		}

		// Skip releases below the profile's minimum custom format score.
		if prof.MinCustomFormatScore != 0 && cfScore < prof.MinCustomFormatScore {
			continue
		}

		// Skip releases the quality profile doesn't want — unless this is an
		// edition upgrade: the movie has a preferred edition, the current file
		// doesn't match it, and this release does. In that case the release
		// must still be in the profile's allowed quality set.
		wantByQuality := prof.WantRelease(r.Quality, currentQuality)
		wantByEdition := mov.PreferredEdition != "" &&
			!strings.EqualFold(currentEdition, mov.PreferredEdition) &&
			strings.EqualFold(r.Edition, mov.PreferredEdition)
		if !wantByQuality && !wantByEdition {
			continue
		}

		// Try submitting to a download client.
		if s.downloaderSvc == nil {
			return nil, fmt.Errorf("no download client service configured")
		}
		dcID, itemID, addErr := s.downloaderSvc.Add(ctx, r.Release, allowedClientIDs)
		if addErr != nil {
			if errors.Is(addErr, downloader.ErrNoCompatibleClient) {
				return nil, fmt.Errorf("no download client configured for protocol %s", r.Protocol)
			}
			// Download client rejected this release — auto-blocklist and try next.
			s.logger.Warn("auto-search: download client rejected release, trying next",
				"movie_id", movieID,
				"release", r.Title,
				"error", addErr,
			)
			if s.blocklistSvc != nil {
				blErr := s.blocklistSvc.Add(ctx, movieID, r.GUID, r.Title,
					r.IndexerID, string(r.Protocol), r.Size, "auto-search: download client rejected")
				if blErr != nil && !errors.Is(blErr, blocklist.ErrAlreadyBlocklisted) {
					s.logger.Warn("auto-search: failed to auto-blocklist",
						"guid", r.GUID, "error", blErr)
				}
			}
			continue
		}

		// Log any quality conflicts (warn-only — does not block the grab).
		if currentQuality != nil {
			conflicts := conflict.Compare(*currentQuality, r.Quality, currentEdition, r.Edition)
			for _, c := range conflicts {
				s.logger.Warn("auto-search: conflict detected",
					"movie_id", movieID,
					"release", r.Title,
					"conflict", c.Summary,
				)
			}
		}

		// Compute score breakdown for history.
		_, breakdown := prof.ScoreWithBreakdown(r.Quality)
		edBonus := edition.Bonus(mov.PreferredEdition, r.Edition)
		if edBonus > 0 {
			breakdown.EditionBonus = edBonus
			breakdown.Total += edBonus
			breakdown.Dimensions = append(breakdown.Dimensions, plugin.ScoreDimension{
				Name:    "edition",
				Score:   edBonus,
				Max:     edition.EditionBonus,
				Matched: true,
				Got:     r.Edition,
				Want:    mov.PreferredEdition,
			})
		}
		// Include custom format scores in the breakdown.
		if cfScore != 0 || len(matchedCFIDs) > 0 {
			breakdown.CustomFormatScore = cfScore
			breakdown.MatchedFormats = matchedCFNames(allCFs, matchedCFIDs)
		}
		breakdownJSON, _ := json.Marshal(breakdown)

		// Record grab in history. The unique partial index on grab_history
		// prevents duplicate active grabs for the same movie.
		history, grabErr := s.indexerSvc.Grab(ctx, movieID, r.IndexerID, r.Release, dcID, itemID, string(breakdownJSON))
		if grabErr != nil {
			// If this is a unique constraint violation, another grab is active.
			if isUniqueViolation(grabErr) {
				return &Result{
					MovieID: movieID,
					Status:  StatusSkipped,
					Reason:  "already downloading",
				}, nil
			}
			return nil, fmt.Errorf("recording grab history: %w", grabErr)
		}

		s.logger.Info("auto-search: grabbed release",
			"movie_id", movieID,
			"movie_title", mov.Title,
			"release", r.Title,
			"quality_score", r.QualityScore,
		)

		var bd json.RawMessage
		if len(breakdownJSON) > 0 {
			bd = breakdownJSON
		}

		return &Result{
			MovieID: movieID,
			Status:  StatusGrabbed,
			Grab: &GrabInfo{
				ID:           history.ID,
				MovieID:      history.MovieID,
				ReleaseTitle: history.ReleaseTitle,
				Protocol:     history.Protocol,
				Size:         history.Size,
				GrabbedAt:    history.GrabbedAt,
				Breakdown:    bd,
			},
		}, nil
	}

	// All candidates exhausted.
	return &Result{
		MovieID: movieID,
		Status:  StatusNoMatch,
		Reason:  "no releases satisfy quality profile",
	}, nil
}

// ExplainResult holds the dry-run decisions for all candidate releases.
type ExplainResult struct {
	MovieID     string            `json:"movie_id"`
	ProfileName string            `json:"profile_name"`
	CurrentFile *plugin.Quality   `json:"current_file,omitempty"`
	Decisions   []ReleaseDecision `json:"decisions"`
}

// SearchMovieExplain performs the same search and evaluation as SearchMovie
// but does NOT grab anything. It returns a decision for every candidate,
// explaining why each was accepted or rejected.
func (s *Service) SearchMovieExplain(ctx context.Context, movieID string) (*ExplainResult, error) {
	mov, err := s.movieSvc.Get(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("fetching movie: %w", err)
	}

	allowedIndexerIDs, _ := s.allowedEntityIDs(ctx, movieID)

	query := plugin.SearchQuery{
		TMDBID: mov.TMDBID, IMDBID: mov.IMDBID,
		Query: mov.Title, Year: mov.Year,
	}
	results, _ := s.indexerSvc.Search(ctx, query, allowedIndexerIDs)

	prof, err := s.qualitySvc.Get(ctx, mov.QualityProfileID)
	if err != nil {
		return nil, fmt.Errorf("loading quality profile: %w", err)
	}

	// Load CF data.
	var allCFs []customformat.CustomFormat
	var profileCFScores map[string]int
	if s.cfSvc != nil {
		if cfs, e := s.cfSvc.List(ctx); e == nil {
			allCFs = cfs
		}
		if sc, e := s.cfSvc.ListScores(ctx, prof.ID); e == nil {
			profileCFScores = sc
		}
	}

	var currentQuality *plugin.Quality
	var currentEdition string
	if files, fErr := s.movieSvc.ListFiles(ctx, movieID); fErr == nil && len(files) > 0 {
		best := bestFileQuality(files)
		currentQuality = &best
		currentEdition = bestFileEdition(files)
	}

	// Apply edition bonus and re-sort (same as SearchMovie).
	if mov.PreferredEdition != "" {
		for i := range results {
			bonus := edition.Bonus(mov.PreferredEdition, results[i].Edition)
			results[i].QualityScore += bonus
		}
		sort.SliceStable(results, func(i, j int) bool {
			si, sj := results[i].QualityScore, results[j].QualityScore
			if si != sj {
				return si > sj
			}
			return results[i].Seeds > results[j].Seeds
		})
	}

	var decisions []ReleaseDecision
	grabbed := false

	for _, r := range results {
		// CF evaluation.
		var cfScore int
		var matchedCFIDs []string
		if len(allCFs) > 0 {
			cfRel := buildCFReleaseInfo(r.Release)
			matchedCFIDs = customformat.MatchRelease(allCFs, cfRel)
			cfScore = customformat.ScoreRelease(matchedCFIDs, profileCFScores)
		}
		cfNames := matchedCFNames(allCFs, matchedCFIDs)

		_, breakdown := prof.ScoreWithBreakdown(r.Quality)
		breakdown.CustomFormatScore = cfScore
		breakdown.MatchedFormats = cfNames

		explainCtx := ExplainContext{
			ProfileName:    prof.Name,
			MinCFScore:     prof.MinCustomFormatScore,
			CurrentFile:    currentFileLabel(currentQuality),
			CFScore:        cfScore,
			MatchedFormats: cfNames,
			QualityName:    r.Quality.Name,
			QualityScore:   r.QualityScore,
			TotalScore:     breakdown.Total + cfScore,
		}

		mkDecision := func(reason SkipReason) ReleaseDecision {
			outcome := "skipped"
			if reason == ReasonGrabbed {
				outcome = "grabbed"
			}
			return ReleaseDecision{
				Title:          r.Title,
				GUID:           r.GUID,
				Outcome:        outcome,
				Reason:         reason,
				Explanation:    Explain(reason, explainCtx),
				QualityScore:   r.QualityScore,
				CFScore:        cfScore,
				MatchedFormats: cfNames,
				Breakdown:      &breakdown,
			}
		}

		// Blocklist.
		if s.blocklistSvc != nil {
			blocked, _ := s.blocklistSvc.IsBlocklisted(ctx, r.GUID)
			if blocked {
				decisions = append(decisions, mkDecision(ReasonBlocklisted))
				continue
			}
		}

		// CF minimum.
		if prof.MinCustomFormatScore != 0 && cfScore < prof.MinCustomFormatScore {
			decisions = append(decisions, mkDecision(ReasonCFScoreBelowMinimum))
			continue
		}

		// Quality profile.
		wantByQuality := prof.WantRelease(r.Quality, currentQuality)
		wantByEdition := mov.PreferredEdition != "" &&
			!strings.EqualFold(currentEdition, mov.PreferredEdition) &&
			strings.EqualFold(r.Edition, mov.PreferredEdition)

		if !wantByQuality && !wantByEdition {
			reason := SkipReason(prof.RejectReason(r.Quality, currentQuality))
			if reason == "" {
				reason = ReasonNoUpgradeNeeded
			}
			decisions = append(decisions, mkDecision(reason))
			continue
		}

		// Would be grabbed (first passing candidate).
		if !grabbed {
			decisions = append(decisions, mkDecision(ReasonGrabbed))
			grabbed = true
		} else {
			// Subsequent passing candidates — they would have been grabbed
			// if the first one wasn't available.
			decisions = append(decisions, mkDecision(ReasonGrabbed))
		}
	}

	return &ExplainResult{
		MovieID:     movieID,
		ProfileName: prof.Name,
		CurrentFile: currentQuality,
		Decisions:   decisions,
	}, nil
}

func currentFileLabel(q *plugin.Quality) string {
	if q == nil {
		return "(no file)"
	}
	if q.Name != "" {
		return q.Name
	}
	return string(q.Resolution) + " " + string(q.Source)
}

// SearchMovies runs SearchMovie for each movie ID with bounded concurrency.
// Progress events are published to the event bus for WebSocket delivery.
func (s *Service) SearchMovies(ctx context.Context, movieIDs []string) *BulkResult {
	bulk := &BulkResult{
		Searched: len(movieIDs),
		Results:  make([]*Result, len(movieIDs)),
	}

	var mu sync.Mutex
	sem := make(chan struct{}, bulkConcurrency)

	var wg sync.WaitGroup
	for i, id := range movieIDs {
		wg.Add(1)
		go func(idx int, movieID string) {
			defer wg.Done()

			// Acquire semaphore slot (limits concurrency).
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				mu.Lock()
				bulk.Results[idx] = &Result{
					MovieID: movieID,
					Status:  StatusSkipped,
					Reason:  "cancelled",
				}
				mu.Unlock()
				return
			}
			defer func() { <-sem }()

			// Stagger after acquiring slot to avoid burst-searching indexers.
			if idx > 0 {
				select {
				case <-ctx.Done():
					mu.Lock()
					bulk.Results[idx] = &Result{
						MovieID: movieID,
						Status:  StatusSkipped,
						Reason:  "cancelled",
					}
					mu.Unlock()
					return
				case <-time.After(bulkStagger):
				}
			}

			result, err := s.SearchMovie(ctx, movieID)
			if err != nil {
				result = &Result{
					MovieID: movieID,
					Status:  StatusNoMatch,
					Reason:  err.Error(),
				}
			}

			mu.Lock()
			bulk.Results[idx] = result
			if result.Status == StatusGrabbed {
				bulk.Grabbed++
			}
			mu.Unlock()

			// Publish progress event for WebSocket clients.
			if s.bus != nil {
				s.bus.Publish(ctx, events.Event{
					Type:    events.TypeBulkSearchProgress,
					MovieID: movieID,
					Data: map[string]any{
						"result":  result.Status,
						"reason":  result.Reason,
						"current": idx + 1,
						"total":   len(movieIDs),
					},
				})
			}
		}(i, id)
	}

	wg.Wait()

	// Publish completion event.
	if s.bus != nil {
		s.bus.Publish(ctx, events.Event{
			Type: events.TypeBulkSearchComplete,
			Data: map[string]any{
				"searched": bulk.Searched,
				"grabbed":  bulk.Grabbed,
			},
		})
	}

	return bulk
}

// bestFileQuality returns the highest-scoring quality among the given files.
func bestFileQuality(files []movie.FileInfo) plugin.Quality {
	var best plugin.Quality
	for _, f := range files {
		if f.Quality.BetterThan(best) {
			best = f.Quality
		}
	}
	return best
}

// bestFileEdition returns the edition of the highest-scoring file.
// Returns empty string when files have no edition tag.
func bestFileEdition(files []movie.FileInfo) string {
	var bestQ plugin.Quality
	var bestEdition string
	for _, f := range files {
		if f.Quality.BetterThan(bestQ) {
			bestQ = f.Quality
			bestEdition = f.Edition
		}
	}
	return bestEdition
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// buildCFReleaseInfo constructs a customformat.ReleaseInfo from a plugin.Release
// for custom format evaluation.
func buildCFReleaseInfo(r plugin.Release) customformat.ReleaseInfo {
	ri := customformat.ReleaseInfo{
		Title:         r.Title,
		Edition:       r.Edition,
		Source:        string(r.Quality.Source),
		Resolution:    string(r.Quality.Resolution),
		ReleaseGroup:  r.ReleaseGroup,
		AudioCodec:    string(r.Quality.AudioCodec),
		AudioChannels: string(r.Quality.AudioChannels),
		SizeBytes:     r.Size,
	}
	// Set modifier for remux/brdisk/rawhd sources.
	switch r.Quality.Source { //nolint:exhaustive // only modifier sources need handling
	case plugin.SourceRemux, plugin.SourceBRDisk, plugin.SourceRawHD:
		ri.Modifier = string(r.Quality.Source)
	}
	// Map indexer flags to strings.
	for _, f := range r.IndexerFlags {
		ri.IndexerFlags = append(ri.IndexerFlags, string(f))
	}
	return ri
}

// matchedCFNames returns the display names of the matched custom format IDs.
func matchedCFNames(allCFs []customformat.CustomFormat, matchedIDs []string) []string {
	if len(matchedIDs) == 0 {
		return nil
	}
	lookup := make(map[string]string, len(allCFs))
	for _, cf := range allCFs {
		lookup[cf.ID] = cf.Name
	}
	names := make([]string, 0, len(matchedIDs))
	for _, id := range matchedIDs {
		if name, ok := lookup[id]; ok {
			names = append(names, name)
		}
	}
	return names
}

// allowedEntityIDs returns the indexer and download client IDs that are allowed
// for the given movie based on tag overlap. Returns nil slices (= no filter)
// when the tag service is not configured.
func (s *Service) allowedEntityIDs(ctx context.Context, movieID string) (indexerIDs, clientIDs []string) {
	if s.tagSvc == nil {
		return nil, nil
	}
	movieTags, err := s.tagSvc.MovieTagIDs(ctx, movieID)
	if err != nil || len(movieTags) == 0 {
		// No movie tags → all entities are eligible.
		return nil, nil
	}

	// Filter indexers.
	indexerConfigs, err := s.indexerSvc.List(ctx)
	if err == nil {
		for _, cfg := range indexerConfigs {
			entityTags, _ := s.tagSvc.IndexerTagIDs(ctx, cfg.ID)
			if tag.TagsOverlap(movieTags, entityTags) {
				indexerIDs = append(indexerIDs, cfg.ID)
			}
		}
	}

	// Filter download clients.
	clientConfigs, err := s.downloaderSvc.List(ctx)
	if err == nil {
		for _, cfg := range clientConfigs {
			entityTags, _ := s.tagSvc.DownloadClientTagIDs(ctx, cfg.ID)
			if tag.TagsOverlap(movieTags, entityTags) {
				clientIDs = append(clientIDs, cfg.ID)
			}
		}
	}

	return indexerIDs, clientIDs
}
