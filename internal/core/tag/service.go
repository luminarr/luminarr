// Package tag manages the tag system used to link movies with indexers,
// download clients, and notifications.
package tag

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/luminarr/luminarr/internal/core/dbutil"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
)

// Tag is the domain representation of a tag.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Usage counts — populated by List().
	MovieCount          int64 `json:"movie_count"`
	IndexerCount        int64 `json:"indexer_count"`
	DownloadClientCount int64 `json:"download_client_count"`
	NotificationCount   int64 `json:"notification_count"`
}

// Service manages tags and their entity associations.
type Service struct {
	q dbsqlite.Querier
}

// NewService creates a new Service.
func NewService(q dbsqlite.Querier) *Service {
	return &Service{q: q}
}

// List returns all tags with usage counts.
func (s *Service) List(ctx context.Context) ([]Tag, error) {
	rows, err := s.q.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	tags := make([]Tag, len(rows))
	for i, r := range rows {
		tags[i] = Tag{ID: r.ID, Name: r.Name}

		mc, _ := s.q.CountMoviesForTag(ctx, r.ID)
		ic, _ := s.q.CountIndexersForTag(ctx, r.ID)
		dc, _ := s.q.CountDownloadClientsForTag(ctx, r.ID)
		nc, _ := s.q.CountNotificationsForTag(ctx, r.ID)

		tags[i].MovieCount = mc
		tags[i].IndexerCount = ic
		tags[i].DownloadClientCount = dc
		tags[i].NotificationCount = nc
	}
	return tags, nil
}

// Get returns a single tag by ID.
func (s *Service) Get(ctx context.Context, id string) (Tag, error) {
	row, err := s.q.GetTag(ctx, id)
	if err != nil {
		return Tag{}, fmt.Errorf("getting tag %q: %w", id, err)
	}
	return Tag{ID: row.ID, Name: row.Name}, nil
}

// Create creates a new tag with the given name.
func (s *Service) Create(ctx context.Context, name string) (Tag, error) {
	row, err := s.q.CreateTag(ctx, dbsqlite.CreateTagParams{
		ID:   uuid.New().String(),
		Name: name,
	})
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return Tag{}, fmt.Errorf("tag %q already exists", name)
		}
		return Tag{}, fmt.Errorf("creating tag: %w", err)
	}
	return Tag{ID: row.ID, Name: row.Name}, nil
}

// Update renames an existing tag.
func (s *Service) Update(ctx context.Context, id, name string) (Tag, error) {
	row, err := s.q.UpdateTag(ctx, dbsqlite.UpdateTagParams{ID: id, Name: name})
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return Tag{}, fmt.Errorf("tag %q already exists", name)
		}
		return Tag{}, fmt.Errorf("updating tag %q: %w", id, err)
	}
	return Tag{ID: row.ID, Name: row.Name}, nil
}

// Delete removes a tag. Cascade deletes handle junction table cleanup.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.q.DeleteTag(ctx, id); err != nil {
		return fmt.Errorf("deleting tag %q: %w", id, err)
	}
	return nil
}

// SetMovieTags replaces all tags for a movie.
func (s *Service) SetMovieTags(ctx context.Context, movieID string, tagIDs []string) error {
	if err := s.q.SetMovieTags(ctx, movieID); err != nil {
		return fmt.Errorf("clearing movie tags: %w", err)
	}
	for _, tid := range tagIDs {
		if err := s.q.AddMovieTag(ctx, dbsqlite.AddMovieTagParams{MovieID: movieID, TagID: tid}); err != nil {
			return fmt.Errorf("adding movie tag %q: %w", tid, err)
		}
	}
	return nil
}

// MovieTagIDs returns the tag IDs for a movie.
func (s *Service) MovieTagIDs(ctx context.Context, movieID string) ([]string, error) {
	ids, err := s.q.ListMovieTagIDs(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("listing movie tags: %w", err)
	}
	if ids == nil {
		return []string{}, nil
	}
	return ids, nil
}

// SetIndexerTags replaces all tags for an indexer.
func (s *Service) SetIndexerTags(ctx context.Context, indexerID string, tagIDs []string) error {
	if err := s.q.SetIndexerTags(ctx, indexerID); err != nil {
		return fmt.Errorf("clearing indexer tags: %w", err)
	}
	for _, tid := range tagIDs {
		if err := s.q.AddIndexerTag(ctx, dbsqlite.AddIndexerTagParams{IndexerID: indexerID, TagID: tid}); err != nil {
			return fmt.Errorf("adding indexer tag %q: %w", tid, err)
		}
	}
	return nil
}

// IndexerTagIDs returns the tag IDs for an indexer.
func (s *Service) IndexerTagIDs(ctx context.Context, indexerID string) ([]string, error) {
	ids, err := s.q.ListIndexerTagIDs(ctx, indexerID)
	if err != nil {
		return nil, fmt.Errorf("listing indexer tags: %w", err)
	}
	if ids == nil {
		return []string{}, nil
	}
	return ids, nil
}

// SetDownloadClientTags replaces all tags for a download client.
func (s *Service) SetDownloadClientTags(ctx context.Context, clientID string, tagIDs []string) error {
	if err := s.q.SetDownloadClientTags(ctx, clientID); err != nil {
		return fmt.Errorf("clearing download client tags: %w", err)
	}
	for _, tid := range tagIDs {
		if err := s.q.AddDownloadClientTag(ctx, dbsqlite.AddDownloadClientTagParams{DownloadClientID: clientID, TagID: tid}); err != nil {
			return fmt.Errorf("adding download client tag %q: %w", tid, err)
		}
	}
	return nil
}

// DownloadClientTagIDs returns the tag IDs for a download client.
func (s *Service) DownloadClientTagIDs(ctx context.Context, clientID string) ([]string, error) {
	ids, err := s.q.ListDownloadClientTagIDs(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("listing download client tags: %w", err)
	}
	if ids == nil {
		return []string{}, nil
	}
	return ids, nil
}

// SetNotificationTags replaces all tags for a notification.
func (s *Service) SetNotificationTags(ctx context.Context, notifID string, tagIDs []string) error {
	if err := s.q.SetNotificationTags(ctx, notifID); err != nil {
		return fmt.Errorf("clearing notification tags: %w", err)
	}
	for _, tid := range tagIDs {
		if err := s.q.AddNotificationTag(ctx, dbsqlite.AddNotificationTagParams{NotificationID: notifID, TagID: tid}); err != nil {
			return fmt.Errorf("adding notification tag %q: %w", tid, err)
		}
	}
	return nil
}

// NotificationTagIDs returns the tag IDs for a notification.
func (s *Service) NotificationTagIDs(ctx context.Context, notifID string) ([]string, error) {
	ids, err := s.q.ListNotificationTagIDs(ctx, notifID)
	if err != nil {
		return nil, fmt.Errorf("listing notification tags: %w", err)
	}
	if ids == nil {
		return []string{}, nil
	}
	return ids, nil
}

// TagsOverlap returns true if the two tag sets share at least one tag.
// An empty entityTags set means the entity is "untagged" and available to all movies.
func TagsOverlap(movieTags, entityTags []string) bool {
	if len(entityTags) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(movieTags))
	for _, t := range movieTags {
		set[t] = struct{}{}
	}
	for _, t := range entityTags {
		if _, ok := set[t]; ok {
			return true
		}
	}
	return false
}
