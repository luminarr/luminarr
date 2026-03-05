// Package nzbget implements the plugin.DownloadClient interface for
// NZBGet's JSON-RPC API. Tested against NZBGet 21.x and nzbget.com forks.
package nzbget

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/davidfic/luminarr/internal/registry"
	"github.com/davidfic/luminarr/internal/safedialer"
	"github.com/davidfic/luminarr/pkg/plugin"
)

func init() {
	registry.Default.RegisterDownloader("nzbget", func(s json.RawMessage) (plugin.DownloadClient, error) {
		var cfg Config
		if err := json.Unmarshal(s, &cfg); err != nil {
			return nil, fmt.Errorf("nzbget: invalid settings: %w", err)
		}
		if cfg.URL == "" {
			return nil, errors.New("nzbget: url is required")
		}
		return New(cfg), nil
	})
	registry.Default.RegisterDownloaderSanitizer("nzbget", func(settings json.RawMessage) json.RawMessage {
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

// Config holds the connection settings for an NZBGet instance.
type Config struct {
	URL      string `json:"url"` // e.g. "http://localhost:6789"
	Username string `json:"username"`
	Password string `json:"password"`
	Category string `json:"category,omitempty"` // category applied to added NZBs
}

// Client implements plugin.DownloadClient against the NZBGet JSON-RPC API.
type Client struct {
	cfg    Config
	http   *http.Client
	rpcURL string // precomputed: {url}/jsonrpc
	id     atomic.Int64
}

// New creates a new NZBGet client.
func New(cfg Config) *Client {
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: 30 * time.Second, Transport: safedialer.LANTransport()},
		rpcURL: strings.TrimRight(cfg.URL, "/") + "/jsonrpc",
	}
}

// NewWithHTTPClient creates a Client with a caller-supplied http.Client.
// Intended for unit tests that need to bypass the safe dialer.
func NewWithHTTPClient(cfg Config, client *http.Client) *Client {
	return &Client{
		cfg:    cfg,
		http:   client,
		rpcURL: strings.TrimRight(cfg.URL, "/") + "/jsonrpc",
	}
}

func (c *Client) Name() string              { return "NZBGet" }
func (c *Client) Protocol() plugin.Protocol { return plugin.ProtocolNZB }

// Test verifies connectivity by calling the version method.
func (c *Client) Test(ctx context.Context) error {
	var ver string
	if err := c.call(ctx, "version", []any{}, &ver); err != nil {
		return fmt.Errorf("nzbget: connectivity check failed: %w", err)
	}
	if ver == "" {
		return errors.New("nzbget: version returned empty string")
	}
	return nil
}

// Add submits an NZB by URL to NZBGet. Returns the NZBID as a string.
func (c *Client) Add(ctx context.Context, r plugin.Release) (string, error) {
	// append params (positional, all required):
	// NZBFilename, NZBContent, Category, Priority, AddToTop, AddPaused,
	// DupeKey, DupeScore, DupeMode, PPParameters
	params := []any{
		"",             // NZBFilename (empty = NZBGet derives from URL)
		r.DownloadURL,  // NZBContent (URL — NZBGet fetches it)
		c.cfg.Category, // Category
		0,              // Priority (normal)
		false,          // AddToTop
		false,          // AddPaused
		"",             // DupeKey
		0,              // DupeScore
		"Score",        // DupeMode
		[]any{},        // PPParameters
	}

	var nzbID int
	if err := c.call(ctx, "append", params, &nzbID); err != nil {
		return "", fmt.Errorf("nzbget: append: %w", err)
	}
	if nzbID <= 0 {
		return "", errors.New("nzbget: append returned invalid NZBID")
	}
	return strconv.Itoa(nzbID), nil
}

// Status returns the state of a single item by NZBID.
// Checks the active queue first, then history.
func (c *Client) Status(ctx context.Context, clientItemID string) (plugin.QueueItem, error) {
	nzbID, err := strconv.Atoi(clientItemID)
	if err != nil {
		return plugin.QueueItem{}, fmt.Errorf("nzbget: invalid NZBID %q: %w", clientItemID, err)
	}

	// Check active queue.
	groups, err := c.listGroups(ctx)
	if err != nil {
		return plugin.QueueItem{}, err
	}
	for _, g := range groups {
		if g.NZBID == nzbID {
			return g.toQueueItem(), nil
		}
	}

	// Check history.
	history, err := c.listHistory(ctx)
	if err != nil {
		return plugin.QueueItem{}, err
	}
	for _, h := range history {
		if h.NZBID == nzbID {
			return h.toQueueItem(), nil
		}
	}

	return plugin.QueueItem{}, fmt.Errorf("nzbget: item %q not found in queue or history", clientItemID)
}

// GetQueue returns all items from the active queue plus recent history.
func (c *Client) GetQueue(ctx context.Context) ([]plugin.QueueItem, error) {
	groups, err := c.listGroups(ctx)
	if err != nil {
		return nil, err
	}
	history, err := c.listHistory(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]plugin.QueueItem, 0, len(groups)+len(history))
	for _, g := range groups {
		items = append(items, g.toQueueItem())
	}
	for _, h := range history {
		items = append(items, h.toQueueItem())
	}
	return items, nil
}

// Remove deletes an item from NZBGet. Tries queue first, then history.
func (c *Client) Remove(ctx context.Context, clientItemID string, _ bool) error {
	nzbID, err := strconv.Atoi(clientItemID)
	if err != nil {
		return fmt.Errorf("nzbget: invalid NZBID %q: %w", clientItemID, err)
	}

	// Try queue removal first.
	var ok bool
	if err := c.call(ctx, "editqueue", []any{"GroupFinalDelete", "", []int{nzbID}}, &ok); err != nil {
		return fmt.Errorf("nzbget: editqueue: %w", err)
	}
	if ok {
		return nil
	}

	// Try history removal.
	if err := c.call(ctx, "editqueue", []any{"HistoryFinalDelete", "", []int{nzbID}}, &ok); err != nil {
		return fmt.Errorf("nzbget: editqueue history: %w", err)
	}
	if !ok {
		return fmt.Errorf("nzbget: item %q not found for removal", clientItemID)
	}
	return nil
}

// ── JSON-RPC helper ──────────────────────────────────────────────────────────

type rpcRequest struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
	ID     int64  `json:"id"`
}

type rpcResponse struct {
	Version string          `json:"version"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.rpcURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("nzbget: authentication failed — check username and password")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nzbget: unexpected HTTP %d", resp.StatusCode)
	}

	var rpc rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpc); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if rpc.Error != nil {
		return fmt.Errorf("RPC error %d: %s", rpc.Error.Code, rpc.Error.Message)
	}
	if out != nil && rpc.Result != nil {
		if err := json.Unmarshal(rpc.Result, out); err != nil {
			return fmt.Errorf("decoding result: %w", err)
		}
	}
	return nil
}

// ── API call wrappers ────────────────────────────────────────────────────────

func (c *Client) listGroups(ctx context.Context) ([]groupItem, error) {
	var groups []groupItem
	if err := c.call(ctx, "listgroups", []any{0}, &groups); err != nil {
		return nil, fmt.Errorf("nzbget: listgroups: %w", err)
	}
	return groups, nil
}

func (c *Client) listHistory(ctx context.Context) ([]historyItem, error) {
	var history []historyItem
	if err := c.call(ctx, "history", []any{false}, &history); err != nil {
		return nil, fmt.Errorf("nzbget: history: %w", err)
	}
	return history, nil
}

// ── Response types ───────────────────────────────────────────────────────────

type groupItem struct {
	NZBID           int    `json:"NZBID"`
	NZBName         string `json:"NZBName"`
	Status          string `json:"Status"`
	FileSizeMB      int64  `json:"FileSizeMB"`
	RemainingSizeMB int64  `json:"RemainingSizeMB"`
	DestDir         string `json:"DestDir"`
	Category        string `json:"Category"`
}

func (g groupItem) toQueueItem() plugin.QueueItem {
	totalBytes := g.FileSizeMB * 1024 * 1024
	remainingBytes := g.RemainingSizeMB * 1024 * 1024
	return plugin.QueueItem{
		ClientItemID: strconv.Itoa(g.NZBID),
		Title:        g.NZBName,
		Status:       mapGroupStatus(g.Status),
		Size:         totalBytes,
		Downloaded:   totalBytes - remainingBytes,
		ContentPath:  g.DestDir,
	}
}

type historyItem struct {
	NZBID        int    `json:"NZBID"`
	Name         string `json:"Name"`
	Status       string `json:"Status"`
	FileSizeMB   int64  `json:"FileSizeMB"`
	DownloadedMB int64  `json:"DownloadedSizeMB"`
	DestDir      string `json:"DestDir"`
	FinalDir     string `json:"FinalDir"`
	HistoryTime  int64  `json:"HistoryTime"`
	Category     string `json:"Category"`
}

func (h historyItem) toQueueItem() plugin.QueueItem {
	contentPath := h.FinalDir
	if contentPath == "" {
		contentPath = h.DestDir
	}
	return plugin.QueueItem{
		ClientItemID: strconv.Itoa(h.NZBID),
		Title:        h.Name,
		Status:       mapHistoryStatus(h.Status),
		Size:         h.FileSizeMB * 1024 * 1024,
		Downloaded:   h.DownloadedMB * 1024 * 1024,
		ContentPath:  contentPath,
		AddedAt:      h.HistoryTime,
	}
}

// ── Status mapping ───────────────────────────────────────────────────────────

func mapGroupStatus(status string) plugin.DownloadStatus {
	switch status {
	case "DOWNLOADING":
		return plugin.StatusDownloading
	case "PAUSED":
		return plugin.StatusPaused
	case "QUEUED", "FETCHING", "LOADING_PARS":
		return plugin.StatusQueued
	case "PP_QUEUED", "VERIFYING_SOURCES", "REPAIRING", "VERIFYING_REPAIRED",
		"RENAMING", "UNPACKING", "MOVING", "EXECUTING_SCRIPT":
		return plugin.StatusDownloading
	case "PP_FINISHED":
		return plugin.StatusCompleted
	default:
		return plugin.StatusQueued
	}
}

func mapHistoryStatus(status string) plugin.DownloadStatus {
	if strings.HasPrefix(status, "SUCCESS") {
		return plugin.StatusCompleted
	}
	if strings.HasPrefix(status, "FAILURE") || strings.HasPrefix(status, "WARNING") || strings.HasPrefix(status, "DELETED") {
		return plugin.StatusFailed
	}
	return plugin.StatusQueued
}
