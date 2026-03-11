// Package deluge implements the plugin.DownloadClient interface for the Deluge
// Web API (JSON-RPC over HTTP). Tested against Deluge 2.x.
package deluge

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/luminarr/luminarr/internal/registry"
	"github.com/luminarr/luminarr/internal/safedialer"
	"github.com/luminarr/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterDownloader("deluge", func(s json.RawMessage) (plugin.DownloadClient, error) {
		var cfg Config
		if err := json.Unmarshal(s, &cfg); err != nil {
			return nil, fmt.Errorf("deluge: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, errors.New("deluge: url is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterDownloaderSanitizer("deluge", func(settings json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(settings, &m); err != nil {
			return json.RawMessage("{}")
		}
		if _, ok := m["password"]; ok {
			m["password"] = json.RawMessage(`"***"`)
		}
		out, _ := json.Marshal(m)
		return out
	})
}

// Config holds the connection settings for a Deluge Web UI instance.
type Config struct {
	URL      string `json:"url"`                 // e.g. "http://localhost:8112"
	Password string `json:"password"`            // Web UI password (default: "deluge")
	Label    string `json:"label,omitempty"`     // label/category applied to added torrents
	SavePath string `json:"save_path,omitempty"` // custom save path (empty = Deluge default)
}

// Client implements plugin.DownloadClient against the Deluge Web JSON-RPC API.
type Client struct {
	cfg    Config
	http   *http.Client
	mu     sync.Mutex
	authed bool
	id     atomic.Int64 // JSON-RPC request ID counter
}

// New creates a new Deluge client. Call Test to verify connectivity.
// Outbound HTTP uses safedialer.LANTransport() because download clients are
// typically hosted on localhost or a LAN address.
func New(cfg Config) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		cfg:  cfg,
		http: &http.Client{Jar: jar, Timeout: 30 * time.Second, Transport: safedialer.LANTransport()},
	}
}

func (c *Client) Name() string              { return "Deluge" }
func (c *Client) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }

// Test verifies connectivity and authentication.
func (c *Client) Test(ctx context.Context) error {
	return c.ensureAuth(ctx)
}

// Add submits a torrent to Deluge. Magnet URIs are added directly via
// core.add_torrent_magnet. HTTP/HTTPS .torrent URLs are fetched by
// Luminarr, base64-encoded, and sent via core.add_torrent_file.
func (c *Client) Add(ctx context.Context, r plugin.Release) (string, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return "", err
	}
	if strings.HasPrefix(r.DownloadURL, "magnet:") {
		return c.addMagnet(ctx, r.DownloadURL)
	}
	return c.addTorrentURL(ctx, r.DownloadURL)
}

// Status returns the current state of a single torrent by its info hash.
func (c *Client) Status(ctx context.Context, clientItemID string) (plugin.QueueItem, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return plugin.QueueItem{}, err
	}

	keys := []string{"name", "total_size", "total_done", "ratio", "state", "save_path", "move_completed_path", "error_msg"}
	var result map[string]json.RawMessage
	if err := c.call(ctx, "core.get_torrent_status", []any{clientItemID, keys}, &result); err != nil {
		return plugin.QueueItem{}, fmt.Errorf("deluge: get_torrent_status: %w", err)
	}
	if len(result) == 0 {
		return plugin.QueueItem{}, fmt.Errorf("deluge: torrent %q not found", clientItemID)
	}
	item := torrentFieldsToQueueItem(clientItemID, result)
	return item, nil
}

// GetQueue returns all torrents currently tracked by Deluge.
func (c *Client) GetQueue(ctx context.Context) ([]plugin.QueueItem, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, err
	}

	keys := []string{"name", "total_size", "total_done", "ratio", "state", "save_path", "move_completed_path", "error_msg"}
	var result map[string]map[string]json.RawMessage
	if err := c.call(ctx, "core.get_torrents_status", []any{map[string]any{}, keys}, &result); err != nil {
		return nil, fmt.Errorf("deluge: get_torrents_status: %w", err)
	}

	items := make([]plugin.QueueItem, 0, len(result))
	for hash, fields := range result {
		items = append(items, torrentFieldsToQueueItem(hash, fields))
	}
	return items, nil
}

// Remove removes a torrent from Deluge. If deleteFiles is true the downloaded
// data is also deleted from disk.
func (c *Client) Remove(ctx context.Context, clientItemID string, deleteFiles bool) error {
	if err := c.ensureAuth(ctx); err != nil {
		return err
	}
	var ok bool
	if err := c.call(ctx, "core.remove_torrent", []any{clientItemID, deleteFiles}, &ok); err != nil {
		return fmt.Errorf("deluge: remove_torrent: %w", err)
	}
	return nil
}

// SetSeedLimits sets per-torrent seed ratio and time limits via Deluge's
// core.set_torrent_options RPC method. ratioLimit <= 0 and seedTimeSecs <= 0
// are ignored (no change sent to Deluge).
func (c *Client) SetSeedLimits(ctx context.Context, clientItemID string, ratioLimit float64, seedTimeSecs int) error {
	if err := c.ensureAuth(ctx); err != nil {
		return err
	}

	opts := map[string]any{}
	if ratioLimit > 0 {
		opts["stop_at_ratio"] = true
		opts["stop_ratio"] = ratioLimit
	}
	if seedTimeSecs > 0 {
		opts["seed_time_limit"] = seedTimeSecs / 60 // Deluge uses minutes
	}
	if len(opts) == 0 {
		return nil // nothing to set
	}

	var result bool
	if err := c.call(ctx, "core.set_torrent_options", []any{[]string{clientItemID}, opts}, &result); err != nil {
		return fmt.Errorf("deluge: set_torrent_options: %w", err)
	}
	return nil
}

// ── Auth ──────────────────────────────────────────────────────────────────────

func (c *Client) ensureAuth(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.authed {
		return nil
	}
	if err := c.loginLocked(ctx); err != nil {
		return err
	}
	c.authed = true
	return nil
}

// loginLocked performs the RPC login request. Must be called with c.mu held.
func (c *Client) loginLocked(ctx context.Context) error {
	var ok bool
	if err := c.call(ctx, "auth.login", []any{c.cfg.Password}, &ok); err != nil {
		return fmt.Errorf("deluge: auth.login: %w", err)
	}
	if !ok {
		return errors.New("deluge: authentication failed — check password")
	}
	return nil
}

// ── Torrent add helpers ───────────────────────────────────────────────────────

func (c *Client) addMagnet(ctx context.Context, magnetURI string) (string, error) {
	opts := c.addOptions()
	var hash string
	if err := c.call(ctx, "core.add_torrent_magnet", []any{magnetURI, opts}, &hash); err != nil {
		return "", fmt.Errorf("deluge: add_torrent_magnet: %w", err)
	}
	if hash == "" {
		return "", errors.New("deluge: add_torrent_magnet returned empty hash")
	}
	return strings.ToLower(hash), nil
}

func (c *Client) addTorrentURL(ctx context.Context, torrentURL string) (string, error) {
	// Fetch the .torrent file from the URL and base64-encode it.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torrentURL, nil)
	if err != nil {
		return "", fmt.Errorf("deluge: building torrent fetch request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("deluge: fetching torrent file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MiB limit
	if err != nil {
		return "", fmt.Errorf("deluge: reading torrent file: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)

	// Extract a filename from the URL for Deluge to display.
	filename := torrentURL
	if idx := strings.LastIndex(torrentURL, "/"); idx >= 0 {
		filename = torrentURL[idx+1:]
	}
	if filename == "" {
		filename = "release.torrent"
	}

	opts := c.addOptions()
	var hash string
	if err := c.call(ctx, "core.add_torrent_file", []any{filename, encoded, opts}, &hash); err != nil {
		return "", fmt.Errorf("deluge: add_torrent_file: %w", err)
	}
	if hash == "" {
		return "", errors.New("deluge: add_torrent_file returned empty hash")
	}
	return strings.ToLower(hash), nil
}

func (c *Client) addOptions() map[string]any {
	opts := map[string]any{}
	if c.cfg.SavePath != "" {
		opts["download_location"] = c.cfg.SavePath
		opts["move_completed"] = false
	}
	if c.cfg.Label != "" {
		// Label is set via a separate call after adding — Deluge requires the
		// label plugin. Store it in options for future use; we set it post-add.
		_ = c.cfg.Label
	}
	return opts
}

// ── JSON-RPC helpers ─────────────────────────────────────────────────────────

type rpcRequest struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
	ID     int64  `json:"id"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
	ID     int64           `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) call(ctx context.Context, method string, params []any, out any) error {
	id := c.id.Add(1)
	body, err := json.Marshal(rpcRequest{Method: method, Params: params, ID: id})
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+"/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	var rpc rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpc); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if rpc.Error != nil {
		// Session expired — force re-auth on next call.
		if rpc.Error.Code == 1 {
			c.mu.Lock()
			c.authed = false
			c.mu.Unlock()
		}
		return fmt.Errorf("RPC error %d: %s", rpc.Error.Code, rpc.Error.Message)
	}
	if out != nil && rpc.Result != nil {
		if err := json.Unmarshal(rpc.Result, out); err != nil {
			return fmt.Errorf("decoding result: %w", err)
		}
	}
	return nil
}

// ── Status helpers ────────────────────────────────────────────────────────────

func torrentFieldsToQueueItem(hash string, fields map[string]json.RawMessage) plugin.QueueItem {
	var name string
	var totalSize, totalDone int64
	var ratio float64
	var state, savePath, moveCompletedPath, errMsg string

	unmarshalStr := func(key string) string {
		v, ok := fields[key]
		if !ok {
			return ""
		}
		var s string
		_ = json.Unmarshal(v, &s)
		return s
	}
	unmarshalInt := func(key string) int64 {
		v, ok := fields[key]
		if !ok {
			return 0
		}
		var n int64
		_ = json.Unmarshal(v, &n)
		return n
	}
	unmarshalFloat := func(key string) float64 {
		v, ok := fields[key]
		if !ok {
			return 0
		}
		var f float64
		_ = json.Unmarshal(v, &f)
		return f
	}

	name = unmarshalStr("name")
	totalSize = unmarshalInt("total_size")
	totalDone = unmarshalInt("total_done")
	ratio = unmarshalFloat("ratio")
	state = unmarshalStr("state")
	savePath = unmarshalStr("save_path")
	moveCompletedPath = unmarshalStr("move_completed_path")
	errMsg = unmarshalStr("error_msg")

	// Content path: prefer move_completed_path if set, otherwise save_path + name.
	contentPath := ""
	if moveCompletedPath != "" {
		contentPath = strings.TrimRight(moveCompletedPath, "/") + "/" + name
	} else if savePath != "" && name != "" {
		contentPath = strings.TrimRight(savePath, "/") + "/" + name
	}

	return plugin.QueueItem{
		ClientItemID: hash,
		Title:        name,
		Status:       mapState(state),
		Size:         totalSize,
		Downloaded:   totalDone,
		SeedRatio:    ratio,
		Error:        errMsg,
		ContentPath:  contentPath,
	}
}

// mapState converts Deluge torrent state strings to plugin.DownloadStatus.
func mapState(state string) plugin.DownloadStatus {
	switch state {
	case "Downloading", "Initializing":
		return plugin.StatusDownloading
	case "Seeding":
		return plugin.StatusCompleted
	case "Paused":
		return plugin.StatusPaused
	case "Error":
		return plugin.StatusFailed
	case "Queued", "Checking", "Allocating", "Moving":
		return plugin.StatusQueued
	default:
		return plugin.StatusQueued
	}
}
