package mock

import (
	"context"
	"sync"

	"github.com/luminarr/luminarr/pkg/plugin"
)

// DownloadClient is a configurable mock of plugin.DownloadClient.
type DownloadClient struct {
	AddFunc           func(ctx context.Context, r plugin.Release) (string, error)
	StatusFunc        func(ctx context.Context, clientItemID string) (plugin.QueueItem, error)
	GetQueueFunc      func(ctx context.Context) ([]plugin.QueueItem, error)
	RemoveFunc        func(ctx context.Context, clientItemID string, deleteFiles bool) error
	TestFunc          func(ctx context.Context) error
	SetSeedLimitsFunc func(ctx context.Context, clientItemID string, ratioLimit float64, seedTimeSecs int) error

	mu    sync.Mutex
	Calls []string
}

func (m *DownloadClient) recordCall(name string) {
	m.mu.Lock()
	m.Calls = append(m.Calls, name)
	m.mu.Unlock()
}

// GetCalls returns a snapshot of recorded calls (thread-safe).
func (m *DownloadClient) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.Calls))
	copy(out, m.Calls)
	return out
}

func (m *DownloadClient) Name() string              { return "MockDownloadClient" }
func (m *DownloadClient) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }

func (m *DownloadClient) Add(ctx context.Context, r plugin.Release) (string, error) {
	m.recordCall("Add")
	if m.AddFunc != nil {
		return m.AddFunc(ctx, r)
	}
	return "mock-item-id", nil
}

func (m *DownloadClient) Status(ctx context.Context, clientItemID string) (plugin.QueueItem, error) {
	m.recordCall("Status")
	if m.StatusFunc != nil {
		return m.StatusFunc(ctx, clientItemID)
	}
	return plugin.QueueItem{ClientItemID: clientItemID, Status: plugin.StatusDownloading}, nil
}

func (m *DownloadClient) GetQueue(ctx context.Context) ([]plugin.QueueItem, error) {
	m.recordCall("GetQueue")
	if m.GetQueueFunc != nil {
		return m.GetQueueFunc(ctx)
	}
	return nil, nil
}

func (m *DownloadClient) Remove(ctx context.Context, clientItemID string, deleteFiles bool) error {
	m.recordCall("Remove")
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, clientItemID, deleteFiles)
	}
	return nil
}

func (m *DownloadClient) Test(ctx context.Context) error {
	m.recordCall("Test")
	if m.TestFunc != nil {
		return m.TestFunc(ctx)
	}
	return nil
}

func (m *DownloadClient) SetSeedLimits(ctx context.Context, clientItemID string, ratioLimit float64, seedTimeSecs int) error {
	m.recordCall("SetSeedLimits")
	if m.SetSeedLimitsFunc != nil {
		return m.SetSeedLimitsFunc(ctx, clientItemID, ratioLimit, seedTimeSecs)
	}
	return nil
}
