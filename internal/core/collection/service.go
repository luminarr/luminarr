// Package collection manages director/actor filmography collections.
package collection

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/davidfic/luminarr/internal/core/movie"
	dbsqlite "github.com/davidfic/luminarr/internal/db/generated/sqlite"
	"github.com/davidfic/luminarr/internal/metadata/tmdb"
)

// MetadataProvider is the subset of the TMDB client needed by this package.
type MetadataProvider interface {
	GetPerson(ctx context.Context, personID int) (*tmdb.Person, error)
	GetPersonFilmography(ctx context.Context, personID int, personType string) ([]tmdb.FilmographyItem, error)
	SearchPeople(ctx context.Context, query string) ([]tmdb.PersonSearchResult, error)
}

// Sentinel errors.
var (
	ErrNotFound      = errors.New("collection not found")
	ErrAlreadyExists = errors.New("collection already exists for this person")
)

// Item is one film in a person's filmography.
type Item struct {
	TMDBID     int
	Title      string
	Year       int
	PosterPath string
	InLibrary  bool
	MovieID    string // set when InLibrary=true
	Monitored  bool   // set when InLibrary=true
}

// Collection is the full view of a collection record.
type Collection struct {
	ID         string
	Name       string
	PersonID   int
	PersonType string
	CreatedAt  time.Time
	Items      []Item // nil on List; populated on Get
	Total      int
	InLibrary  int
	Missing    int
}

// AddMissingRequest carries settings for adding all missing films at once.
type AddMissingRequest struct {
	LibraryID           string
	QualityProfileID    string
	MinimumAvailability string
}

// AddMissingResult summarises what happened.
type AddMissingResult struct {
	Added             int
	SkippedDuplicates int
}

// Service manages collection records.
type Service struct {
	q        dbsqlite.Querier
	provider MetadataProvider // nil when TMDB not configured
	movieSvc *movie.Service
}

// NewService creates a new Service. provider may be nil when TMDB is not configured;
// Create and SearchPeople will return an error in that case.
func NewService(q dbsqlite.Querier, provider MetadataProvider, movieSvc *movie.Service) *Service {
	return &Service{q: q, provider: provider, movieSvc: movieSvc}
}

// SearchPeople searches TMDB for people by name.
func (s *Service) SearchPeople(ctx context.Context, query string) ([]tmdb.PersonSearchResult, error) {
	if s.provider == nil {
		return nil, errors.New("TMDB not configured")
	}
	return s.provider.SearchPeople(ctx, query)
}

// Create fetches the person's name from TMDB and inserts a collection record.
// Returns ErrAlreadyExists if a collection for that person+type already exists.
func (s *Service) Create(ctx context.Context, personID int, personType string) (*Collection, error) {
	if s.provider == nil {
		return nil, errors.New("TMDB not configured")
	}

	// Duplicate check before hitting TMDB.
	if _, err := s.q.GetCollectionByPerson(ctx, dbsqlite.GetCollectionByPersonParams{
		PersonID:   int64(personID),
		PersonType: personType,
	}); err == nil {
		return nil, ErrAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("checking for existing collection: %w", err)
	}

	person, err := s.provider.GetPerson(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("fetching person from TMDB: %w", err)
	}

	row, err := s.q.CreateCollection(ctx, dbsqlite.CreateCollectionParams{
		ID:         uuid.New().String(),
		Name:       person.Name,
		PersonID:   int64(personID),
		PersonType: personType,
		CreatedAt:  time.Now().UTC(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("creating collection: %w", err)
	}
	return rowToCollection(row), nil
}

// List returns all collections without item lists.
func (s *Service) List(ctx context.Context) ([]Collection, error) {
	rows, err := s.q.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing collections: %w", err)
	}
	result := make([]Collection, 0, len(rows))
	for _, r := range rows {
		result = append(result, *rowToCollection(r))
	}
	return result, nil
}

// Get returns a collection with its full item list fetched live from TMDB.
func (s *Service) Get(ctx context.Context, id string) (*Collection, error) {
	row, err := s.q.GetCollection(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting collection: %w", err)
	}
	coll := rowToCollection(row)

	if s.provider == nil {
		return coll, nil
	}

	items, err := s.provider.GetPersonFilmography(ctx, int(row.PersonID), row.PersonType)
	if err != nil {
		return nil, fmt.Errorf("fetching filmography: %w", err)
	}

	collItems := make([]Item, 0, len(items))
	for _, item := range items {
		ci := Item{
			TMDBID:     item.TMDBID,
			Title:      item.Title,
			Year:       item.Year,
			PosterPath: item.PosterPath,
		}
		// Cross-reference with the movie library.
		m, lookupErr := s.q.GetMovieByTMDBID(ctx, int64(item.TMDBID))
		if lookupErr == nil {
			ci.InLibrary = true
			ci.MovieID = m.ID
			ci.Monitored = m.Monitored != 0
		}
		collItems = append(collItems, ci)
	}

	// Sort: in-library first, then by year descending.
	sort.SliceStable(collItems, func(i, j int) bool {
		if collItems[i].InLibrary != collItems[j].InLibrary {
			return collItems[i].InLibrary
		}
		return collItems[i].Year > collItems[j].Year
	})

	coll.Items = collItems
	coll.Total = len(collItems)
	for _, ci := range collItems {
		if ci.InLibrary {
			coll.InLibrary++
		} else {
			coll.Missing++
		}
	}
	return coll, nil
}

// Delete removes a collection record. Returns ErrNotFound if it does not exist.
func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.q.GetCollection(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("checking collection: %w", err)
	}
	return s.q.DeleteCollection(ctx, id)
}

// AddMissing adds all films in the collection that are not yet in the library.
func (s *Service) AddMissing(ctx context.Context, id string, req AddMissingRequest) (AddMissingResult, error) {
	coll, err := s.Get(ctx, id)
	if err != nil {
		return AddMissingResult{}, err
	}

	var result AddMissingResult
	for _, item := range coll.Items {
		if item.InLibrary {
			continue
		}
		_, addErr := s.movieSvc.Add(ctx, movie.AddRequest{
			TMDBID:              item.TMDBID,
			LibraryID:           req.LibraryID,
			QualityProfileID:    req.QualityProfileID,
			Monitored:           true,
			MinimumAvailability: req.MinimumAvailability,
		})
		if errors.Is(addErr, movie.ErrAlreadyExists) {
			result.SkippedDuplicates++
			continue
		}
		if addErr != nil {
			return result, fmt.Errorf("adding %q (tmdb_id=%d): %w", item.Title, item.TMDBID, addErr)
		}
		result.Added++
	}
	return result, nil
}

func rowToCollection(r dbsqlite.Collection) *Collection {
	return &Collection{
		ID:         r.ID,
		Name:       r.Name,
		PersonID:   int(r.PersonID),
		PersonType: r.PersonType,
		CreatedAt:  r.CreatedAt,
	}
}
