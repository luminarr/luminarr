package downloader_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/luminarr/luminarr/internal/core/downloader"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/testutil"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// ── Mock downloader ───────────────────────────────────────────────────────────

type mockClient struct {
	testErr error
	addErr  error
}

func (m *mockClient) Name() string              { return "mock" }
func (m *mockClient) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }
func (m *mockClient) Test(_ context.Context) error {
	return m.testErr
}
func (m *mockClient) Add(_ context.Context, _ plugin.Release) (string, error) {
	if m.addErr != nil {
		return "", m.addErr
	}
	return "mock-item-id", nil
}
func (m *mockClient) Status(_ context.Context, id string) (plugin.QueueItem, error) {
	return plugin.QueueItem{ClientItemID: id, Status: plugin.StatusDownloading}, nil
}
func (m *mockClient) GetQueue(_ context.Context) ([]plugin.QueueItem, error) { return nil, nil }
func (m *mockClient) Remove(_ context.Context, _ string, _ bool) error       { return nil }

// ── Helpers ───────────────────────────────────────────────────────────────────

func newTestReg(mock *mockClient) *registry.Registry {
	reg := registry.New()
	reg.RegisterDownloader("mock", func(_ json.RawMessage) (plugin.DownloadClient, error) {
		return mock, nil
	})
	return reg
}

func newServiceFromSQL(sqlDB *sql.DB, mock *mockClient) *downloader.Service {
	q := dbsqlite.New(sqlDB)
	return downloader.NewService(q, newTestReg(mock), nil)
}

func sampleSettings() json.RawMessage {
	b, _ := json.Marshal(map[string]string{"url": "http://localhost:8080"})
	return b
}

func sampleCreateReq() downloader.CreateRequest {
	return downloader.CreateRequest{
		Name:     "My qBittorrent",
		Kind:     "mock",
		Enabled:  true,
		Priority: 10,
		Settings: sampleSettings(),
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestService_Create(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	req := sampleCreateReq()
	cfg, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cfg.ID == "" {
		t.Error("Create() returned empty ID")
	}
	if cfg.Name != req.Name {
		t.Errorf("Name = %q, want %q", cfg.Name, req.Name)
	}
	if cfg.Kind != "mock" {
		t.Errorf("Kind = %q, want mock", cfg.Kind)
	}
	if !cfg.Enabled {
		t.Error("Enabled = false, want true")
	}
	if cfg.Priority != 10 {
		t.Errorf("Priority = %d, want 10", cfg.Priority)
	}
}

func TestService_Create_UnknownKind(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Kind = "does-not-exist"
	_, err := svc.Create(ctx, req)
	if err == nil {
		t.Fatal("Create() with unknown kind should return error")
	}
}

func TestService_Create_DefaultPriority(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Priority = 0
	cfg, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cfg.Priority != 25 {
		t.Errorf("Priority = %d, want 25 (default)", cfg.Priority)
	}
}

func TestService_Create_PersistsToDatabase(t *testing.T) {
	// Prove that Create writes to the DB and List reads it back.
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	created, err := svc.Create(ctx, sampleCreateReq())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// List must return the created item.
	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() after Create error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("List() count = %d, want 1", len(items))
	}
	if items[0].ID != created.ID {
		t.Errorf("List()[0].ID = %q, want %q", items[0].ID, created.ID)
	}
	if items[0].Name != created.Name {
		t.Errorf("List()[0].Name = %q, want %q", items[0].Name, created.Name)
	}
}

func TestService_Get(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	_, err := svc.Get(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, downloader.ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestService_List_Empty(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("List() count = %d, want 0", len(items))
	}
}

func TestService_List_OrderedByPriorityThenName(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Name, req.Priority = "A", 5
	_, _ = svc.Create(ctx, req)
	req.Name, req.Priority = "B", 1
	_, _ = svc.Create(ctx, req)

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() count = %d, want 2", len(items))
	}
	// priority 1 (B) before priority 5 (A)
	if items[0].Name != "B" {
		t.Errorf("first item = %q, want B (lowest priority first)", items[0].Name)
	}
}

func TestService_Update(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	updated, err := svc.Update(ctx, created.ID, downloader.CreateRequest{
		Name:     "Updated Name",
		Kind:     "mock",
		Enabled:  false,
		Priority: 50,
		Settings: created.Settings,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("Name = %q, want Updated Name", updated.Name)
	}
	if updated.Enabled {
		t.Error("Enabled = true, want false")
	}
	if updated.Priority != 50 {
		t.Errorf("Priority = %d, want 50", updated.Priority)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	_, err := svc.Update(ctx, "00000000-0000-0000-0000-000000000000", sampleCreateReq())
	if !errors.Is(err, downloader.ErrNotFound) {
		t.Errorf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestService_Delete(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err := svc.Get(ctx, created.ID)
	if !errors.Is(err, downloader.ErrNotFound) {
		t.Errorf("Get() after Delete error = %v, want ErrNotFound", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	err := svc.Delete(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, downloader.ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestService_Test_Success(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{testErr: nil})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	if err := svc.Test(ctx, created.ID); err != nil {
		t.Errorf("Test() error = %v, want nil", err)
	}
}

func TestService_Test_Failure(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{testErr: errors.New("connection refused")})
	ctx := context.Background()

	created, _ := svc.Create(ctx, sampleCreateReq())
	if err := svc.Test(ctx, created.ID); err == nil {
		t.Error("Test() should return error when client test fails")
	}
}

func TestService_Add_SubmitsToClient(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	req := sampleCreateReq()
	req.Enabled = true
	_, _ = svc.Create(ctx, req)

	release := plugin.Release{
		GUID:        "rel-1",
		Title:       "Movie.2024.1080p.BluRay.x264",
		Protocol:    plugin.ProtocolTorrent,
		DownloadURL: "magnet:?xt=urn:btih:aabbccddeeff00112233445566778899aabbccdd",
	}
	clientID, itemID, err := svc.Add(ctx, release, nil)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if clientID == "" {
		t.Error("Add() returned empty clientID")
	}
	if itemID != "mock-item-id" {
		t.Errorf("Add() itemID = %q, want mock-item-id", itemID)
	}
}

func TestService_Add_NoCompatibleClient(t *testing.T) {
	_, sqlDB := testutil.NewTestDBWithSQL(t)
	svc := newServiceFromSQL(sqlDB, &mockClient{})
	ctx := context.Background()

	// No enabled clients registered — should fail.
	_, _, err := svc.Add(ctx, plugin.Release{Protocol: plugin.ProtocolTorrent}, nil)
	if !errors.Is(err, downloader.ErrNoCompatibleClient) {
		t.Errorf("Add() error = %v, want ErrNoCompatibleClient", err)
	}
}
