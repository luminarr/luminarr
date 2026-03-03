// Package radarrimport fetches data from a running Radarr instance and
// creates matching records in Luminarr's database using the existing service
// layer. It is used only for the one-time migration wizard in the UI.
package radarrimport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/davidfic/luminarr/internal/safedialer"
)

// ── Radarr API types ──────────────────────────────────────────────────────────

type radarrStatus struct {
	Version string `json:"version"`
}

type radarrProfile struct {
	ID             int                  `json:"id"`
	Name           string               `json:"name"`
	UpgradeAllowed bool                 `json:"upgradeAllowed"`
	Cutoff         radarrProfileQuality `json:"cutoff"`
	Items          []radarrProfileItem  `json:"items"`
}

type radarrProfileQuality struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// radarrProfileItem can be a single quality or a group of qualities.
type radarrProfileItem struct {
	Quality radarrProfileQuality `json:"quality"`
	Items   []radarrProfileItem  `json:"items"` // non-nil when this is a group
	Allowed bool                 `json:"allowed"`
}

// Qualities returns all non-group, allowed qualities from this item,
// recursing into groups.
func (item radarrProfileItem) qualities() []radarrProfileQuality {
	if len(item.Items) == 0 {
		// Leaf quality item.
		if item.Allowed {
			return []radarrProfileQuality{item.Quality}
		}
		return nil
	}
	// Group — recurse into its children.
	var out []radarrProfileQuality
	for _, child := range item.Items {
		out = append(out, child.qualities()...)
	}
	return out
}

type radarrRootFolder struct {
	Path       string `json:"path"`
	FreeSpace  int64  `json:"freeSpace"`
	Accessible bool   `json:"accessible"`
}

type radarrField struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

// stringValue extracts the string value of this field, returning "" if it
// is not a JSON string.
func (f radarrField) stringValue() string {
	var s string
	if err := json.Unmarshal(f.Value, &s); err != nil {
		return ""
	}
	return s
}

// intValue extracts the integer value of this field, returning 0 on failure.
func (f radarrField) intValue() int {
	var n int
	if err := json.Unmarshal(f.Value, &n); err != nil {
		return 0
	}
	return n
}

// boolValue extracts the boolean value of this field, returning false on failure.
func (f radarrField) boolValue() bool {
	var b bool
	if err := json.Unmarshal(f.Value, &b); err != nil {
		return false
	}
	return b
}

type radarrIndexer struct {
	ID             int           `json:"id"`
	Name           string        `json:"name"`
	ConfigContract string        `json:"configContract"`
	EnableRss      bool          `json:"enableRss"`
	Fields         []radarrField `json:"fields"`
}

type radarrDownloadClient struct {
	ID             int           `json:"id"`
	Name           string        `json:"name"`
	ConfigContract string        `json:"configContract"`
	Enable         bool          `json:"enable"`
	Fields         []radarrField `json:"fields"`
}

type radarrMovie struct {
	ID               int    `json:"id"`
	TmdbID           int    `json:"tmdbId"`
	Title            string `json:"title"`
	Monitored        bool   `json:"monitored"`
	QualityProfileID int    `json:"qualityProfileId"`
	RootFolderPath   string `json:"rootFolderPath"`
}

// ── Client ────────────────────────────────────────────────────────────────────

// Client is an HTTP client for the Radarr v3 API.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient creates a new Radarr API client. baseURL should be the root URL
// of the Radarr instance, e.g. "http://radarr.local:7878".
//
// The client uses safedialer.Transport() to block SSRF to internal addresses.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		// LANTransport allows RFC-1918 and loopback (Radarr is typically on
		// localhost or a LAN address) while still blocking cloud-metadata
		// link-local ranges (169.254.169.254 etc.).
		http: &http.Client{Timeout: 30 * time.Second, Transport: safedialer.LANTransport()},
	}
}

// get makes a GET request to the Radarr API and decodes the JSON response into
// out. Returns a descriptive error if the request fails or the status is not 2xx.
func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to Radarr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("Radarr rejected the API key (401 Unauthorized)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Radarr returned HTTP %d for %s", resp.StatusCode, path)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding Radarr response from %s: %w", path, err)
	}
	return nil
}

// GetStatus verifies the connection and returns the Radarr version.
func (c *Client) GetStatus(ctx context.Context) (*radarrStatus, error) {
	var s radarrStatus
	if err := c.get(ctx, "/api/v3/system/status", &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetQualityProfiles returns all quality profiles from Radarr.
func (c *Client) GetQualityProfiles(ctx context.Context) ([]radarrProfile, error) {
	var profiles []radarrProfile
	if err := c.get(ctx, "/api/v3/qualityProfile", &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

// GetRootFolders returns all root folders configured in Radarr.
func (c *Client) GetRootFolders(ctx context.Context) ([]radarrRootFolder, error) {
	var folders []radarrRootFolder
	if err := c.get(ctx, "/api/v3/rootFolder", &folders); err != nil {
		return nil, err
	}
	return folders, nil
}

// GetIndexers returns all indexers configured in Radarr.
func (c *Client) GetIndexers(ctx context.Context) ([]radarrIndexer, error) {
	var indexers []radarrIndexer
	if err := c.get(ctx, "/api/v3/indexer", &indexers); err != nil {
		return nil, err
	}
	return indexers, nil
}

// GetDownloadClients returns all download clients configured in Radarr.
func (c *Client) GetDownloadClients(ctx context.Context) ([]radarrDownloadClient, error) {
	var clients []radarrDownloadClient
	if err := c.get(ctx, "/api/v3/downloadclient", &clients); err != nil {
		return nil, err
	}
	return clients, nil
}

// GetMovies returns all movies in the Radarr library.
func (c *Client) GetMovies(ctx context.Context) ([]radarrMovie, error) {
	var movies []radarrMovie
	if err := c.get(ctx, "/api/v3/movie", &movies); err != nil {
		return nil, err
	}
	return movies, nil
}

// ── Field helpers ─────────────────────────────────────────────────────────────

// fieldString extracts the string value of the named field from a slice.
// Returns "" if the field is absent or not a string.
func fieldString(fields []radarrField, name string) string {
	for _, f := range fields {
		if f.Name == name {
			return f.stringValue()
		}
	}
	return ""
}

// fieldInt extracts the integer value of the named field. Returns 0 if absent.
func fieldInt(fields []radarrField, name string) int {
	for _, f := range fields {
		if f.Name == name {
			return f.intValue()
		}
	}
	return 0
}

// fieldBool extracts the boolean value of the named field. Returns false if absent.
func fieldBool(fields []radarrField, name string) bool {
	for _, f := range fields {
		if f.Name == name {
			return f.boolValue()
		}
	}
	return false
}

// buildURL constructs an HTTP(S) URL from host, port, and SSL flag.
func buildURL(host string, port int, useSsl bool, urlBase string) string {
	scheme := "http"
	if useSsl {
		scheme = "https"
	}
	base := fmt.Sprintf("%s://%s:%d", scheme, host, port)
	if urlBase != "" && urlBase != "/" {
		base += "/" + strings.Trim(urlBase, "/")
	}
	return base
}
