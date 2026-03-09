package v3

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/danielgtaylor/huma/v2"
)

// tagStore is a simple in-memory tag store for Radarr v3 compatibility.
// Tags are not persisted — they exist only for the lifetime of the process.
// This is sufficient for Overseerr which creates tags on add and passes them
// along, but doesn't depend on persistence.
type tagStore struct {
	mu   sync.RWMutex
	tags []RadarrTag
	next atomic.Int64
}

func newTagStore() *tagStore {
	s := &tagStore{}
	s.next.Store(1)
	return s
}

func (s *tagStore) list() []RadarrTag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]RadarrTag, len(s.tags))
	copy(result, s.tags)
	return result
}

func (s *tagStore) create(label string) RadarrTag {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return existing if label matches.
	for _, t := range s.tags {
		if t.Label == label {
			return t
		}
	}

	tag := RadarrTag{
		ID:    s.next.Add(1) - 1,
		Label: label,
	}
	s.tags = append(s.tags, tag)
	return tag
}

func registerTagRoutes(api huma.API) *tagStore {
	store := newTagStore()

	huma.Register(api, huma.Operation{
		OperationID: "radarr-list-tags",
		Method:      http.MethodGet,
		Path:        "/api/v3/tag",
		Summary:     "List tags (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(_ context.Context, _ *struct{}) (*struct{ Body []RadarrTag }, error) {
		return &struct{ Body []RadarrTag }{Body: store.list()}, nil
	})

	type createTagInput struct {
		Body struct {
			Label string `json:"label" required:"true"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "radarr-create-tag",
		Method:      http.MethodPost,
		Path:        "/api/v3/tag",
		Summary:     "Create tag (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(_ context.Context, input *createTagInput) (*struct {
		Status int
		Body   RadarrTag
	}, error) {
		tag := store.create(input.Body.Label)
		return &struct {
			Status int
			Body   RadarrTag
		}{Status: http.StatusCreated, Body: tag}, nil
	})

	return store
}
