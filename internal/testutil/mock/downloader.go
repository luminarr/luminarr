package mock

import (
	"context"

	"github.com/davidfic/luminarr/pkg/plugin"
)

// DownloadClient is a configurable mock of plugin.DownloadClient.
type DownloadClient struct {
	AddFunc      func(ctx context.Context, r plugin.Release) (string, error)
	StatusFunc   func(ctx context.Context, clientItemID string) (plugin.QueueItem, error)
	GetQueueFunc func(ctx context.Context) ([]plugin.QueueItem, error)
	RemoveFunc   func(ctx context.Context, clientItemID string, deleteFiles bool) error
	TestFunc     func(ctx context.Context) error

	Calls []string
}

func (m *DownloadClient) Name() string              { return "MockDownloadClient" }
func (m *DownloadClient) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }

func (m *DownloadClient) Add(ctx context.Context, r plugin.Release) (string, error) {
	m.Calls = append(m.Calls, "Add")
	if m.AddFunc != nil {
		return m.AddFunc(ctx, r)
	}
	return "mock-item-id", nil
}

func (m *DownloadClient) Status(ctx context.Context, clientItemID string) (plugin.QueueItem, error) {
	m.Calls = append(m.Calls, "Status")
	if m.StatusFunc != nil {
		return m.StatusFunc(ctx, clientItemID)
	}
	return plugin.QueueItem{ClientItemID: clientItemID, Status: plugin.StatusDownloading}, nil
}

func (m *DownloadClient) GetQueue(ctx context.Context) ([]plugin.QueueItem, error) {
	m.Calls = append(m.Calls, "GetQueue")
	if m.GetQueueFunc != nil {
		return m.GetQueueFunc(ctx)
	}
	return nil, nil
}

func (m *DownloadClient) Remove(ctx context.Context, clientItemID string, deleteFiles bool) error {
	m.Calls = append(m.Calls, "Remove")
	if m.RemoveFunc != nil {
		return m.RemoveFunc(ctx, clientItemID, deleteFiles)
	}
	return nil
}

func (m *DownloadClient) Test(ctx context.Context) error {
	m.Calls = append(m.Calls, "Test")
	if m.TestFunc != nil {
		return m.TestFunc(ctx)
	}
	return nil
}
