package activity_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/luminarr/luminarr/internal/core/activity"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/testutil"
)

func newSvc(t *testing.T) (*activity.Service, *events.Bus) {
	t.Helper()
	q := testutil.NewTestDB(t)
	logger := slog.Default()
	bus := events.New(logger)
	svc := activity.NewService(q, logger)
	svc.Subscribe(bus)
	return svc, bus
}

func publishAndWait(bus *events.Bus, e events.Event) {
	bus.Publish(context.Background(), e)
	// Event handlers run in goroutines — give them a moment.
	time.Sleep(50 * time.Millisecond)
}

func TestHandleEvent_GrabStarted(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	publishAndWait(bus, events.Event{
		Type:      events.TypeGrabStarted,
		Timestamp: time.Now(),
		MovieID:   "movie-1",
		Data: map[string]any{
			"release_title": "Alien.1979.DC.1080p.BluRay",
			"indexer":       "The Pirate Bay",
		},
	})

	result, err := svc.List(ctx, nil, nil, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(result.Activities))
	}

	a := result.Activities[0]
	if a.Category != "grab" {
		t.Errorf("category: got %q, want %q", a.Category, "grab")
	}
	if a.Type != "grab_started" {
		t.Errorf("type: got %q, want %q", a.Type, "grab_started")
	}
	if a.MovieID == nil || *a.MovieID != "movie-1" {
		t.Errorf("movie_id: got %v, want %q", a.MovieID, "movie-1")
	}
	if a.Title != "Grabbed Alien.1979.DC.1080p.BluRay from The Pirate Bay" {
		t.Errorf("title: got %q", a.Title)
	}
}

func TestHandleEvent_AllTypes(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	testEvents := []events.Event{
		{Type: events.TypeGrabStarted, Timestamp: time.Now(), Data: map[string]any{"release_title": "R1", "indexer": "Idx"}},
		{Type: events.TypeGrabFailed, Timestamp: time.Now(), Data: map[string]any{"release_title": "R2", "reason": "rejected"}},
		{Type: events.TypeDownloadDone, Timestamp: time.Now(), Data: map[string]any{"release_title": "R3"}},
		{Type: events.TypeImportComplete, Timestamp: time.Now(), Data: map[string]any{"movie_title": "Alien", "quality": "1080p bluray"}},
		{Type: events.TypeImportFailed, Timestamp: time.Now(), Data: map[string]any{"movie_title": "Alien", "reason": "file not found"}},
		{Type: events.TypeMovieAdded, Timestamp: time.Now(), Data: map[string]any{"title": "Alien"}},
		{Type: events.TypeMovieDeleted, Timestamp: time.Now(), Data: map[string]any{"title": "Alien"}},
		{Type: events.TypeTaskStarted, Timestamp: time.Now(), Data: map[string]any{"task": "RSS Sync"}},
		{Type: events.TypeTaskFinished, Timestamp: time.Now(), Data: map[string]any{"task": "RSS Sync"}},
		{Type: events.TypeHealthIssue, Timestamp: time.Now(), Data: map[string]any{"check": "disk_space", "message": "path not accessible"}},
		{Type: events.TypeHealthOK, Timestamp: time.Now(), Data: map[string]any{"check": "disk_space"}},
		{Type: events.TypeBulkSearchComplete, Timestamp: time.Now(), Data: map[string]any{}},
	}

	for _, e := range testEvents {
		publishAndWait(bus, e)
	}

	result, err := svc.List(ctx, nil, nil, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Activities) != len(testEvents) {
		t.Fatalf("expected %d activities, got %d", len(testEvents), len(result.Activities))
	}
}

func TestHandleEvent_UnknownType_Skipped(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	publishAndWait(bus, events.Event{
		Type:      events.TypeBulkSearchProgress, // not mapped to activity
		Timestamp: time.Now(),
	})

	result, err := svc.List(ctx, nil, nil, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Activities) != 0 {
		t.Fatalf("expected 0 activities for unmapped event, got %d", len(result.Activities))
	}
}

func TestHandleEvent_NoMovieID(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	publishAndWait(bus, events.Event{
		Type:      events.TypeTaskFinished,
		Timestamp: time.Now(),
		Data:      map[string]any{"task": "RSS Sync"},
	})

	result, err := svc.List(ctx, nil, nil, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(result.Activities))
	}
	if result.Activities[0].MovieID != nil {
		t.Errorf("expected nil movie_id, got %v", result.Activities[0].MovieID)
	}
}

func TestList_CategoryFilter(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	publishAndWait(bus, events.Event{Type: events.TypeGrabStarted, Timestamp: time.Now(), Data: map[string]any{"release_title": "R1"}})
	publishAndWait(bus, events.Event{Type: events.TypeTaskFinished, Timestamp: time.Now(), Data: map[string]any{"task": "RSS"}})

	cat := "grab"
	result, err := svc.List(ctx, &cat, nil, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Activities) != 1 {
		t.Fatalf("expected 1 grab activity, got %d", len(result.Activities))
	}
	if result.Activities[0].Category != "grab" {
		t.Errorf("category: got %q, want %q", result.Activities[0].Category, "grab")
	}
}

func TestList_SinceFilter(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	old := time.Now().Add(-2 * time.Hour)
	recent := time.Now()

	publishAndWait(bus, events.Event{Type: events.TypeTaskStarted, Timestamp: old, Data: map[string]any{"task": "Old"}})
	publishAndWait(bus, events.Event{Type: events.TypeTaskFinished, Timestamp: recent, Data: map[string]any{"task": "Recent"}})

	since := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	result, err := svc.List(ctx, nil, &since, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Activities) != 1 {
		t.Fatalf("expected 1 recent activity, got %d", len(result.Activities))
	}
}

func TestList_Limit(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		publishAndWait(bus, events.Event{Type: events.TypeMovieAdded, Timestamp: time.Now(), Data: map[string]any{"title": "Movie"}})
	}

	result, err := svc.List(ctx, nil, nil, 3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Activities) != 3 {
		t.Fatalf("expected 3 activities, got %d", len(result.Activities))
	}
	if result.Total != 5 {
		t.Errorf("total: got %d, want 5", result.Total)
	}
}

func TestList_NewestFirst(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	t1 := time.Now().Add(-2 * time.Second)
	t2 := time.Now()

	publishAndWait(bus, events.Event{Type: events.TypeMovieAdded, Timestamp: t1, Data: map[string]any{"title": "First"}})
	publishAndWait(bus, events.Event{Type: events.TypeMovieAdded, Timestamp: t2, Data: map[string]any{"title": "Second"}})

	result, err := svc.List(ctx, nil, nil, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Activities) != 2 {
		t.Fatalf("expected 2, got %d", len(result.Activities))
	}
	if result.Activities[0].Title != "Added Second to library" {
		t.Errorf("expected newest first, got %q", result.Activities[0].Title)
	}
}

func TestList_EmptyResult(t *testing.T) {
	t.Parallel()
	svc, _ := newSvc(t)
	ctx := context.Background()

	result, err := svc.List(ctx, nil, nil, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result.Activities == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(result.Activities) != 0 {
		t.Fatalf("expected 0, got %d", len(result.Activities))
	}
	if result.Total != 0 {
		t.Errorf("total: got %d, want 0", result.Total)
	}
}

func TestPrune(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now()

	publishAndWait(bus, events.Event{Type: events.TypeMovieAdded, Timestamp: old, Data: map[string]any{"title": "Old"}})
	publishAndWait(bus, events.Event{Type: events.TypeMovieAdded, Timestamp: recent, Data: map[string]any{"title": "Recent"}})

	if err := svc.Prune(ctx, 24*time.Hour); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	result, err := svc.List(ctx, nil, nil, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Activities) != 1 {
		t.Fatalf("expected 1 surviving activity, got %d", len(result.Activities))
	}
	if result.Activities[0].Title != "Added Recent to library" {
		t.Errorf("wrong activity survived: %q", result.Activities[0].Title)
	}
}

func TestConcurrentWrites(t *testing.T) {
	t.Parallel()
	svc, bus := newSvc(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(ctx, events.Event{
				Type:      events.TypeMovieAdded,
				Timestamp: time.Now(),
				Data:      map[string]any{"title": "Movie"},
			})
		}()
	}
	wg.Wait()

	// Event handlers run in goroutines. Poll until all have completed
	// rather than using a fixed sleep (avoids flakiness under -race).
	deadline := time.After(5 * time.Second)
	for {
		result, err := svc.List(ctx, nil, nil, 100)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(result.Activities) == 50 {
			return // success
		}
		select {
		case <-deadline:
			t.Fatalf("timed out: expected 50 activities, got %d", len(result.Activities))
		case <-time.After(50 * time.Millisecond):
			// retry
		}
	}
}

func TestValidCategory(t *testing.T) {
	t.Parallel()
	valid := []string{"grab", "import", "task", "health", "movie"}
	for _, c := range valid {
		if !activity.ValidCategory(c) {
			t.Errorf("expected %q to be valid", c)
		}
	}
	invalid := []string{"", "unknown", "system", "GRAB"}
	for _, c := range invalid {
		if activity.ValidCategory(c) {
			t.Errorf("expected %q to be invalid", c)
		}
	}
}
