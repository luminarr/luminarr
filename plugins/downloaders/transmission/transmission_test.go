package transmission_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davidfic/luminarr/pkg/plugin"
	"github.com/davidfic/luminarr/plugins/downloaders/transmission"
)

// rpcHandler is a mock handler for Transmission RPC requests.
// It provides the session-ID automatically on the first call (409 dance).
type rpcHandler struct {
	sessionID string
	handlers  map[string]func(args json.RawMessage) (any, string)
}

func newRPCHandler() *rpcHandler {
	return &rpcHandler{
		sessionID: "test-session-id",
		handlers:  make(map[string]func(args json.RawMessage) (any, string)),
	}
}

func (h *rpcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enforce session-ID dance.
	if r.Header.Get("X-Transmission-Session-Id") != h.sessionID {
		w.Header().Set("X-Transmission-Session-Id", h.sessionID)
		w.WriteHeader(http.StatusConflict)
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req struct {
		Method    string          `json:"method"`
		Arguments json.RawMessage `json:"arguments"`
	}
	_ = json.Unmarshal(body, &req)

	handler, ok := h.handlers[req.Method]
	if !ok {
		writeRPC(w, nil, "method not found")
		return
	}
	result, errStr := handler(req.Arguments)
	writeRPC(w, result, errStr)
}

func writeRPC(w http.ResponseWriter, result any, errResult string) {
	if errResult == "" {
		errResult = "success"
	}
	resp := map[string]any{"result": errResult}
	if result != nil {
		b, _ := json.Marshal(result)
		resp["arguments"] = json.RawMessage(b)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func newClient(t *testing.T, h *rpcHandler) (*transmission.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	c := transmission.NewWithHTTPClient(
		transmission.Config{URL: srv.URL},
		&http.Client{},
	)
	return c, srv
}

func TestTest_Success(t *testing.T) {
	h := newRPCHandler()
	h.handlers["session-get"] = func(_ json.RawMessage) (any, string) {
		return map[string]any{"version": "4.0.0"}, ""
	}
	c, srv := newClient(t, h)
	defer srv.Close()

	if err := c.Test(context.Background()); err != nil {
		t.Fatalf("Test() error: %v", err)
	}
}

func TestTest_SessionIDRetry(t *testing.T) {
	// Verifies that the 409 → retry dance works transparently.
	calls := 0
	h := newRPCHandler()
	h.handlers["session-get"] = func(_ json.RawMessage) (any, string) {
		calls++
		return map[string]any{}, ""
	}
	c, srv := newClient(t, h)
	defer srv.Close()

	if err := c.Test(context.Background()); err != nil {
		t.Fatalf("Test() error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 successful RPC call, got %d", calls)
	}
}

func TestTest_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := transmission.NewWithHTTPClient(
		transmission.Config{URL: srv.URL, Username: "bad", Password: "bad"},
		&http.Client{},
	)
	if err := c.Test(context.Background()); err == nil {
		t.Fatal("expected auth failure error, got nil")
	}
}

func TestAdd_MagnetLink(t *testing.T) {
	const magnet = "magnet:?xt=urn:btih:a94a8fe5ccb19ba61c4c0873d391e987982fbbd3&dn=test"

	h := newRPCHandler()
	h.handlers["torrent-add"] = func(_ json.RawMessage) (any, string) {
		return map[string]any{
			"torrent-added": map[string]any{
				"id":         42,
				"hashString": "A94A8FE5CCB19BA61C4C0873D391E987982FBBD3",
				"name":       "test",
			},
		}, ""
	}
	c, srv := newClient(t, h)
	defer srv.Close()

	id, err := c.Add(context.Background(), plugin.Release{DownloadURL: magnet})
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if id != "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3" {
		t.Errorf("expected lowercase hash, got %q", id)
	}
}

func TestAdd_Duplicate(t *testing.T) {
	const magnet = "magnet:?xt=urn:btih:a94a8fe5ccb19ba61c4c0873d391e987982fbbd3&dn=test"

	h := newRPCHandler()
	h.handlers["torrent-add"] = func(_ json.RawMessage) (any, string) {
		return map[string]any{
			"torrent-duplicate": map[string]any{
				"id":         17,
				"hashString": "A94A8FE5CCB19BA61C4C0873D391E987982FBBD3",
				"name":       "test",
			},
		}, ""
	}
	c, srv := newClient(t, h)
	defer srv.Close()

	id, err := c.Add(context.Background(), plugin.Release{DownloadURL: magnet})
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty hash for duplicate")
	}
}

func TestGetQueue(t *testing.T) {
	h := newRPCHandler()
	h.handlers["torrent-get"] = func(_ json.RawMessage) (any, string) {
		return map[string]any{
			"torrents": []map[string]any{
				{
					"id": 1, "hashString": "ABC123", "name": "Movie.2024",
					"status": 4, "percentDone": 0.5, "sizeWhenDone": int64(10000),
					"leftUntilDone": int64(5000), "downloadDir": "/downloads",
					"error": 0, "errorString": "", "uploadRatio": 0.0,
					"addedDate": int64(1700000000),
				},
				{
					"id": 2, "hashString": "DEF456", "name": "Film.2023",
					"status": 6, "percentDone": 1.0, "sizeWhenDone": int64(8000),
					"leftUntilDone": int64(0), "downloadDir": "/downloads",
					"error": 0, "errorString": "", "uploadRatio": 1.5,
					"addedDate": int64(1699000000),
				},
			},
		}, ""
	}
	c, srv := newClient(t, h)
	defer srv.Close()

	items, err := c.GetQueue(context.Background())
	if err != nil {
		t.Fatalf("GetQueue() error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].ClientItemID != "abc123" {
		t.Errorf("expected hash abc123, got %q", items[0].ClientItemID)
	}
	if items[0].Status != plugin.StatusDownloading {
		t.Errorf("expected StatusDownloading, got %v", items[0].Status)
	}
	if items[0].Downloaded != 5000 {
		t.Errorf("expected downloaded 5000, got %d", items[0].Downloaded)
	}
	if items[0].ContentPath != "/downloads/Movie.2024" {
		t.Errorf("expected content path /downloads/Movie.2024, got %q", items[0].ContentPath)
	}

	if items[1].Status != plugin.StatusCompleted {
		t.Errorf("expected StatusCompleted for seeding, got %v", items[1].Status)
	}
}

func TestStatus_NotFound(t *testing.T) {
	h := newRPCHandler()
	h.handlers["torrent-get"] = func(_ json.RawMessage) (any, string) {
		return map[string]any{"torrents": []any{}}, ""
	}
	c, srv := newClient(t, h)
	defer srv.Close()

	_, err := c.Status(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing torrent, got nil")
	}
}

func TestRemove(t *testing.T) {
	h := newRPCHandler()
	// Remove needs torrent-get first to look up the numeric ID.
	h.handlers["torrent-get"] = func(_ json.RawMessage) (any, string) {
		return map[string]any{
			"torrents": []map[string]any{
				{
					"id": 42, "hashString": "ABC123", "name": "Movie",
					"status": 6, "percentDone": 1.0, "sizeWhenDone": int64(0),
					"leftUntilDone": int64(0), "downloadDir": "/dl",
					"error": 0, "errorString": "", "uploadRatio": 0.0,
					"addedDate": int64(0),
				},
			},
		}, ""
	}
	var gotDeleteFiles bool
	h.handlers["torrent-remove"] = func(args json.RawMessage) (any, string) {
		var a struct {
			IDs            []int64 `json:"ids"`
			DeleteLocalData bool   `json:"delete-local-data"`
		}
		_ = json.Unmarshal(args, &a)
		gotDeleteFiles = a.DeleteLocalData
		return nil, ""
	}
	c, srv := newClient(t, h)
	defer srv.Close()

	if err := c.Remove(context.Background(), "abc123", true); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
	if !gotDeleteFiles {
		t.Error("expected delete-local-data=true")
	}
}

func TestStatusMapping(t *testing.T) {
	tests := []struct {
		status   int
		hasError bool
		expected plugin.DownloadStatus
	}{
		{0, false, plugin.StatusPaused},       // Stopped
		{1, false, plugin.StatusQueued},        // QueuedToVerify
		{2, false, plugin.StatusQueued},        // Verifying
		{3, false, plugin.StatusQueued},        // QueuedToDownload
		{4, false, plugin.StatusDownloading},   // Downloading
		{5, false, plugin.StatusCompleted},     // QueuedToSeed
		{6, false, plugin.StatusCompleted},     // Seeding
		{4, true, plugin.StatusFailed},         // Downloading with error
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("status_%d_err_%v", tc.status, tc.hasError), func(t *testing.T) {
			errCode := 0
			errStr := ""
			if tc.hasError {
				errCode = 2 // tracker error
				errStr = "tracker gave HTTP 404"
			}
			h := newRPCHandler()
			h.handlers["torrent-get"] = func(_ json.RawMessage) (any, string) {
				return map[string]any{
					"torrents": []map[string]any{{
						"id": 1, "hashString": "H", "name": "n",
						"status": tc.status, "percentDone": 0.0,
						"sizeWhenDone": int64(0), "leftUntilDone": int64(0),
						"downloadDir": "/dl", "error": errCode,
						"errorString": errStr, "uploadRatio": 0.0,
						"addedDate": int64(0),
					}},
				}, ""
			}
			c, srv := newClient(t, h)
			defer srv.Close()

			items, err := c.GetQueue(context.Background())
			if err != nil {
				t.Fatalf("GetQueue() error: %v", err)
			}
			if len(items) == 0 {
				t.Fatal("expected 1 item")
			}
			if items[0].Status != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, items[0].Status)
			}
		})
	}
}

func TestFactoryValidation(t *testing.T) {
	// Verify New() doesn't panic with empty config and returns correct metadata.
	c := transmission.New(transmission.Config{})
	if c.Name() != "Transmission" {
		t.Errorf("expected name Transmission, got %q", c.Name())
	}
	if c.Protocol() != plugin.ProtocolTorrent {
		t.Errorf("expected ProtocolTorrent, got %v", c.Protocol())
	}
}
