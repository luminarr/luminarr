// Package notifications wires the event bus to the notification plugin system.
// The Dispatcher subscribes to all bus events and forwards them to every
// enabled notification channel that has opted in to that event type.
package notifications

import (
	"context"
	"log/slog"

	"github.com/davidfic/luminarr/internal/core/notification"
	dbsqlite "github.com/davidfic/luminarr/internal/db/generated/sqlite"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/registry"
)

// Dispatcher subscribes to the event bus and dispatches matching events to
// all enabled notification plugins.
type Dispatcher struct {
	q      dbsqlite.Querier
	reg    *registry.Registry
	bus    *events.Bus
	logger *slog.Logger
}

// NewDispatcher creates a Dispatcher. Call Subscribe() to start receiving events.
func NewDispatcher(q dbsqlite.Querier, reg *registry.Registry, bus *events.Bus, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{q: q, reg: reg, bus: bus, logger: logger}
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

// subscribesTo reports whether the JSON-encoded on_events list includes eventType.
// An empty list means subscribe to all events.
func subscribesTo(onEventsJSON string, eventType string) bool {
	if onEventsJSON == "" || onEventsJSON == "[]" || onEventsJSON == "null" {
		return true // no filter = subscribe to everything
	}

	// Fast path: string contains check avoids JSON parsing for most cases.
	// We look for the quoted event type surrounded by string delimiters.
	needle := `"` + eventType + `"`
	for i := 0; i <= len(onEventsJSON)-len(needle); i++ {
		if onEventsJSON[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
