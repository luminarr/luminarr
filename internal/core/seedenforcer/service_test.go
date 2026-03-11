package seedenforcer_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/luminarr/luminarr/internal/core/indexer"
	"github.com/luminarr/luminarr/internal/core/seedenforcer"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/internal/testutil/mock"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// ── Mock providers ──────────────────────────────────────────────────────────

type mockSeedCriteriaProvider struct {
	criteria indexer.SeedCriteria
	err      error
}

func (m *mockSeedCriteriaProvider) GetSeedCriteria(_ context.Context, _ string) (indexer.SeedCriteria, error) {
	return m.criteria, m.err
}

type mockClientProvider struct {
	client plugin.DownloadClient
	err    error
}

func (m *mockClientProvider) ClientFor(_ context.Context, _ string) (plugin.DownloadClient, error) {
	return m.client, m.err
}

// mockQuerier implements just GetGrabByID for the tests.
type mockQuerier struct {
	dbsqlite.Querier
	grab dbsqlite.GrabHistory
	err  error
}

func (m *mockQuerier) GetGrabByID(_ context.Context, _ string) (dbsqlite.GrabHistory, error) {
	return m.grab, m.err
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }

func torrentGrab() dbsqlite.GrabHistory {
	return dbsqlite.GrabHistory{
		ID:               "grab-1",
		MovieID:          "movie-1",
		IndexerID:        strPtr("indexer-1"),
		Protocol:         string(plugin.ProtocolTorrent),
		DownloadClientID: strPtr("client-1"),
		ClientItemID:     strPtr("hash-abc123"),
	}
}

// publishAndWait publishes an event and waits long enough for the bus handler
// goroutine to finish. Uses a generous sleep to avoid flakiness.
func publishAndWait(bus *events.Bus, e events.Event) {
	bus.Publish(context.Background(), e)
	time.Sleep(100 * time.Millisecond)
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestSeedEnforcer_HappyPath(t *testing.T) {
	dl := &mock.DownloadClient{}
	var gotRatio float64
	var gotTime int
	var gotItemID string
	called := make(chan struct{})
	dl.SetSeedLimitsFunc = func(_ context.Context, clientItemID string, ratioLimit float64, seedTimeSecs int) error {
		gotItemID = clientItemID
		gotRatio = ratioLimit
		gotTime = seedTimeSecs
		close(called)
		return nil
	}

	bus := events.New(slog.Default())
	svc := seedenforcer.NewService(
		&mockQuerier{grab: torrentGrab()},
		&mockSeedCriteriaProvider{criteria: indexer.SeedCriteria{SeedRatio: 2.0, SeedTimeMinutes: 120}},
		&mockClientProvider{client: dl},
		bus,
		slog.Default(),
	)
	svc.Subscribe()

	bus.Publish(context.Background(), events.Event{
		Type: events.TypeImportComplete,
		Data: map[string]any{"grab_id": "grab-1"},
	})

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SetSeedLimits")
	}

	if gotItemID != "hash-abc123" {
		t.Errorf("clientItemID = %q, want hash-abc123", gotItemID)
	}
	if gotRatio != 2.0 {
		t.Errorf("ratioLimit = %v, want 2.0", gotRatio)
	}
	if gotTime != 7200 {
		t.Errorf("seedTimeSecs = %v, want 7200", gotTime)
	}
}

func TestSeedEnforcer_NZBProtocolSkipped(t *testing.T) {
	dl := &mock.DownloadClient{}
	grab := torrentGrab()
	grab.Protocol = string(plugin.ProtocolNZB)

	bus := events.New(slog.Default())
	svc := seedenforcer.NewService(
		&mockQuerier{grab: grab},
		&mockSeedCriteriaProvider{criteria: indexer.SeedCriteria{SeedRatio: 2.0}},
		&mockClientProvider{client: dl},
		bus,
		slog.Default(),
	)
	svc.Subscribe()

	publishAndWait(bus, events.Event{
		Type: events.TypeImportComplete,
		Data: map[string]any{"grab_id": "grab-1"},
	})

	for _, c := range dl.GetCalls() {
		if c == "SetSeedLimits" {
			t.Error("SetSeedLimits should not be called for usenet protocol")
		}
	}
}

func TestSeedEnforcer_MissingIDsSkipped(t *testing.T) {
	dl := &mock.DownloadClient{}
	grab := torrentGrab()
	grab.IndexerID = nil // missing indexer ID

	bus := events.New(slog.Default())
	svc := seedenforcer.NewService(
		&mockQuerier{grab: grab},
		&mockSeedCriteriaProvider{criteria: indexer.SeedCriteria{SeedRatio: 2.0}},
		&mockClientProvider{client: dl},
		bus,
		slog.Default(),
	)
	svc.Subscribe()

	publishAndWait(bus, events.Event{
		Type: events.TypeImportComplete,
		Data: map[string]any{"grab_id": "grab-1"},
	})

	for _, c := range dl.GetCalls() {
		if c == "SetSeedLimits" {
			t.Error("SetSeedLimits should not be called when indexer_id is nil")
		}
	}
}

func TestSeedEnforcer_ZeroCriteriaSkipped(t *testing.T) {
	dl := &mock.DownloadClient{}

	bus := events.New(slog.Default())
	svc := seedenforcer.NewService(
		&mockQuerier{grab: torrentGrab()},
		&mockSeedCriteriaProvider{criteria: indexer.SeedCriteria{SeedRatio: 0, SeedTimeMinutes: 0}},
		&mockClientProvider{client: dl},
		bus,
		slog.Default(),
	)
	svc.Subscribe()

	publishAndWait(bus, events.Event{
		Type: events.TypeImportComplete,
		Data: map[string]any{"grab_id": "grab-1"},
	})

	for _, c := range dl.GetCalls() {
		if c == "SetSeedLimits" {
			t.Error("SetSeedLimits should not be called when both criteria are zero")
		}
	}
}

func TestSeedEnforcer_NonSeedLimiterClient(t *testing.T) {
	// A download client that does NOT implement SeedLimiter.
	nonLimiter := &basicClient{}

	bus := events.New(slog.Default())
	svc := seedenforcer.NewService(
		&mockQuerier{grab: torrentGrab()},
		&mockSeedCriteriaProvider{criteria: indexer.SeedCriteria{SeedRatio: 2.0}},
		&mockClientProvider{client: nonLimiter},
		bus,
		slog.Default(),
	)
	svc.Subscribe()

	publishAndWait(bus, events.Event{
		Type: events.TypeImportComplete,
		Data: map[string]any{"grab_id": "grab-1"},
	})
	// Should not panic — just gracefully skip.
}

func TestSeedEnforcer_SetSeedLimitsError(t *testing.T) {
	dl := &mock.DownloadClient{}
	called := make(chan struct{})
	dl.SetSeedLimitsFunc = func(_ context.Context, _ string, _ float64, _ int) error {
		close(called)
		return errors.New("connection refused")
	}

	bus := events.New(slog.Default())
	svc := seedenforcer.NewService(
		&mockQuerier{grab: torrentGrab()},
		&mockSeedCriteriaProvider{criteria: indexer.SeedCriteria{SeedRatio: 1.5}},
		&mockClientProvider{client: dl},
		bus,
		slog.Default(),
	)
	svc.Subscribe()

	bus.Publish(context.Background(), events.Event{
		Type: events.TypeImportComplete,
		Data: map[string]any{"grab_id": "grab-1"},
	})

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SetSeedLimits")
	}
	// Should not panic — error is logged but not propagated.
}

func TestSeedEnforcer_WrongEventTypeIgnored(t *testing.T) {
	dl := &mock.DownloadClient{}

	bus := events.New(slog.Default())
	svc := seedenforcer.NewService(
		&mockQuerier{grab: torrentGrab()},
		&mockSeedCriteriaProvider{criteria: indexer.SeedCriteria{SeedRatio: 2.0}},
		&mockClientProvider{client: dl},
		bus,
		slog.Default(),
	)
	svc.Subscribe()

	publishAndWait(bus, events.Event{
		Type: events.TypeGrabStarted, // wrong event type
		Data: map[string]any{"grab_id": "grab-1"},
	})

	for _, c := range dl.GetCalls() {
		if c == "SetSeedLimits" {
			t.Error("SetSeedLimits should not be called for non-import events")
		}
	}
}

// basicClient implements plugin.DownloadClient but NOT plugin.SeedLimiter.
type basicClient struct{}

func (b *basicClient) Name() string              { return "Basic" }
func (b *basicClient) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }
func (b *basicClient) Test(_ context.Context) error {
	return nil
}
func (b *basicClient) Add(_ context.Context, _ plugin.Release) (string, error) {
	return "", nil
}
func (b *basicClient) Status(_ context.Context, _ string) (plugin.QueueItem, error) {
	return plugin.QueueItem{}, nil
}
func (b *basicClient) GetQueue(_ context.Context) ([]plugin.QueueItem, error) {
	return nil, nil
}
func (b *basicClient) Remove(_ context.Context, _ string, _ bool) error {
	return nil
}
