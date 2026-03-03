// Package qbittorrent implements the plugin.DownloadClient interface for qBittorrent Web API v2.
package qbittorrent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/internal/safedialer"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterDownloader("qbittorrent", func(s json.RawMessage) (plugin.DownloadClient, error) {
		var cfg Config
		if err := json.Unmarshal(s, &cfg); err != nil {
			return nil, fmt.Errorf("qbittorrent: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, errors.New("qbittorrent: url is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterDownloaderSanitizer("qbittorrent", func(settings json.RawMessage) json.RawMessage {
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

// Config holds the connection settings for a qBittorrent instance.
type Config struct {
	URL      string `json:"url"` // e.g. "http://localhost:8080"
	Username string `json:"username"`
	Password string `json:"password"`
	Category string `json:"category,omitempty"` // label applied to added torrents
	SavePath string `json:"save_path,omitempty"`
}

// Client implements plugin.DownloadClient against the qBittorrent Web API v2.
type Client struct {
	cfg    Config
	http   *http.Client
	mu     sync.Mutex
	authed bool
}

// New creates a new qBittorrent client. Call Test to verify connectivity.
// Outbound HTTP uses safedialer.LANTransport() because download clients are
// typically hosted on localhost or a LAN address. It blocks cloud-metadata
// ranges (169.254.0.0/16 etc.) but allows RFC-1918 and loopback.
// For unit tests that use httptest.Server, use NewWithHTTPClient.
func New(cfg Config) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		cfg:  cfg,
		http: &http.Client{Jar: jar, Timeout: 30 * time.Second, Transport: safedialer.LANTransport()},
	}
}

// NewWithHTTPClient creates a Client with a caller-supplied http.Client.
// Intended for unit tests that need to bypass the SSRF-blocking safe dialer
// and connect to httptest.Server instances on 127.0.0.1.
func NewWithHTTPClient(cfg Config, client *http.Client) *Client {
	return &Client{cfg: cfg, http: client}
}

func (c *Client) Name() string              { return "qBittorrent" }
func (c *Client) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }

// Test verifies connectivity and authentication to the qBittorrent instance.
func (c *Client) Test(ctx context.Context) error {
	if err := c.login(ctx); err != nil {
		return err
	}
	resp, err := c.get(ctx, "/api/v2/app/version")
	if err != nil {
		return fmt.Errorf("qbittorrent: connectivity check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbittorrent: unexpected status %d from app/version", resp.StatusCode)
	}
	return nil
}

// Add submits a torrent to qBittorrent. For magnet links the hash is parsed
// directly from the URI. For HTTP .torrent URLs the torrent is added and the
// most recently added torrent hash is returned (best-effort; see TODO.md).
func (c *Client) Add(ctx context.Context, r plugin.Release) (string, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return "", err
	}

	if strings.HasPrefix(r.DownloadURL, "magnet:") {
		return c.addMagnet(ctx, r.DownloadURL)
	}
	return c.addTorrentURL(ctx, r.DownloadURL)
}

// Status returns the current state of a single torrent by its hash.
func (c *Client) Status(ctx context.Context, clientItemID string) (plugin.QueueItem, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return plugin.QueueItem{}, err
	}

	resp, err := c.get(ctx, "/api/v2/torrents/info?hashes="+url.QueryEscape(clientItemID))
	if err != nil {
		return plugin.QueueItem{}, fmt.Errorf("qbittorrent: status request failed: %w", err)
	}
	defer resp.Body.Close()

	var torrents []torrentInfo
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return plugin.QueueItem{}, fmt.Errorf("qbittorrent: decoding status response: %w", err)
	}
	if len(torrents) == 0 {
		return plugin.QueueItem{}, fmt.Errorf("torrent %q not found in qBittorrent", clientItemID)
	}
	return torrents[0].toQueueItem(), nil
}

// GetQueue returns all torrents currently tracked by qBittorrent.
func (c *Client) GetQueue(ctx context.Context) ([]plugin.QueueItem, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/api/v2/torrents/info")
	if err != nil {
		return nil, fmt.Errorf("qbittorrent: queue request failed: %w", err)
	}
	defer resp.Body.Close()

	var torrents []torrentInfo
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, fmt.Errorf("qbittorrent: decoding queue response: %w", err)
	}

	items := make([]plugin.QueueItem, len(torrents))
	for i, t := range torrents {
		items[i] = t.toQueueItem()
	}
	return items, nil
}

// Remove deletes a torrent from qBittorrent. If deleteFiles is true the
// downloaded data is also removed from disk.
func (c *Client) Remove(ctx context.Context, clientItemID string, deleteFiles bool) error {
	if err := c.ensureAuth(ctx); err != nil {
		return err
	}

	deleteFStr := "false"
	if deleteFiles {
		deleteFStr = "true"
	}

	form := url.Values{
		"hashes":      {clientItemID},
		"deleteFiles": {deleteFStr},
	}
	resp, err := c.post(ctx, "/api/v2/torrents/delete", form)
	if err != nil {
		return fmt.Errorf("qbittorrent: remove request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbittorrent: remove returned status %d", resp.StatusCode)
	}
	return nil
}

// ── Auth ─────────────────────────────────────────────────────────────────────

func (c *Client) ensureAuth(ctx context.Context) error {
	c.mu.Lock()
	authed := c.authed
	c.mu.Unlock()
	if authed {
		return nil
	}
	return c.login(ctx)
}

func (c *Client) login(ctx context.Context) error {
	form := url.Values{
		"username": {c.cfg.Username},
		"password": {c.cfg.Password},
	}
	resp, err := c.post(ctx, "/api/v2/auth/login", form)
	if err != nil {
		return fmt.Errorf("qbittorrent: login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10)) // 1 KiB — response is "Ok." or "Fails."
	bodyStr := strings.TrimSpace(string(body))

	if resp.StatusCode != http.StatusOK || bodyStr == "Fails." {
		return errors.New("qbittorrent: authentication failed — check username and password")
	}

	c.mu.Lock()
	c.authed = true
	c.mu.Unlock()
	return nil
}

// ── Torrent add helpers ───────────────────────────────────────────────────────

func (c *Client) addMagnet(ctx context.Context, magnetURL string) (string, error) {
	hash, ok := parseMagnetHash(magnetURL)
	if !ok {
		return "", errors.New("qbittorrent: could not parse info hash from magnet link")
	}

	form := c.addFormBase()
	form.Set("urls", magnetURL)

	resp, err := c.post(ctx, "/api/v2/torrents/add", form)
	if err != nil {
		return "", fmt.Errorf("qbittorrent: add magnet failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10)) // 1 KiB — response is "Ok." or error message
	if strings.TrimSpace(string(body)) != "Ok." {
		return "", fmt.Errorf("qbittorrent: add magnet returned %q", string(body))
	}
	return hash, nil
}

// errMagnetRedirect signals that the torrent URL redirected to a magnet link.
type errMagnetRedirect struct{ magnetURL string }

func (e errMagnetRedirect) Error() string { return "magnet redirect: " + e.magnetURL }

func (c *Client) addTorrentURL(ctx context.Context, torrentURL string) (string, error) {
	// Download the .torrent file using Luminarr's own HTTP client so that
	// qBittorrent does not need network access to the indexer/Prowlarr URL.
	torrentData, err := c.fetchURL(ctx, torrentURL)
	if err != nil {
		// Some indexers redirect their download URLs to magnet links.
		// Prowlarr does this for public trackers. Handle it transparently.
		var mag errMagnetRedirect
		if errors.As(err, &mag) {
			return c.addMagnet(ctx, mag.magnetURL)
		}
		return "", fmt.Errorf("qbittorrent: downloading torrent file: %w", err)
	}

	// Record the time before adding so we can identify the newly added torrent.
	before := time.Now().Add(-2 * time.Second) // small buffer for clock skew

	resp, err := c.uploadTorrentFile(ctx, torrentData)
	if err != nil {
		return "", fmt.Errorf("qbittorrent: uploading torrent file: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10)) // 1 KiB — response is "Ok." or error message
	if strings.TrimSpace(string(body)) != "Ok." {
		return "", fmt.Errorf("qbittorrent: add torrent returned %q", string(body))
	}

	// Poll for the torrent to appear. Best-effort: if the hash can't be
	// identified we still consider the add successful.
	hash, err := c.waitForRecentTorrent(ctx, before)
	if err != nil {
		return "", nil
	}
	return hash, nil
}

// fetchURL downloads content from a URL using a plain (unauthenticated) HTTP
// client. If any redirect in the chain points to a magnet: URI, it returns
// errMagnetRedirect instead of following it (Go's HTTP client cannot do so).
func (c *Client) fetchURL(ctx context.Context, rawURL string) ([]byte, error) {
	client := &http.Client{
		Transport: safedialer.Transport(),
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if strings.HasPrefix(req.URL.String(), "magnet:") {
				return errMagnetRedirect{magnetURL: req.URL.String()}
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		// Unwrap *url.Error to surface errMagnetRedirect.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			var mag errMagnetRedirect
			if errors.As(urlErr.Err, &mag) {
				return nil, mag
			}
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching torrent", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MiB — same limit as Deluge plugin
}

// uploadTorrentFile posts torrent file bytes to qBittorrent as a multipart upload.
func (c *Client) uploadTorrentFile(ctx context.Context, data []byte) (*http.Response, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// Optional category / save path fields.
	if c.cfg.Category != "" {
		_ = mw.WriteField("category", c.cfg.Category)
	}
	if c.cfg.SavePath != "" {
		_ = mw.WriteField("savepath", c.cfg.SavePath)
	}

	fw, err := mw.CreateFormFile("torrents", "file.torrent")
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(data); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.URL+"/api/v2/torrents/add", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.http.Do(req)
}

// waitForRecentTorrent polls the torrent list until a torrent added after
// `since` appears, returning its hash. Times out after 5 seconds.
func (c *Client) waitForRecentTorrent(ctx context.Context, since time.Time) (string, error) {
	sinceUnix := since.Unix()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		time.Sleep(500 * time.Millisecond)

		items, err := c.GetQueue(ctx)
		if err != nil {
			continue
		}
		// Find the most recently added torrent that appeared after `since`.
		// AddedAt is the qBittorrent added_on Unix timestamp.
		//
		// NOTE: This is a best-effort heuristic. It can misidentify the torrent
		// if multiple grabs happen concurrently. See TODO.md for the bencode
		// hash extraction approach that would make this deterministic.
		var best *plugin.QueueItem
		for i := range items {
			item := &items[i]
			if item.AddedAt < sinceUnix {
				continue // torrent predates this add
			}
			if best == nil || item.AddedAt > best.AddedAt {
				best = item
			}
		}
		if best != nil {
			return best.ClientItemID, nil
		}
	}
	return "", errors.New("qbittorrent: timed out waiting for torrent to appear after add")
}

func (c *Client) addFormBase() url.Values {
	form := url.Values{}
	if c.cfg.Category != "" {
		form.Set("category", c.cfg.Category)
	}
	if c.cfg.SavePath != "" {
		form.Set("savepath", c.cfg.SavePath)
	}
	return form
}

// ── HTTP helpers ─────────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		c.mu.Lock()
		c.authed = false
		c.mu.Unlock()
		return nil, errors.New("qbittorrent: session expired — re-authentication required")
	}
	return resp, nil
}

func (c *Client) post(ctx context.Context, path string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.URL+path,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.http.Do(req)
}

// ── Torrent state types ───────────────────────────────────────────────────────

// torrentInfo is the JSON shape returned by /api/v2/torrents/info.
type torrentInfo struct {
	Hash        string  `json:"hash"`
	Name        string  `json:"name"`
	Size        int64   `json:"size"`
	Downloaded  int64   `json:"downloaded"`
	Ratio       float64 `json:"ratio"`
	State       string  `json:"state"`
	Error       string  `json:"tracker_msg,omitempty"`
	AddedOn     int64   `json:"added_on"`
	SavePath    string  `json:"save_path"`
	ContentPath string  `json:"content_path"` // added in qBittorrent 4.3.2
}

func (t torrentInfo) toQueueItem() plugin.QueueItem {
	// content_path is the precise path to content (file or directory).
	// Fall back to save_path + name for older qBittorrent versions.
	contentPath := t.ContentPath
	if contentPath == "" && t.SavePath != "" {
		contentPath = strings.TrimRight(t.SavePath, "/") + "/" + t.Name
	}
	return plugin.QueueItem{
		ClientItemID: t.Hash,
		Title:        t.Name,
		Status:       mapState(t.State),
		Size:         t.Size,
		Downloaded:   t.Downloaded,
		SeedRatio:    t.Ratio,
		Error:        t.Error,
		ContentPath:  contentPath,
		AddedAt:      t.AddedOn,
	}
}

// mapState converts qBittorrent state strings to plugin.DownloadStatus.
// See: https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)
func mapState(state string) plugin.DownloadStatus {
	switch state {
	case "downloading", "metaDL", "forcedDL":
		return plugin.StatusDownloading
	case "stalledDL", "checkingDL", "allocating", "queuedDL":
		return plugin.StatusQueued
	case "uploading", "stalledUP", "checkingUP", "queuedUP", "forcedUP", "seeding":
		return plugin.StatusCompleted
	case "pausedDL", "stoppedDL":
		return plugin.StatusPaused
	case "error", "missingFiles", "unknown", "pausedUP", "stoppedUP":
		return plugin.StatusFailed
	default:
		return plugin.StatusQueued
	}
}

// ── Magnet hash parsing ───────────────────────────────────────────────────────

// parseMagnetHash extracts the lowercase hex info hash from a magnet URI.
// Handles both 40-char hex and 32-char base32 btih encodings.
func parseMagnetHash(magnetURL string) (string, bool) {
	u, err := url.Parse(magnetURL)
	if err != nil {
		return "", false
	}
	xt := u.Query().Get("xt")
	const prefix = "urn:btih:"
	if !strings.HasPrefix(xt, prefix) {
		return "", false
	}
	raw := strings.ToLower(xt[len(prefix):])
	if len(raw) == 40 {
		// Standard 40-char hex SHA1.
		return raw, true
	}
	if len(raw) == 32 {
		// Base32-encoded hash — decode then re-encode as hex.
		decoded, err := base32Decode(strings.ToUpper(raw))
		if err != nil {
			return "", false
		}
		return fmt.Sprintf("%x", decoded), true
	}
	return "", false
}

// base32Decode decodes a standard base32 string (no padding required).
func base32Decode(s string) ([]byte, error) {
	// Pad to a multiple of 8.
	for len(s)%8 != 0 {
		s += "="
	}
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	out := make([]byte, 0, len(s)*5/8)
	var buf uint64
	var bits int
	for _, ch := range s {
		if ch == '=' {
			break
		}
		idx := strings.IndexRune(alphabet, ch)
		if idx < 0 {
			return nil, fmt.Errorf("invalid base32 character %q", ch)
		}
		buf = buf<<5 | uint64(idx)
		bits += 5
		if bits >= 8 {
			bits -= 8
			out = append(out, byte(buf>>uint(bits))) //nolint:gosec // G115: bits is 0-7 after subtraction, safe
			buf &= (1 << uint(bits)) - 1             //nolint:gosec // G115: bits is 0-7, safe
		}
	}
	return out, nil
}
