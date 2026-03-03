// Package registry holds plugin factories and instantiates plugin instances
// from stored JSON configuration. Built-in plugins self-register via init().
package registry

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/davidfic/luminarr/pkg/plugin"
)

// IndexerFactory constructs an Indexer from a JSON settings blob.
type IndexerFactory func(settings json.RawMessage) (plugin.Indexer, error)

// DownloaderFactory constructs a DownloadClient from a JSON settings blob.
type DownloaderFactory func(settings json.RawMessage) (plugin.DownloadClient, error)

// NotifierFactory constructs a Notifier from a JSON settings blob.
type NotifierFactory func(settings json.RawMessage) (plugin.Notifier, error)

// SanitizerFunc redacts sensitive fields from a plugin settings blob so it is
// safe to include in API responses and logs. It must never return nil.
// Plugins register one via RegisterIndexerSanitizer / RegisterDownloaderSanitizer /
// RegisterNotifierSanitizer. If no sanitizer is registered the registry falls
// back to returning an empty JSON object so credentials are never exposed.
type SanitizerFunc func(settings json.RawMessage) json.RawMessage

// emptySanitizer is the safe fallback when no plugin sanitizer is registered.
var emptySanitizer SanitizerFunc = func(_ json.RawMessage) json.RawMessage {
	return json.RawMessage("{}")
}

// Registry maps plugin kind strings to their factory functions.
// Use Default for the application-wide singleton.
type Registry struct {
	mu                   sync.RWMutex
	indexers             map[string]IndexerFactory
	indexerSanitizers    map[string]SanitizerFunc
	downloaders          map[string]DownloaderFactory
	downloaderSanitizers map[string]SanitizerFunc
	notifiers            map[string]NotifierFactory
	notifierSanitizers   map[string]SanitizerFunc
}

// New returns an empty, ready-to-use Registry.
func New() *Registry {
	return &Registry{
		indexers:             make(map[string]IndexerFactory),
		indexerSanitizers:    make(map[string]SanitizerFunc),
		downloaders:          make(map[string]DownloaderFactory),
		downloaderSanitizers: make(map[string]SanitizerFunc),
		notifiers:            make(map[string]NotifierFactory),
		notifierSanitizers:   make(map[string]SanitizerFunc),
	}
}

// Default is the application-wide plugin registry.
// Built-in plugins register themselves here in their init() functions.
var Default = New()

// RegisterIndexer adds a factory for the given kind string.
// Panics if kind is already registered (caught at startup, not runtime).
func (r *Registry) RegisterIndexer(kind string, factory IndexerFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.indexers[kind]; exists {
		panic(fmt.Sprintf("registry: indexer kind %q already registered", kind))
	}
	r.indexers[kind] = factory
}

// RegisterIndexerSanitizer registers a settings sanitizer for the given indexer
// kind. Call this from the same init() as RegisterIndexer.
func (r *Registry) RegisterIndexerSanitizer(kind string, fn SanitizerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.indexerSanitizers[kind] = fn
}

// SanitizeIndexerSettings returns a redacted copy of the settings blob safe for
// API responses and logs. Falls back to "{}" if no sanitizer is registered.
func (r *Registry) SanitizeIndexerSettings(kind string, settings json.RawMessage) json.RawMessage {
	r.mu.RLock()
	fn, ok := r.indexerSanitizers[kind]
	r.mu.RUnlock()
	if !ok {
		return emptySanitizer(settings)
	}
	return fn(settings)
}

// NewIndexer constructs an Indexer from the given kind and JSON settings.
// Returns an error if the kind is unknown or settings are invalid.
func (r *Registry) NewIndexer(kind string, settings json.RawMessage) (plugin.Indexer, error) {
	r.mu.RLock()
	factory, ok := r.indexers[kind]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown indexer kind %q", kind)
	}
	return factory(settings)
}

// IndexerKinds returns the list of registered indexer kind strings.
func (r *Registry) IndexerKinds() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	kinds := make([]string, 0, len(r.indexers))
	for k := range r.indexers {
		kinds = append(kinds, k)
	}
	return kinds
}

// RegisterDownloader adds a factory for the given download client kind string.
// Panics if kind is already registered (caught at startup, not runtime).
func (r *Registry) RegisterDownloader(kind string, factory DownloaderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.downloaders[kind]; exists {
		panic(fmt.Sprintf("registry: downloader kind %q already registered", kind))
	}
	r.downloaders[kind] = factory
}

// RegisterDownloaderSanitizer registers a settings sanitizer for the given
// download client kind.
func (r *Registry) RegisterDownloaderSanitizer(kind string, fn SanitizerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.downloaderSanitizers[kind] = fn
}

// SanitizeDownloaderSettings returns a redacted copy of the settings blob safe
// for API responses and logs. Falls back to "{}" if no sanitizer is registered.
func (r *Registry) SanitizeDownloaderSettings(kind string, settings json.RawMessage) json.RawMessage {
	r.mu.RLock()
	fn, ok := r.downloaderSanitizers[kind]
	r.mu.RUnlock()
	if !ok {
		return emptySanitizer(settings)
	}
	return fn(settings)
}

// NewDownloader constructs a DownloadClient from the given kind and JSON settings.
// Returns an error if the kind is unknown or settings are invalid.
func (r *Registry) NewDownloader(kind string, settings json.RawMessage) (plugin.DownloadClient, error) {
	r.mu.RLock()
	factory, ok := r.downloaders[kind]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown downloader kind %q", kind)
	}
	return factory(settings)
}

// DownloaderKinds returns the list of registered downloader kind strings.
func (r *Registry) DownloaderKinds() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	kinds := make([]string, 0, len(r.downloaders))
	for k := range r.downloaders {
		kinds = append(kinds, k)
	}
	return kinds
}

// RegisterNotifier adds a factory for the given notifier kind string.
// Panics if kind is already registered (caught at startup, not runtime).
func (r *Registry) RegisterNotifier(kind string, factory NotifierFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.notifiers[kind]; exists {
		panic(fmt.Sprintf("registry: notifier kind %q already registered", kind))
	}
	r.notifiers[kind] = factory
}

// RegisterNotifierSanitizer registers a settings sanitizer for the given
// notifier kind.
func (r *Registry) RegisterNotifierSanitizer(kind string, fn SanitizerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifierSanitizers[kind] = fn
}

// SanitizeNotifierSettings returns a redacted copy of the settings blob safe
// for API responses and logs. Falls back to "{}" if no sanitizer is registered.
func (r *Registry) SanitizeNotifierSettings(kind string, settings json.RawMessage) json.RawMessage {
	r.mu.RLock()
	fn, ok := r.notifierSanitizers[kind]
	r.mu.RUnlock()
	if !ok {
		return emptySanitizer(settings)
	}
	return fn(settings)
}

// NewNotifier constructs a Notifier from the given kind and JSON settings.
// Returns an error if the kind is unknown or settings are invalid.
func (r *Registry) NewNotifier(kind string, settings json.RawMessage) (plugin.Notifier, error) {
	r.mu.RLock()
	factory, ok := r.notifiers[kind]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown notifier kind %q", kind)
	}
	return factory(settings)
}

// NotifierKinds returns the list of registered notifier kind strings.
func (r *Registry) NotifierKinds() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	kinds := make([]string, 0, len(r.notifiers))
	for k := range r.notifiers {
		kinds = append(kinds, k)
	}
	return kinds
}
