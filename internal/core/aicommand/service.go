// Package aicommand provides AI-powered command interpretation for the command palette.
package aicommand

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/luminarr/luminarr/internal/anthropic"
	"github.com/luminarr/luminarr/internal/core/autosearch"
	"github.com/luminarr/luminarr/internal/core/library"
	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/quality"
	"github.com/luminarr/luminarr/internal/core/stats"
)

// Service orchestrates AI command processing.
type Service struct {
	movieSvc     *movie.Service
	statsSvc     *stats.Service
	autoSvc      *autosearch.Service
	librarySvc   *library.Service
	qualitySvc   *quality.Service
	logger       *slog.Logger
	pendingStore *PendingStore

	// mu guards client and rate-limit state.
	mu        sync.Mutex
	client    *anthropic.Client
	tokens    int
	maxTokens int
	lastReset time.Time
}

// NewService creates a new AI command service. The client may be nil; use
// SetClient to configure it later (e.g. when the user sets an API key at
// runtime).
func NewService(client *anthropic.Client, movieSvc *movie.Service, statsSvc *stats.Service, autoSvc *autosearch.Service, librarySvc *library.Service, qualitySvc *quality.Service, logger *slog.Logger) *Service {
	return &Service{
		client:       client,
		movieSvc:     movieSvc,
		statsSvc:     statsSvc,
		autoSvc:      autoSvc,
		librarySvc:   librarySvc,
		qualitySvc:   qualitySvc,
		logger:       logger,
		pendingStore: NewPendingStore(),
		tokens:       10,
		maxTokens:    10,
		lastReset:    time.Now(),
	}
}

// SetClient replaces the Anthropic client at runtime (hot-reload when the
// user saves an API key via the settings UI).
func (s *Service) SetClient(c *anthropic.Client) {
	s.mu.Lock()
	s.client = c
	s.mu.Unlock()
}

// Enabled returns true when a Claude API client is configured.
func (s *Service) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client != nil
}

// ErrRateLimited is returned when the rate limit is exceeded.
var ErrRateLimited = fmt.Errorf("rate limit exceeded, try again in a moment")

// ErrNotConfigured is returned when no API key is set.
var ErrNotConfigured = fmt.Errorf("AI is not configured — add a Claude API key in Settings > App")

// allowRequest checks that the client is configured and implements a simple
// token bucket (10 RPM). Returns the client to use for the request.
func (s *Service) allowRequest() (*anthropic.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return nil, ErrNotConfigured
	}

	now := time.Now()
	elapsed := now.Sub(s.lastReset)

	// Refill tokens based on elapsed time (10 per minute).
	refill := int(elapsed.Seconds() / 6) // 1 token per 6 seconds
	if refill > 0 {
		s.tokens += refill
		if s.tokens > s.maxTokens {
			s.tokens = s.maxTokens
		}
		s.lastReset = now
	}

	if s.tokens <= 0 {
		return nil, ErrRateLimited
	}
	s.tokens--
	return s.client, nil
}

// ProcessCommand interprets a natural language command and returns a structured action.
func (s *Service) ProcessCommand(ctx context.Context, text string) (*CommandResponse, error) {
	client, err := s.allowRequest()
	if err != nil {
		return nil, err
	}

	systemPrompt := s.buildSystemPrompt(ctx)

	messages := []anthropic.Message{
		{Role: "user", Content: text},
	}

	resp, err := client.CreateMessage(ctx, systemPrompt, messages, 1024)
	if err != nil {
		s.logger.Error("AI command request failed", "error", err)
		return &CommandResponse{
			Action:      ActionFallback,
			Explanation: "Sorry, I couldn't process that request. Please try again.",
		}, nil
	}

	result, err := s.parseResponse(resp.Text())
	if err != nil {
		s.logger.Warn("failed to parse AI response", "raw", resp.Text(), "error", err)
		return &CommandResponse{
			Action:      ActionFallback,
			Explanation: "I had trouble understanding that. Could you rephrase your request?",
		}, nil
	}

	// For state-modifying actions, resolve the movie and store for confirmation.
	if result.Action.RequiresConfirmation() {
		var prepErr error
		result, prepErr = s.prepareConfirmation(ctx, result)
		if prepErr != nil {
			return &CommandResponse{ //nolint:nilerr // graceful degradation — show error as fallback message
				Action:      ActionFallback,
				Explanation: prepErr.Error(),
			}, nil
		}
	}

	return result, nil
}

// prepareConfirmation resolves references (e.g. movie title → ID) and stores
// the action for later confirmation.
func (s *Service) prepareConfirmation(ctx context.Context, resp *CommandResponse) (*CommandResponse, error) {
	switch resp.Action { //nolint:exhaustive // only state-modifying actions reach here
	case ActionAutoSearch:
		return s.prepareAutoSearch(ctx, resp)
	case ActionRunTask:
		// Pass through — task name is in params, no resolution needed.
	}

	id := s.pendingStore.Add(resp.Action, resp.Params)
	resp.RequiresConfirm = true
	resp.PendingActionID = id
	return resp, nil
}

// prepareAutoSearch resolves a movie title via TMDB. If the movie is already
// in the library, the pending action searches immediately on confirm. If not,
// confirming adds it to the library first, then searches.
func (s *Service) prepareAutoSearch(ctx context.Context, resp *CommandResponse) (*CommandResponse, error) {
	query, _ := resp.Params["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("I need a movie title to search for")
	}

	results, err := s.movieSvc.Lookup(ctx, movie.LookupRequest{Query: query})
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("I couldn't find a movie matching %q on TMDB", query)
	}

	qualityPref, _ := resp.Params["quality"].(string)
	tmdbResult := results[0]

	params := map[string]any{
		"tmdb_id":     tmdbResult.ID,
		"movie_title": tmdbResult.Title,
	}
	if qualityPref != "" {
		params["quality"] = qualityPref
	}

	// Check if the movie is already in the library.
	libraryMovie, err := s.movieSvc.GetByTMDBID(ctx, tmdbResult.ID)
	if err == nil {
		// Already in library — just need to search.
		params["movie_id"] = libraryMovie.ID
		params["needs_add"] = false
	} else {
		// Not in library — will add on confirm.
		params["needs_add"] = true
	}

	id := s.pendingStore.Add(ActionAutoSearch, params)
	resp.Params = params
	resp.RequiresConfirm = true
	resp.PendingActionID = id

	qualityDesc := "the best available release"
	if qualityPref != "" {
		qualityDesc = "a " + qualityPref + " release"
	}

	if params["needs_add"] == true {
		resp.ConfirmationMessage = fmt.Sprintf("Add %q to your library and search for %s?", tmdbResult.Title, qualityDesc)
	} else {
		resp.ConfirmationMessage = fmt.Sprintf("Search for %s of %q?", qualityDesc, tmdbResult.Title)
	}
	return resp, nil
}

// ConfirmAction executes a previously stored pending action.
func (s *Service) ConfirmAction(ctx context.Context, actionID string) (*CommandResponse, error) {
	pa := s.pendingStore.Take(actionID)
	if pa == nil {
		return nil, fmt.Errorf("action not found or expired — please try again")
	}

	switch pa.Action {
	case ActionAutoSearch:
		return s.executeAutoSearch(ctx, pa)
	case ActionRunTask:
		// RunTask is handled by the API layer (scheduler), not here.
		return &CommandResponse{
			Action:      ActionRunTask,
			Params:      pa.Params,
			Explanation: "Task execution confirmed.",
		}, nil
	default:
		return nil, fmt.Errorf("unknown action type for confirmation: %q", pa.Action)
	}
}

func (s *Service) executeAutoSearch(ctx context.Context, pa *PendingAction) (*CommandResponse, error) {
	movieTitle, _ := pa.Params["movie_title"].(string)
	movieID, _ := pa.Params["movie_id"].(string)
	needsAdd, _ := pa.Params["needs_add"].(bool)

	if s.autoSvc == nil {
		return nil, fmt.Errorf("automatic search is not available — check that indexers and download clients are configured")
	}

	// If the movie isn't in the library yet, add it first.
	if needsAdd {
		tmdbID, _ := pa.Params["tmdb_id"].(float64) // JSON numbers are float64
		if tmdbID == 0 {
			return nil, fmt.Errorf("missing TMDB ID")
		}

		// Use the first available library and quality profile.
		libID, profID, err := s.resolveDefaults(ctx)
		if err != nil {
			return nil, err
		}

		added, err := s.movieSvc.Add(ctx, movie.AddRequest{
			TMDBID:           int(tmdbID),
			LibraryID:        libID,
			QualityProfileID: profID,
			Monitored:        true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add %q to library: %w", movieTitle, err)
		}
		movieID = added.ID
		movieTitle = added.Title
		s.logger.Info("AI added movie to library", "movie", movieTitle, "id", movieID)
	}

	if movieID == "" {
		return nil, fmt.Errorf("missing movie ID")
	}

	result, err := s.autoSvc.SearchMovie(ctx, movieID)
	if err != nil {
		return &CommandResponse{
			Action:      ActionFallback,
			Explanation: fmt.Sprintf("Search failed for %q: %s", movieTitle, err.Error()),
		}, nil
	}

	explanation := fmt.Sprintf("Searched for %q — %s", movieTitle, result.Status)
	if result.Reason != "" {
		explanation += ": " + result.Reason
	}
	if needsAdd {
		explanation = fmt.Sprintf("Added %q to library. ", movieTitle) + explanation
	}

	return &CommandResponse{
		Action:      ActionAutoSearch,
		Explanation: explanation,
		Result: map[string]any{
			"status":   string(result.Status),
			"reason":   result.Reason,
			"movie_id": movieID,
		},
	}, nil
}

// resolveDefaults picks the first library and its default quality profile for auto-adding movies.
func (s *Service) resolveDefaults(ctx context.Context) (libID, profID string, err error) {
	if s.librarySvc == nil {
		return "", "", fmt.Errorf("no libraries configured — add one in Settings > Libraries first")
	}
	libs, err := s.librarySvc.List(ctx)
	if err != nil || len(libs) == 0 {
		return "", "", fmt.Errorf("no libraries configured — add one in Settings > Libraries first")
	}

	lib := libs[0]
	if lib.DefaultQualityProfileID != "" {
		return lib.ID, lib.DefaultQualityProfileID, nil
	}

	// Library has no default profile — fall back to first available.
	if s.qualitySvc == nil {
		return "", "", fmt.Errorf("no quality profiles configured — add one in Settings > Quality Profiles first")
	}
	profiles, err := s.qualitySvc.List(ctx)
	if err != nil || len(profiles) == 0 {
		return "", "", fmt.Errorf("no quality profiles configured — add one in Settings > Quality Profiles first")
	}

	return lib.ID, profiles[0].ID, nil
}

func (s *Service) buildSystemPrompt(ctx context.Context) string {
	var sb strings.Builder
	sb.WriteString(`You are a command interpreter for Luminarr, a movie collection manager. Your job is to understand the user's intent and return a JSON action.

IMPORTANT: Respond with ONLY valid JSON. No markdown, no code fences, no explanation outside the JSON.

Available actions:

1. "navigate" — Go to a page in the app.
   {"action": "navigate", "params": {"path": "/some/path"}, "explanation": "brief description"}

   Valid paths:
   - "/" (Dashboard)
   - "/calendar" (Calendar)
   - "/wanted" (Wanted/Missing movies)
   - "/library-sync" (Library Sync)
   - "/stats" (Statistics)
   - "/queue" (Download Queue)
   - "/history" (History)
   - "/collections" (Collections)
   - "/settings/libraries" (Libraries)
   - "/settings/media-management" (Media Management)
   - "/settings/media-scanning" (Media Scanning)
   - "/settings/quality-profiles" (Quality Profiles)
   - "/settings/quality-definitions" (Quality Definitions)
   - "/settings/indexers" (Indexers)
   - "/settings/download-clients" (Download Clients)
   - "/settings/notifications" (Notifications)
   - "/settings/media-servers" (Media Servers)
   - "/settings/blocklist" (Blocklist)
   - "/settings/import" (Import)
   - "/settings/system" (System)
   - "/settings/app" (App Settings)
   - "/settings/custom-formats" (Custom Formats)

2. "search_movie" — Search for a movie by title (just shows results, does not download).
   {"action": "search_movie", "params": {"query": "movie title"}, "explanation": "brief description"}

3. "query_library" — Answer a question about the library using provided stats.
   {"action": "query_library", "params": {}, "result": {"answer": "text answer"}, "explanation": "brief description"}

4. "search_releases" — Search for releases/downloads for a specific movie (requires movie ID).
   {"action": "search_releases", "params": {"movie_id": 123}, "explanation": "brief description"}

5. "explain" — Explain a Luminarr concept.
   {"action": "explain", "params": {}, "explanation": "Clear explanation of the concept"}

6. "auto_search" — Search indexers for a release of a movie and automatically grab/download it. Use when the user says "grab", "download", "get", "fetch", or similar.
   {"action": "auto_search", "params": {"query": "movie title", "quality": "user's stated quality preference or empty"}, "explanation": "brief description"}
   The "quality" param MUST reflect exactly what the user asked for (e.g. "ultra-hd", "4K", "1080p", "remux", "best available"). Leave it empty ONLY if the user did not mention any quality preference.

7. "run_task" — Trigger a scheduled task. Valid task names: "rss_sync", "library_scan", "refresh_metadata".
   {"action": "run_task", "params": {"task_name": "library_scan"}, "explanation": "brief description"}

8. "fallback" — When you can't determine intent.
   {"action": "fallback", "explanation": "Helpful message about what you can do"}

`)

	// Add library context (aggregate stats only — no movie titles or paths).
	if s.statsSvc != nil {
		if cs, err := s.statsSvc.Collection(ctx); err == nil {
			sb.WriteString(fmt.Sprintf(`Current library stats:
- Total movies: %d
- Monitored: %d
- With files: %d
- Missing files: %d
- Needs upgrade: %d

`, cs.TotalMovies, cs.Monitored, cs.WithFile, cs.Missing, cs.NeedsUpgrade))
		}

		if tiers, err := s.statsSvc.QualityTiers(ctx); err == nil && len(tiers) > 0 {
			sb.WriteString("Quality breakdown:\n")
			for _, t := range tiers {
				sb.WriteString(fmt.Sprintf("- %s %s: %d movies\n", t.Resolution, t.Source, t.Count))
			}
			sb.WriteByte('\n')
		}

		if storage, err := s.statsSvc.Storage(ctx); err == nil {
			gb := float64(storage.TotalBytes) / (1024 * 1024 * 1024)
			sb.WriteString(fmt.Sprintf("Storage: %.1f GB across %d files\n\n", gb, storage.FileCount))
		}
	}

	sb.WriteString("Remember: respond with ONLY the JSON object, nothing else.")

	return sb.String()
}

func (s *Service) parseResponse(raw string) (*CommandResponse, error) {
	// Strip markdown code fences if present.
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var resp CommandResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	// Validate action type.
	switch resp.Action {
	case ActionNavigate, ActionSearchMovie, ActionQueryLibrary,
		ActionSearchRelease, ActionExplain, ActionFallback,
		ActionAutoSearch, ActionRunTask:
		// valid
	default:
		return nil, fmt.Errorf("unknown action type: %q", resp.Action)
	}

	return &resp, nil
}
