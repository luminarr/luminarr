// Package jellyfin implements a Luminarr media server plugin for Jellyfin.
// On import_complete it triggers a full library refresh.
package jellyfin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/safedialer"
	"github.com/luminarr/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterMediaServer("jellyfin", func(settings json.RawMessage) (plugin.MediaServer, error) {
		var cfg Config
		if err := json.Unmarshal(settings, &cfg); err != nil {
			return nil, fmt.Errorf("jellyfin: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, fmt.Errorf("jellyfin: url is required")
		}
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("jellyfin: api_key is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterMediaServerSanitizer("jellyfin", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["api_key"]; ok {
			m["api_key"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the user-supplied settings for a Jellyfin server.
type Config struct {
	URL           string `json:"url"`
	APIKey        string `json:"api_key"`
	SkipTLSVerify bool   `json:"skip_tls_verify,omitempty"`
}

// Server is a Jellyfin media server plugin instance.
type Server struct {
	cfg    Config
	client *http.Client
}

// New creates a new Server from the given config.
func New(cfg Config) *Server {
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	transport := safedialer.LANTransport()
	if cfg.SkipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in for self-signed certs
	}
	return &Server{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second, Transport: transport},
	}
}

func (s *Server) Name() string { return "Jellyfin" }

// RefreshLibrary triggers a full library refresh on the Jellyfin server.
func (s *Server) RefreshLibrary(ctx context.Context, _ string) error {
	url := fmt.Sprintf("%s/Library/Refresh", s.cfg.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("jellyfin: building refresh request: %w", err)
	}
	req.Header.Set("X-Emby-Token", s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("jellyfin: refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("jellyfin: refresh returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

// Test verifies that the Jellyfin server is reachable with the configured API key.
func (s *Server) Test(ctx context.Context) error {
	url := fmt.Sprintf("%s/System/Info", s.cfg.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("jellyfin: building test request: %w", err)
	}
	req.Header.Set("X-Emby-Token", s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("jellyfin: test request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("jellyfin: test returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

// ── WatchProvider implementation ─────────────────────────────────────────────

// WatchHistory returns watch events since the given timestamp.
// Implements plugin.WatchProvider.
func (s *Server) WatchHistory(ctx context.Context, since time.Time) ([]plugin.WatchEvent, error) {
	// Jellyfin: GET /Items?IncludeItemTypes=Movie&IsPlayed=true&SortBy=DatePlayed&SortOrder=Descending
	// We need to first get the admin user ID.
	userID, err := s.getFirstUserID(ctx)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/Users/%s/Items?IncludeItemTypes=Movie&IsPlayed=true&SortBy=DatePlayed&SortOrder=Descending&Fields=ProviderIds&Limit=200",
		s.cfg.URL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("jellyfin: building watch-history request: %w", err)
	}
	req.Header.Set("X-Emby-Token", s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jellyfin: watch-history request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("jellyfin: watch-history returned %d: %s", resp.StatusCode, body)
	}

	var envelope struct {
		Items []struct {
			Name        string   `json:"Name"`
			ProviderIDs struct { //nolint:revive // matches Jellyfin JSON field name
				Tmdb string `json:"Tmdb"`
			} `json:"ProviderIds"`
			UserData struct {
				LastPlayedDate string `json:"LastPlayedDate"`
				PlayCount      int    `json:"PlayCount"`
				Played         bool   `json:"Played"`
			} `json:"UserData"`
		} `json:"Items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("jellyfin: decoding watch history: %w", err)
	}

	var events []plugin.WatchEvent
	for _, item := range envelope.Items {
		if !item.UserData.Played || item.ProviderIDs.Tmdb == "" {
			continue
		}
		tmdbID, err := strconv.Atoi(item.ProviderIDs.Tmdb)
		if err != nil || tmdbID == 0 {
			continue
		}
		watchedAt, err := time.Parse(time.RFC3339, item.UserData.LastPlayedDate)
		if err != nil {
			// Try alternative format.
			watchedAt, err = time.Parse("2006-01-02T15:04:05.0000000Z", item.UserData.LastPlayedDate)
			if err != nil {
				continue
			}
		}
		if watchedAt.Before(since) {
			continue
		}
		events = append(events, plugin.WatchEvent{
			TMDBID:    tmdbID,
			Title:     item.Name,
			WatchedAt: watchedAt.UTC(),
			UserName:  "jellyfin",
		})
	}
	return events, nil
}

// getFirstUserID returns the first admin user's ID from the Jellyfin server.
func (s *Server) getFirstUserID(ctx context.Context) (string, error) {
	reqURL := fmt.Sprintf("%s/Users", s.cfg.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("jellyfin: building users request: %w", err)
	}
	req.Header.Set("X-Emby-Token", s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("jellyfin: users request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("jellyfin: users returned %d", resp.StatusCode)
	}

	var users []struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return "", fmt.Errorf("jellyfin: decoding users: %w", err)
	}
	if len(users) == 0 {
		return "", fmt.Errorf("jellyfin: no users found")
	}
	return users[0].ID, nil
}
