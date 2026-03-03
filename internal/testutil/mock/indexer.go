// Package mock provides hand-written mock implementations of Luminarr's plugin
// interfaces for use in unit and integration tests.
package mock

import (
	"context"

	"github.com/davidfic/luminarr/pkg/plugin"
)

// Indexer is a configurable mock of plugin.Indexer.
// Set the Func fields to control return values; leave nil to return zero values.
// Calls records the names of methods that were invoked.
type Indexer struct {
	CapabilitiesFunc func(ctx context.Context) (plugin.Capabilities, error)
	SearchFunc       func(ctx context.Context, q plugin.SearchQuery) ([]plugin.Release, error)
	GetRecentFunc    func(ctx context.Context) ([]plugin.Release, error)
	TestFunc         func(ctx context.Context) error

	Calls []string
}

func (m *Indexer) Name() string              { return "MockIndexer" }
func (m *Indexer) Protocol() plugin.Protocol { return plugin.ProtocolTorrent }

func (m *Indexer) Capabilities(ctx context.Context) (plugin.Capabilities, error) {
	m.Calls = append(m.Calls, "Capabilities")
	if m.CapabilitiesFunc != nil {
		return m.CapabilitiesFunc(ctx)
	}
	return plugin.Capabilities{SearchAvailable: true, MovieSearch: true}, nil
}

func (m *Indexer) Search(ctx context.Context, q plugin.SearchQuery) ([]plugin.Release, error) {
	m.Calls = append(m.Calls, "Search")
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, q)
	}
	return nil, nil
}

func (m *Indexer) GetRecent(ctx context.Context) ([]plugin.Release, error) {
	m.Calls = append(m.Calls, "GetRecent")
	if m.GetRecentFunc != nil {
		return m.GetRecentFunc(ctx)
	}
	return nil, nil
}

func (m *Indexer) Test(ctx context.Context) error {
	m.Calls = append(m.Calls, "Test")
	if m.TestFunc != nil {
		return m.TestFunc(ctx)
	}
	return nil
}
