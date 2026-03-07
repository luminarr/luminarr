// Package notifications wires the event bus to the notification plugin system.
// The Dispatcher subscribes to all bus events and forwards them to every
// enabled notification channel that has opted in to that event type.
package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/luminarr/luminarr/internal/core/movie"
	"github.com/luminarr/luminarr/internal/core/notification"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// Dispatcher subscribes to the event bus and dispatches matching events to
// all enabled notification plugins.
type Dispatcher struct {
	q        dbsqlite.Querier
	reg      *registry.Registry
	bus      *events.Bus
	logger   *slog.Logger
	movieSvc *movie.Service
}

// NewDispatcher creates a Dispatcher. Call Subscribe() to start receiving events.
// movieSvc may be nil — enrichment will be skipped.
func NewDispatcher(q dbsqlite.Querier, reg *registry.Registry, bus *events.Bus, logger *slog.Logger, movieSvc *movie.Service) *Dispatcher {
	return &Dispatcher{q: q, reg: reg, bus: bus, logger: logger, movieSvc: movieSvc}
}

// Subscribe registers the dispatcher as a handler on the event bus.
// Must be called once after construction, before the server starts serving.
func (d *Dispatcher) Subscribe() {
	d.bus.Subscribe(func(ctx context.Context, e events.Event) {
		d.handle(ctx, e)
	})
}

func (d *Dispatcher) handle(ctx context.Context, e events.Event) {
	// Load all enabled notification configs fresh on each event so that
	// config changes take effect immediately without a restart.
	rows, err := d.q.ListEnabledNotifications(ctx)
	if err != nil {
		d.logger.Warn("dispatcher: could not load notification configs", "error", err)
		return
	}

	eventType := string(e.Type)

	for _, row := range rows {
		// Check whether this notifier subscribes to this event type.
		if !subscribesTo(row.OnEvents, eventType) {
			continue
		}

		n, err := d.reg.NewNotifier(row.Kind, []byte(row.Settings))
		if err != nil {
			d.logger.Warn("dispatcher: could not instantiate notifier",
				"id", row.ID,
				"kind", row.Kind,
				"error", err,
			)
			continue
		}

		ne := notification.EventToNotification(eventType, e.MovieID, e.Data)
		d.enrichFromMovie(ctx, &ne)

		if err := n.Notify(ctx, ne); err != nil {
			d.logger.Warn("dispatcher: notification failed",
				"id", row.ID,
				"kind", row.Kind,
				"event", eventType,
				"error", err,
			)
		}
	}
}

// enrichFromMovie merges movie metadata into the notification data map.
// Best-effort: errors are silently ignored.
func (d *Dispatcher) enrichFromMovie(ctx context.Context, ne *plugin.NotificationEvent) {
	if d.movieSvc == nil || ne.MovieID == "" {
		return
	}
	m, err := d.movieSvc.Get(ctx, ne.MovieID)
	if err != nil {
		return
	}
	if ne.Data == nil {
		ne.Data = make(map[string]any)
	}
	ne.Data["movie_title"] = m.Title
	ne.Data["movie_year"] = m.Year
	ne.Data["tmdb_id"] = m.TMDBID
	ne.Data["imdb_id"] = m.IMDBID
	ne.Data["poster_url"] = m.PosterURL
	ne.Data["movie_path"] = m.Path
	if m.IMDBID != "" {
		ne.Data["imdb_url"] = fmt.Sprintf("https://www.imdb.com/title/%s/", m.IMDBID)
	}
	if m.TMDBID > 0 {
		ne.Data["tmdb_url"] = fmt.Sprintf("https://www.themoviedb.org/movie/%d", m.TMDBID)
	}
}

// subscribesTo reports whether the JSON-encoded on_events list includes eventType.
// An empty list means subscribe to all events.
func subscribesTo(onEventsJSON string, eventType string) bool {
	if onEventsJSON == "" || onEventsJSON == "[]" || onEventsJSON == "null" {
		return true // no filter = subscribe to everything
	}
	var events []string
	if err := json.Unmarshal([]byte(onEventsJSON), &events); err != nil {
		return false
	}
	for _, e := range events {
		if e == eventType {
			return true
		}
	}
	return false
}
