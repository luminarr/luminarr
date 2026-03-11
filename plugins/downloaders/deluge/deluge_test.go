package deluge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/luminarr/luminarr/pkg/plugin"
)

// fakeDelugeHandler returns a handler that simulates Deluge's JSON-RPC API.
func fakeDelugeHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "auth.login":
			json.NewEncoder(w).Encode(rpcResponse{Result: json.RawMessage(`true`), ID: req.ID})
		case "core.get_torrent_status":
			fields := map[string]any{
				"name":       "Inception.2010.1080p",
				"total_size": 8000000000,
				"total_done": 4000000000,
				"ratio":      0.5,
				"state":      "Downloading",
				"save_path":  "/downloads",
			}
			result, _ := json.Marshal(fields)
			json.NewEncoder(w).Encode(rpcResponse{Result: result, ID: req.ID})
		case "core.get_torrents_status":
			torrents := map[string]map[string]any{
				"aabbccdd": {
					"name":       "Movie.2024",
					"total_size": 5000000000,
					"total_done": 5000000000,
					"ratio":      1.0,
					"state":      "Seeding",
					"save_path":  "/downloads",
				},
			}
			result, _ := json.Marshal(torrents)
			json.NewEncoder(w).Encode(rpcResponse{Result: result, ID: req.ID})
		case "core.add_torrent_magnet":
			json.NewEncoder(w).Encode(rpcResponse{Result: json.RawMessage(`"AABBCCDD"`), ID: req.ID})
		case "core.remove_torrent":
			json.NewEncoder(w).Encode(rpcResponse{Result: json.RawMessage(`true`), ID: req.ID})
		case "core.set_torrent_options":
			json.NewEncoder(w).Encode(rpcResponse{Result: json.RawMessage(`true`), ID: req.ID})
		default:
			json.NewEncoder(w).Encode(rpcResponse{
				Error: &rpcError{Code: -1, Message: "unknown method"},
				ID:    req.ID,
			})
		}
	})
}

func newTestClient(t *testing.T) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(fakeDelugeHandler(t))
	c := New(Config{URL: srv.URL, Password: "deluge"})
	c.http = srv.Client()
	return c, srv
}

func TestTest_Success(t *testing.T) {
	c, srv := newTestClient(t)
	defer srv.Close()

	if err := c.Test(context.Background()); err != nil {
		t.Fatalf("Test() = %v", err)
	}
}

func TestStatus_ReturnsItem(t *testing.T) {
	c, srv := newTestClient(t)
	defer srv.Close()

	item, err := c.Status(context.Background(), "aabbccdd")
	if err != nil {
		t.Fatalf("Status() = %v", err)
	}
	if item.Title != "Inception.2010.1080p" {
		t.Errorf("Title = %q", item.Title)
	}
	if item.Status != plugin.StatusDownloading {
		t.Errorf("Status = %q, want downloading", item.Status)
	}
	if item.Size != 8000000000 {
		t.Errorf("Size = %d", item.Size)
	}
}

func TestGetQueue_ReturnsItems(t *testing.T) {
	c, srv := newTestClient(t)
	defer srv.Close()

	items, err := c.GetQueue(context.Background())
	if err != nil {
		t.Fatalf("GetQueue() = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].Status != plugin.StatusCompleted {
		t.Errorf("Status = %q, want completed (Seeding)", items[0].Status)
	}
}

func TestAdd_Magnet(t *testing.T) {
	c, srv := newTestClient(t)
	defer srv.Close()

	hash, err := c.Add(context.Background(), plugin.Release{
		DownloadURL: "magnet:?xt=urn:btih:aabbccdd",
		Protocol:    plugin.ProtocolTorrent,
	})
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}
	if hash != "aabbccdd" {
		t.Errorf("hash = %q, want aabbccdd", hash)
	}
}

func TestRemove_Success(t *testing.T) {
	c, srv := newTestClient(t)
	defer srv.Close()

	if err := c.Remove(context.Background(), "aabbccdd", false); err != nil {
		t.Fatalf("Remove() = %v", err)
	}
}

func TestSetSeedLimits_HappyPath(t *testing.T) {
	var gotMethod string
	var gotParams []json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "auth.login":
			json.NewEncoder(w).Encode(rpcResponse{Result: json.RawMessage(`true`), ID: req.ID})
		case "core.set_torrent_options":
			gotMethod = req.Method
			for _, p := range req.Params {
				b, _ := json.Marshal(p)
				gotParams = append(gotParams, b)
			}
			json.NewEncoder(w).Encode(rpcResponse{Result: json.RawMessage(`true`), ID: req.ID})
		}
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL, Password: "deluge"})
	c.http = srv.Client()

	err := c.SetSeedLimits(context.Background(), "aabbccdd", 2.0, 7200)
	if err != nil {
		t.Fatalf("SetSeedLimits() = %v", err)
	}
	if gotMethod != "core.set_torrent_options" {
		t.Errorf("method = %q, want core.set_torrent_options", gotMethod)
	}
}

func TestSetSeedLimits_ZeroValuesNoOp(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "auth.login":
			json.NewEncoder(w).Encode(rpcResponse{Result: json.RawMessage(`true`), ID: req.ID})
		case "core.set_torrent_options":
			called = true
			json.NewEncoder(w).Encode(rpcResponse{Result: json.RawMessage(`true`), ID: req.ID})
		}
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL, Password: "deluge"})
	c.http = srv.Client()

	err := c.SetSeedLimits(context.Background(), "aabbccdd", 0, 0)
	if err != nil {
		t.Fatalf("SetSeedLimits() = %v", err)
	}
	if called {
		t.Error("expected no RPC call for zero values, but core.set_torrent_options was called")
	}
}

func TestSetSeedLimits_RPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "auth.login":
			json.NewEncoder(w).Encode(rpcResponse{Result: json.RawMessage(`true`), ID: req.ID})
		case "core.set_torrent_options":
			json.NewEncoder(w).Encode(rpcResponse{
				Error: &rpcError{Code: -1, Message: "torrent not found"},
				ID:    req.ID,
			})
		}
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL, Password: "deluge"})
	c.http = srv.Client()

	err := c.SetSeedLimits(context.Background(), "aabbccdd", 1.5, 3600)
	if err == nil {
		t.Fatal("expected RPC error, got nil")
	}
}

func TestMapState(t *testing.T) {
	tests := []struct {
		state string
		want  plugin.DownloadStatus
	}{
		{"Downloading", plugin.StatusDownloading},
		{"Initializing", plugin.StatusDownloading},
		{"Seeding", plugin.StatusCompleted},
		{"Paused", plugin.StatusPaused},
		{"Error", plugin.StatusFailed},
		{"Queued", plugin.StatusQueued},
		{"Checking", plugin.StatusQueued},
		{"UnknownState", plugin.StatusQueued},
	}
	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := mapState(tt.state)
			if got != tt.want {
				t.Errorf("mapState(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}
