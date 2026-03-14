package quality

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/events"
	"github.com/luminarr/luminarr/pkg/plugin"
)

// ErrNotFound is returned when a quality profile does not exist.
var ErrNotFound = errors.New("quality profile not found")

// ErrInUse is returned when attempting to delete a profile that is referenced
// by one or more movies or libraries.
var ErrInUse = errors.New("quality profile is in use")

// CreateRequest carries the fields needed to create a quality profile.
type CreateRequest struct {
	Name                 string
	Cutoff               plugin.Quality
	Qualities            []plugin.Quality
	UpgradeAllowed       bool
	UpgradeUntil         *plugin.Quality
	MinCustomFormatScore int
	UpgradeUntilCFScore  int
}

// UpdateRequest carries the fields needed to update a quality profile.
// It is identical in shape to CreateRequest.
type UpdateRequest = CreateRequest

// Service manages quality profiles.
type Service struct {
	q   dbsqlite.Querier
	bus *events.Bus
}

// NewService creates a new Service backed by the given querier and event bus.
func NewService(q dbsqlite.Querier, bus *events.Bus) *Service {
	return &Service{q: q, bus: bus}
}

// Create creates a new quality profile and returns the persisted domain type.
func (s *Service) Create(ctx context.Context, req CreateRequest) (Profile, error) {
	cutoffJSON, err := json.Marshal(req.Cutoff)
	if err != nil {
		return Profile{}, fmt.Errorf("marshaling cutoff: %w", err)
	}

	qualitiesJSON, err := json.Marshal(req.Qualities)
	if err != nil {
		return Profile{}, fmt.Errorf("marshaling qualities: %w", err)
	}

	var upgradeUntilJSON *string
	if req.UpgradeUntil != nil {
		b, err := json.Marshal(req.UpgradeUntil)
		if err != nil {
			return Profile{}, fmt.Errorf("marshaling upgrade_until: %w", err)
		}
		s := string(b)
		upgradeUntilJSON = &s
	}

	now := time.Now().UTC().Format(time.RFC3339)
	upgradeAllowed := int64(0)
	if req.UpgradeAllowed {
		upgradeAllowed = 1
	}

	row, err := s.q.CreateQualityProfile(ctx, dbsqlite.CreateQualityProfileParams{
		ID:                   uuid.New().String(),
		Name:                 req.Name,
		CutoffJson:           string(cutoffJSON),
		QualitiesJson:        string(qualitiesJSON),
		UpgradeAllowed:       upgradeAllowed,
		UpgradeUntilJson:     upgradeUntilJSON,
		CreatedAt:            now,
		UpdatedAt:            now,
		MinCustomFormatScore: int64(req.MinCustomFormatScore),
		UpgradeUntilCfScore:  int64(req.UpgradeUntilCFScore),
	})
	if err != nil {
		return Profile{}, fmt.Errorf("inserting quality profile: %w", err)
	}

	return rowToProfile(row)
}

// Get returns a quality profile by ID.
// Returns ErrNotFound if no profile with that ID exists.
func (s *Service) Get(ctx context.Context, id string) (Profile, error) {
	row, err := s.q.GetQualityProfile(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Profile{}, ErrNotFound
		}
		return Profile{}, fmt.Errorf("fetching quality profile %q: %w", id, err)
	}
	return rowToProfile(row)
}

// List returns all quality profiles ordered by name.
func (s *Service) List(ctx context.Context) ([]Profile, error) {
	rows, err := s.q.ListQualityProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing quality profiles: %w", err)
	}

	profiles := make([]Profile, 0, len(rows))
	for _, row := range rows {
		p, err := rowToProfile(row)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// Update replaces the mutable fields of an existing quality profile.
// Returns ErrNotFound if the profile does not exist.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Profile, error) {
	// Confirm the profile exists before attempting an update so we can
	// distinguish "not found" from other DB errors.
	if _, err := s.q.GetQualityProfile(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Profile{}, ErrNotFound
		}
		return Profile{}, fmt.Errorf("fetching quality profile %q for update: %w", id, err)
	}

	cutoffJSON, err := json.Marshal(req.Cutoff)
	if err != nil {
		return Profile{}, fmt.Errorf("marshaling cutoff: %w", err)
	}

	qualitiesJSON, err := json.Marshal(req.Qualities)
	if err != nil {
		return Profile{}, fmt.Errorf("marshaling qualities: %w", err)
	}

	var upgradeUntilJSON *string
	if req.UpgradeUntil != nil {
		b, err := json.Marshal(req.UpgradeUntil)
		if err != nil {
			return Profile{}, fmt.Errorf("marshaling upgrade_until: %w", err)
		}
		s := string(b)
		upgradeUntilJSON = &s
	}

	upgradeAllowed := int64(0)
	if req.UpgradeAllowed {
		upgradeAllowed = 1
	}

	row, err := s.q.UpdateQualityProfile(ctx, dbsqlite.UpdateQualityProfileParams{
		ID:                   id,
		Name:                 req.Name,
		CutoffJson:           string(cutoffJSON),
		QualitiesJson:        string(qualitiesJSON),
		UpgradeAllowed:       upgradeAllowed,
		UpgradeUntilJson:     upgradeUntilJSON,
		UpdatedAt:            time.Now().UTC().Format(time.RFC3339),
		MinCustomFormatScore: int64(req.MinCustomFormatScore),
		UpgradeUntilCfScore:  int64(req.UpgradeUntilCFScore),
	})
	if err != nil {
		return Profile{}, fmt.Errorf("updating quality profile %q: %w", id, err)
	}

	return rowToProfile(row)
}

// Delete removes a quality profile. Returns ErrNotFound if it does not exist,
// and ErrInUse if any movie or library currently references it.
func (s *Service) Delete(ctx context.Context, id string) error {
	// Confirm existence first.
	if _, err := s.q.GetQualityProfile(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("fetching quality profile %q for delete: %w", id, err)
	}

	// Check referential usage (movies + libraries).
	count, err := s.q.QualityProfileInUse(ctx, dbsqlite.QualityProfileInUseParams{
		QualityProfileID:        id,
		DefaultQualityProfileID: id,
	})
	if err != nil {
		return fmt.Errorf("checking quality profile usage for %q: %w", id, err)
	}
	if count > 0 {
		return ErrInUse
	}

	if err := s.q.DeleteQualityProfile(ctx, id); err != nil {
		return fmt.Errorf("deleting quality profile %q: %w", id, err)
	}
	return nil
}

// rowToProfile converts a DB row into the domain Profile type.
func rowToProfile(row dbsqlite.QualityProfile) (Profile, error) {
	var cutoff plugin.Quality
	if err := json.Unmarshal([]byte(row.CutoffJson), &cutoff); err != nil {
		return Profile{}, fmt.Errorf("unmarshaling cutoff for profile %q: %w", row.ID, err)
	}

	var qualities []plugin.Quality
	if err := json.Unmarshal([]byte(row.QualitiesJson), &qualities); err != nil {
		return Profile{}, fmt.Errorf("unmarshaling qualities for profile %q: %w", row.ID, err)
	}

	var upgradeUntil *plugin.Quality
	if row.UpgradeUntilJson != nil {
		var q plugin.Quality
		if err := json.Unmarshal([]byte(*row.UpgradeUntilJson), &q); err != nil {
			return Profile{}, fmt.Errorf("unmarshaling upgrade_until for profile %q: %w", row.ID, err)
		}
		upgradeUntil = &q
	}

	return Profile{
		ID:                   row.ID,
		Name:                 row.Name,
		Cutoff:               cutoff,
		Qualities:            qualities,
		UpgradeAllowed:       row.UpgradeAllowed != 0,
		UpgradeUntil:         upgradeUntil,
		MinCustomFormatScore: int(row.MinCustomFormatScore),
		UpgradeUntilCFScore:  int(row.UpgradeUntilCfScore),
	}, nil
}
