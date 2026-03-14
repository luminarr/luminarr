// Package customformat implements Radarr-compatible custom format management.
// Custom formats are user-defined scoring rules that evaluate releases against
// a set of conditions (specifications). Each condition matches against a specific
// aspect of a release (title regex, source, resolution, indexer flags, etc.).
// Matched formats contribute scores to release ranking within quality profiles.
package customformat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/luminarr/luminarr/internal/core/dbutil"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
)

// CustomFormat is the domain representation of a custom format.
type CustomFormat struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	IncludeWhenRenaming bool            `json:"include_when_renaming"`
	Specifications      []Specification `json:"specifications"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// Specification is a single condition within a custom format.
type Specification struct {
	Name           string            `json:"name"`
	Implementation string            `json:"implementation"`
	Negate         bool              `json:"negate"`
	Required       bool              `json:"required"`
	Fields         map[string]string `json:"fields"`
}

// Valid implementation types for specifications.
const (
	ImplReleaseTitle    = "release_title"
	ImplEdition         = "edition"
	ImplLanguage        = "language"
	ImplIndexerFlag     = "indexer_flag"
	ImplSource          = "source"
	ImplResolution      = "resolution"
	ImplQualityModifier = "quality_modifier"
	ImplSize            = "size"
	ImplReleaseGroup    = "release_group"
	ImplYear            = "year"
)

// CreateRequest holds fields needed to create a custom format.
type CreateRequest struct {
	Name                string          `json:"name"`
	IncludeWhenRenaming bool            `json:"include_when_renaming"`
	Specifications      []Specification `json:"specifications"`
}

// UpdateRequest holds fields needed to update a custom format.
type UpdateRequest struct {
	Name                string          `json:"name"`
	IncludeWhenRenaming bool            `json:"include_when_renaming"`
	Specifications      []Specification `json:"specifications"`
}

// Service manages custom format CRUD.
type Service struct {
	q dbsqlite.Querier
}

// NewService creates a new custom format Service.
func NewService(q dbsqlite.Querier) *Service {
	return &Service{q: q}
}

// List returns all custom formats ordered by name.
func (s *Service) List(ctx context.Context) ([]CustomFormat, error) {
	rows, err := s.q.ListCustomFormats(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing custom formats: %w", err)
	}
	formats := make([]CustomFormat, len(rows))
	for i, r := range rows {
		cf, err := fromRow(r)
		if err != nil {
			return nil, err
		}
		formats[i] = cf
	}
	return formats, nil
}

// Get returns a single custom format by ID.
func (s *Service) Get(ctx context.Context, id string) (CustomFormat, error) {
	row, err := s.q.GetCustomFormat(ctx, id)
	if err != nil {
		return CustomFormat{}, fmt.Errorf("getting custom format %q: %w", id, err)
	}
	return fromRow(row)
}

// Create creates a new custom format.
func (s *Service) Create(ctx context.Context, req CreateRequest) (CustomFormat, error) {
	specsJSON, err := json.Marshal(req.Specifications)
	if err != nil {
		return CustomFormat{}, fmt.Errorf("marshaling specifications: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	row, err := s.q.CreateCustomFormat(ctx, dbsqlite.CreateCustomFormatParams{
		ID:                  uuid.New().String(),
		Name:                req.Name,
		IncludeWhenRenaming: dbutil.BoolToInt(req.IncludeWhenRenaming),
		SpecificationsJson:  string(specsJSON),
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return CustomFormat{}, fmt.Errorf("custom format %q already exists", req.Name)
		}
		return CustomFormat{}, fmt.Errorf("creating custom format: %w", err)
	}
	return fromRow(row)
}

// Update modifies an existing custom format.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (CustomFormat, error) {
	specsJSON, err := json.Marshal(req.Specifications)
	if err != nil {
		return CustomFormat{}, fmt.Errorf("marshaling specifications: %w", err)
	}

	row, err := s.q.UpdateCustomFormat(ctx, dbsqlite.UpdateCustomFormatParams{
		ID:                  id,
		Name:                req.Name,
		IncludeWhenRenaming: dbutil.BoolToInt(req.IncludeWhenRenaming),
		SpecificationsJson:  string(specsJSON),
		UpdatedAt:           time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		if dbutil.IsUniqueViolation(err) {
			return CustomFormat{}, fmt.Errorf("custom format %q already exists", req.Name)
		}
		return CustomFormat{}, fmt.Errorf("updating custom format %q: %w", id, err)
	}
	return fromRow(row)
}

// Delete removes a custom format by ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.q.DeleteCustomFormat(ctx, id); err != nil {
		return fmt.Errorf("deleting custom format %q: %w", id, err)
	}
	return nil
}

// ListScores returns the custom format scores for a quality profile.
func (s *Service) ListScores(ctx context.Context, profileID string) (map[string]int, error) {
	rows, err := s.q.ListCustomFormatScores(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("listing custom format scores: %w", err)
	}
	scores := make(map[string]int, len(rows))
	for _, r := range rows {
		scores[r.CustomFormatID] = int(r.Score)
	}
	return scores, nil
}

// SetScores replaces all custom format scores for a quality profile.
func (s *Service) SetScores(ctx context.Context, profileID string, scores map[string]int) error {
	if err := s.q.DeleteCustomFormatScores(ctx, profileID); err != nil {
		return fmt.Errorf("clearing custom format scores: %w", err)
	}
	for cfID, score := range scores {
		if err := s.q.SetCustomFormatScore(ctx, dbsqlite.SetCustomFormatScoreParams{
			QualityProfileID: profileID,
			CustomFormatID:   cfID,
			Score:            int64(score),
		}); err != nil {
			return fmt.Errorf("setting score for format %q: %w", cfID, err)
		}
	}
	return nil
}

// fromRow converts a DB row to a domain CustomFormat.
func fromRow(r dbsqlite.CustomFormat) (CustomFormat, error) {
	var specs []Specification
	if err := json.Unmarshal([]byte(r.SpecificationsJson), &specs); err != nil {
		return CustomFormat{}, fmt.Errorf("parsing specifications for %q: %w", r.ID, err)
	}

	createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)

	return CustomFormat{
		ID:                  r.ID,
		Name:                r.Name,
		IncludeWhenRenaming: r.IncludeWhenRenaming != 0,
		Specifications:      specs,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
	}, nil
}
