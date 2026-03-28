// Package activity provides a persistent activity log that records events
// from the in-memory event bus so they survive restarts and are queryable.
package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
)

// Category groups related event types for filtering.
type Category string

const (
	CategoryGrab   Category = "grab"
	CategoryImport Category = "import"
	CategoryTask   Category = "task"
	CategoryHealth Category = "health"
	CategoryMovie  Category = "movie"
)

// ValidCategory returns true if c is a known category.
func ValidCategory(c string) bool {
	switch Category(c) {
	case CategoryGrab, CategoryImport, CategoryTask, CategoryHealth, CategoryMovie:
		return true
	}
	return false
}

// Activity is the domain representation of an activity log entry.
type Activity struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Category  string         `json:"category"`
	MovieID   *string        `json:"movie_id,omitempty"`
	Title     string         `json:"title"`
	Detail    map[string]any `json:"detail,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// ListResult is the paginated response from List.
type ListResult struct {
	Activities []Activity `json:"activities"`
	Total      int64      `json:"total"`
}

// Service records and queries activity log entries.
type Service struct {
	q      dbsqlite.Querier
	logger *slog.Logger
}

// NewService creates a new activity service.
func NewService(q dbsqlite.Querier, logger *slog.Logger) *Service {
	return &Service{q: q, logger: logger}
}

// Subscribe registers the service as an event bus handler. Call this once
// during startup after constructing the service.
func (s *Service) Subscribe(bus *events.Bus) {
	bus.Subscribe(s.handleEvent)
}

// handleEvent converts an event bus event into a persistent activity record.
func (s *Service) handleEvent(ctx context.Context, e events.Event) {
	cat, title := s.classify(e)
	if cat == "" {
		return // unknown event type — skip
	}

	var detailStr *string
	if len(e.Data) > 0 {
		b, err := json.Marshal(e.Data)
		if err == nil {
			str := string(b)
			detailStr = &str
		}
	}

	var movieID *string
	if e.MovieID != "" {
		movieID = &e.MovieID
	}

	err := s.q.InsertActivity(ctx, dbsqlite.InsertActivityParams{
		ID:        uuid.New().String(),
		Type:      string(e.Type),
		Category:  string(cat),
		MovieID:   movieID,
		Title:     title,
		Detail:    detailStr,
		CreatedAt: e.Timestamp.UTC().Format(time.RFC3339),
	})
	if err != nil {
		s.logger.Warn("failed to record activity", "type", e.Type, "error", err)
	}
}

// classify maps an event type to a category and human-readable title.
func (s *Service) classify(e events.Event) (Category, string) {
	data := e.Data
	str := func(key string) string {
		if v, ok := data[key].(string); ok {
			return v
		}
		return ""
	}

	switch e.Type {
	case events.TypeGrabStarted:
		release := str("release_title")
		indexer := str("indexer")
		if indexer != "" {
			return CategoryGrab, fmt.Sprintf("Grabbed %s from %s", release, indexer)
		}
		return CategoryGrab, fmt.Sprintf("Grabbed %s", release)

	case events.TypeGrabFailed:
		release := str("release_title")
		reason := str("reason")
		if reason != "" {
			return CategoryGrab, fmt.Sprintf("Grab failed for %s: %s", release, reason)
		}
		return CategoryGrab, fmt.Sprintf("Grab failed for %s", release)

	case events.TypeDownloadDone:
		release := str("release_title")
		return CategoryImport, fmt.Sprintf("Download complete: %s", release)

	case events.TypeImportComplete:
		title := str("movie_title")
		quality := str("quality")
		if quality != "" {
			return CategoryImport, fmt.Sprintf("Imported %s — %s", title, quality)
		}
		return CategoryImport, fmt.Sprintf("Imported %s", title)

	case events.TypeImportFailed:
		title := str("movie_title")
		reason := str("reason")
		if reason != "" {
			return CategoryImport, fmt.Sprintf("Import failed for %s: %s", title, reason)
		}
		return CategoryImport, fmt.Sprintf("Import failed for %s", title)

	case events.TypeMovieAdded:
		title := str("title")
		return CategoryMovie, fmt.Sprintf("Added %s to library", title)

	case events.TypeMovieDeleted:
		title := str("title")
		return CategoryMovie, fmt.Sprintf("Deleted %s", title)

	case events.TypeMovieUpdated:
		title := str("title")
		return CategoryMovie, fmt.Sprintf("Updated %s", title)

	case events.TypeTaskStarted:
		task := str("task")
		return CategoryTask, fmt.Sprintf("%s started", task)

	case events.TypeTaskFinished:
		task := str("task")
		return CategoryTask, fmt.Sprintf("%s completed", task)

	case events.TypeHealthIssue:
		check := str("check")
		message := str("message")
		return CategoryHealth, fmt.Sprintf("%s: %s", check, message)

	case events.TypeHealthOK:
		check := str("check")
		return CategoryHealth, fmt.Sprintf("%s: recovered", check)

	case events.TypeBulkSearchComplete:
		return CategoryGrab, "Bulk search completed"

	default:
		return "", ""
	}
}

// List returns activity entries matching the given filters.
func (s *Service) List(ctx context.Context, category *string, since *string, limit int64) (*ListResult, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var catFilter interface{}
	if category != nil {
		catFilter = *category
	}
	var sinceFilter interface{}
	if since != nil {
		sinceFilter = *since
	}

	rows, err := s.q.ListActivities(ctx, dbsqlite.ListActivitiesParams{
		Category: catFilter,
		Since:    sinceFilter,
		Limit:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("listing activities: %w", err)
	}

	total, err := s.q.CountActivities(ctx, dbsqlite.CountActivitiesParams{
		Category: catFilter,
		Since:    sinceFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("counting activities: %w", err)
	}

	activities := make([]Activity, 0, len(rows))
	for _, r := range rows {
		a := Activity{
			ID:        r.ID,
			Type:      r.Type,
			Category:  r.Category,
			MovieID:   r.MovieID,
			Title:     r.Title,
			CreatedAt: r.CreatedAt,
		}
		if r.Detail != nil {
			_ = json.Unmarshal([]byte(*r.Detail), &a.Detail)
		}
		activities = append(activities, a)
	}

	return &ListResult{
		Activities: activities,
		Total:      total,
	}, nil
}

// Prune deletes activity entries older than the given duration.
func (s *Service) Prune(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	return s.q.PruneActivities(ctx, cutoff)
}
