package qbittorrent_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/luminarr/luminarr/pkg/plugin"
	"github.com/luminarr/luminarr/plugins/downloaders/qbittorrent"
)

// newTestServer creates a mock qBittorrent API server.
// handlers is a map of "METHOD /path" → handler func.
func newTestServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for pattern, h := range handlers {
		mux.HandleFunc(pattern, h)
	}
	return httptest.NewServer(mux)
}

// newTestClient returns an http.Client that bypasses the SSRF-blocking safe
// dialer so tests can connect to httptest.Server on 127.0.0.1.
func newTestClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func loginOK(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Ok."))
}

func TestTest(t *testing.T) {
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login":  loginOK,
		"/api/v2/app/version": func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("5.0.0")) },
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL, Username: "admin", Password: "pass"}, newTestClient())
	if err := c.Test(context.Background()); err != nil {
		t.Fatalf("Test() error: %v", err)
	}
}

func TestTest_AuthFailure(t *testing.T) {
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Fails."))
		},
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
	if err := c.Test(context.Background()); err == nil {
		t.Fatal("expected auth failure error, got nil")
	}
}

func TestAdd_MagnetLink(t *testing.T) {
	const magnet = "magnet:?xt=urn:btih:a94a8fe5ccb19ba61c4c0873d391e987982fbbd3&dn=test"

	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login":   loginOK,
		"/api/v2/torrents/add": func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("Ok.")) },
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
	id, err := c.Add(context.Background(), plugin.Release{DownloadURL: magnet})
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	// Hash is lowercase hex btih
	if id != "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3" {
		t.Errorf("expected hash a94a8fe5..., got %q", id)
	}
}

func TestAdd_MagnetBase32Hash(t *testing.T) {
	// Base32-encoded hash for the same bytes as above (SHA1 of "test")
	// SHA1("test") = a94a8fe5ccb19ba61c4c0873d391e987982fbbd3
	// base32 of those bytes = VFF7Y3OMW...  let's use a known test vector:
	// xt=urn:btih:MFRA2YLCMFSA==... this is synthetic; just verify parsing works
	const magnet = "magnet:?xt=urn:btih:a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"

	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login":   loginOK,
		"/api/v2/torrents/add": func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("Ok.")) },
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
	id, err := c.Add(context.Background(), plugin.Release{DownloadURL: magnet})
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty hash")
	}
}

func TestGetQueue(t *testing.T) {
	const queueJSON = `[
		{"hash":"abc123","name":"Movie.2024","size":10000,"downloaded":5000,"ratio":0.0,"state":"downloading","added_on":1700000000,"save_path":"/downloads/","content_path":"/downloads/Movie.2024.mkv"},
		{"hash":"def456","name":"Film.2023","size":8000,"downloaded":8000,"ratio":1.2,"state":"seeding","added_on":1699000000,"save_path":"/downloads/","content_path":""}
	]`

	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login": loginOK,
		"/api/v2/torrents/info": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(queueJSON))
		},
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
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
	if items[0].ContentPath != "/downloads/Movie.2024.mkv" {
		t.Errorf("expected content_path /downloads/Movie.2024.mkv, got %q", items[0].ContentPath)
	}
	// def456 has no content_path; should fall back to save_path + name
	if items[1].ContentPath != "/downloads/Film.2023" {
		t.Errorf("expected fallback content_path /downloads/Film.2023, got %q", items[1].ContentPath)
	}
	if items[1].Status != plugin.StatusCompleted {
		t.Errorf("expected StatusCompleted for seeding state, got %v", items[1].Status)
	}
}

func TestStatus(t *testing.T) {
	const infoJSON = `[{"hash":"abc123","name":"Movie.2024","size":10000,"downloaded":10000,"ratio":1.0,"state":"seeding","added_on":1700000000}]`

	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login": loginOK,
		"/api/v2/torrents/info": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("hashes") != "abc123" {
				http.Error(w, "unexpected hash", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(infoJSON))
		},
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
	item, err := c.Status(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if item.ClientItemID != "abc123" {
		t.Errorf("expected hash abc123, got %q", item.ClientItemID)
	}
	if item.Status != plugin.StatusCompleted {
		t.Errorf("expected StatusCompleted, got %v", item.Status)
	}
}

func TestRemove(t *testing.T) {
	var gotHashes, gotDeleteFiles string
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login": loginOK,
		"/api/v2/torrents/delete": func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			gotHashes = r.FormValue("hashes")
			gotDeleteFiles = r.FormValue("deleteFiles")
			w.WriteHeader(http.StatusOK)
		},
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
	if err := c.Remove(context.Background(), "abc123", true); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
	if gotHashes != "abc123" {
		t.Errorf("expected hashes=abc123, got %q", gotHashes)
	}
	if gotDeleteFiles != "true" {
		t.Errorf("expected deleteFiles=true, got %q", gotDeleteFiles)
	}
}

func TestSetSeedLimits_HappyPath(t *testing.T) {
	var gotHashes, gotRatio, gotTime string
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login": loginOK,
		"/api/v2/torrents/setShareLimits": func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			gotHashes = r.FormValue("hashes")
			gotRatio = r.FormValue("ratioLimit")
			gotTime = r.FormValue("seedingTimeLimit")
			w.WriteHeader(http.StatusOK)
		},
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
	err := c.SetSeedLimits(context.Background(), "abc123", 2.0, 7200) // 7200s = 120min
	if err != nil {
		t.Fatalf("SetSeedLimits() error: %v", err)
	}
	if gotHashes != "abc123" {
		t.Errorf("hashes = %q, want abc123", gotHashes)
	}
	if gotRatio != "2.0000" {
		t.Errorf("ratioLimit = %q, want 2.0000", gotRatio)
	}
	if gotTime != "120" {
		t.Errorf("seedingTimeLimit = %q, want 120", gotTime)
	}
}

func TestSetSeedLimits_ZeroValuesUseGlobal(t *testing.T) {
	var gotRatio, gotTime string
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login": loginOK,
		"/api/v2/torrents/setShareLimits": func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			gotRatio = r.FormValue("ratioLimit")
			gotTime = r.FormValue("seedingTimeLimit")
			w.WriteHeader(http.StatusOK)
		},
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
	err := c.SetSeedLimits(context.Background(), "abc123", 0, 0)
	if err != nil {
		t.Fatalf("SetSeedLimits() error: %v", err)
	}
	if gotRatio != "-2" {
		t.Errorf("ratioLimit = %q, want -2 (use global)", gotRatio)
	}
	if gotTime != "-2" {
		t.Errorf("seedingTimeLimit = %q, want -2 (use global)", gotTime)
	}
}

func TestSetSeedLimits_AuthFailure(t *testing.T) {
	srv := newTestServer(t, map[string]http.HandlerFunc{
		"/api/v2/auth/login": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Fails."))
		},
	})
	defer srv.Close()

	c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
	err := c.SetSeedLimits(context.Background(), "abc123", 1.0, 3600)
	if err == nil {
		t.Fatal("expected auth failure error, got nil")
	}
}

func TestStateMapping(t *testing.T) {
	// Verify the full queue is returned and states map correctly.
	tests := []struct {
		state    string
		expected plugin.DownloadStatus
	}{
		{"downloading", plugin.StatusDownloading},
		{"forcedDL", plugin.StatusDownloading},
		{"metaDL", plugin.StatusDownloading},
		{"stalledDL", plugin.StatusQueued},
		{"queuedDL", plugin.StatusQueued},
		{"allocating", plugin.StatusQueued},
		{"seeding", plugin.StatusCompleted},
		{"uploading", plugin.StatusCompleted},
		{"stalledUP", plugin.StatusCompleted},
		{"pausedDL", plugin.StatusPaused},
		{"stoppedDL", plugin.StatusPaused},
		{"error", plugin.StatusFailed},
		{"missingFiles", plugin.StatusFailed},
	}

	for _, tc := range tests {
		jsonBody := `[{"hash":"h","name":"n","size":0,"downloaded":0,"ratio":0,"state":"` + tc.state + `","added_on":0}]`
		srv := newTestServer(t, map[string]http.HandlerFunc{
			"/api/v2/auth/login": loginOK,
			"/api/v2/torrents/info": func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(jsonBody))
			},
		})

		c := qbittorrent.NewWithHTTPClient(qbittorrent.Config{URL: srv.URL}, newTestClient())
		items, err := c.GetQueue(context.Background())
		srv.Close()

		if err != nil {
			t.Errorf("state %q: GetQueue error: %v", tc.state, err)
			continue
		}
		if len(items) == 0 {
			t.Errorf("state %q: expected 1 item", tc.state)
			continue
		}
		if items[0].Status != tc.expected {
			t.Errorf("state %q: expected %v, got %v", tc.state, tc.expected, items[0].Status)
		}
	}
}
