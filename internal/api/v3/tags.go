package v3

import (
	"context"
	"net/http"
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/luminarr/luminarr/internal/core/tag"
)

// tagMapper provides a stable bidirectional mapping between our UUID-based tag
// IDs and the integer IDs that Radarr v3 clients (Overseerr, Homepage) expect.
// The mapping is rebuilt lazily from the real tag service and persists for the
// process lifetime. Integer IDs are assigned sequentially starting from 1.
type tagMapper struct {
	mu      sync.RWMutex
	uuidInt map[string]int64 // UUID → integer ID
	intUUID map[int64]string // integer ID → UUID
	next    int64
}

func newTagMapper() *tagMapper {
	return &tagMapper{
		uuidInt: make(map[string]int64),
		intUUID: make(map[int64]string),
		next:    1,
	}
}

// intID returns a stable integer ID for the given UUID, creating one if needed.
func (m *tagMapper) intID(uuid string) int64 {
	m.mu.RLock()
	if id, ok := m.uuidInt[uuid]; ok {
		m.mu.RUnlock()
		return id
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check after acquiring write lock.
	if id, ok := m.uuidInt[uuid]; ok {
		return id
	}
	id := m.next
	m.next++
	m.uuidInt[uuid] = id
	m.intUUID[id] = uuid
	return id
}

func registerTagRoutes(api huma.API, svc *tag.Service) *tagMapper {
	mapper := newTagMapper()

	if svc == nil {
		return mapper
	}

	huma.Register(api, huma.Operation{
		OperationID: "radarr-list-tags",
		Method:      http.MethodGet,
		Path:        "/api/v3/tag",
		Summary:     "List tags (Radarr v3 compatible)",
		Tags:        []string{"RadarrCompat"},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body []RadarrTag }, error) {
		tags, err := svc.List(ctx)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "failed to list tags", err)
		}
		result := make([]RadarrTag, len(tags))
		for i, t := range tags {
			result[i] = RadarrTag{
				ID:    mapper.intID(t.ID),
				Label: t.Name,
			}
		}
		return &struct{ Body []RadarrTag }{Body: result}, nil
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
	}, func(ctx context.Context, input *createTagInput) (*struct {
		Status int
		Body   RadarrTag
	}, error) {
		t, err := svc.Create(ctx, input.Body.Label)
		if err != nil {
			return nil, huma.NewError(http.StatusConflict, err.Error())
		}
		return &struct {
			Status int
			Body   RadarrTag
		}{Status: http.StatusCreated, Body: RadarrTag{
			ID:    mapper.intID(t.ID),
			Label: t.Name,
		}}, nil
	})

	return mapper
}
