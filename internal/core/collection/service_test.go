package collection_test

import (
	"context"
	"errors"
	"testing"

	"github.com/davidfic/luminarr/internal/core/collection"
	"github.com/davidfic/luminarr/internal/core/movie"
	"github.com/davidfic/luminarr/internal/events"
	"github.com/davidfic/luminarr/internal/metadata/tmdb"
	"github.com/davidfic/luminarr/internal/testutil"
	"github.com/davidfic/luminarr/internal/testutil/mock"

	"log/slog"
	"os"
)

func newServices(t *testing.T) (*collection.Service, *movie.Service, *mock.CollectionProvider, context.Context) {
	t.Helper()
	q := testutil.NewTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	bus := events.New(logger)
	movieSvc := movie.NewService(q, nil, bus, logger)
	provider := &mock.CollectionProvider{}
	svc := collection.NewService(q, provider, movieSvc)
	return svc, movieSvc, provider, context.Background()
}

func TestCreate(t *testing.T) {
	svc, _, _, ctx := newServices(t)

	c, err := svc.Create(ctx, 525, "director")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.Name != "Christopher Nolan" {
		t.Errorf("Name: got %q, want %q", c.Name, "Christopher Nolan")
	}
	if c.PersonID != 525 {
		t.Errorf("PersonID: got %d, want 525", c.PersonID)
	}
	if c.PersonType != "director" {
		t.Errorf("PersonType: got %q, want %q", c.PersonType, "director")
	}
}

func TestCreate_duplicate(t *testing.T) {
	svc, _, _, ctx := newServices(t)

	if _, err := svc.Create(ctx, 525, "director"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := svc.Create(ctx, 525, "director")
	if !errors.Is(err, collection.ErrAlreadyExists) {
		t.Errorf("second Create: got %v, want ErrAlreadyExists", err)
	}
}

func TestList(t *testing.T) {
	svc, _, _, ctx := newServices(t)

	if _, err := svc.Create(ctx, 525, "director"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	colls, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(colls) != 1 {
		t.Errorf("len: got %d, want 1", len(colls))
	}
}

func TestGet_crossReferencesLibrary(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	q := testutil.NewTestDB(t)
	bus := events.New(logger)

	provider := &mock.CollectionProvider{
		GetPersonFilmographyFunc: func(_ context.Context, _ int, _ string) ([]tmdb.FilmographyItem, error) {
			return []tmdb.FilmographyItem{
				{TMDBID: 27205, Title: "Inception", Year: 2010},
			}, nil
		},
	}
	tmdbMock := &mock.TMDBClient{
		GetMovieFunc: func(_ context.Context, id int) (*tmdb.MovieDetail, error) {
			return &tmdb.MovieDetail{ID: id, Title: "Inception", Year: 2010, Status: "released"}, nil
		},
	}
	movieSvc := movie.NewService(q, tmdbMock, bus, logger)
	svc := collection.NewService(q, provider, movieSvc)

	lib := testutil.SeedLibrary(t, q)
	qp := testutil.SeedQualityProfile(t, q)
	if _, err := movieSvc.Add(ctx, movie.AddRequest{
		TMDBID:           27205,
		LibraryID:        lib.ID,
		QualityProfileID: qp.ID,
		Monitored:        true,
	}); err != nil {
		t.Fatalf("Add movie: %v", err)
	}

	coll, err := svc.Create(ctx, 525, "director")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	full, err := svc.Get(ctx, coll.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(full.Items) != 1 {
		t.Fatalf("Items len: got %d, want 1", len(full.Items))
	}
	if !full.Items[0].InLibrary {
		t.Errorf("InLibrary: got false, want true")
	}
	if full.InLibrary != 1 {
		t.Errorf("InLibrary count: got %d, want 1", full.InLibrary)
	}
	if full.Missing != 0 {
		t.Errorf("Missing count: got %d, want 0", full.Missing)
	}
}

func TestGet_missing(t *testing.T) {
	svc, _, provider, ctx := newServices(t)

	provider.GetPersonFilmographyFunc = func(_ context.Context, _ int, _ string) ([]tmdb.FilmographyItem, error) {
		return []tmdb.FilmographyItem{
			{TMDBID: 27205, Title: "Inception", Year: 2010},
		}, nil
	}

	coll, err := svc.Create(ctx, 525, "director")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	full, err := svc.Get(ctx, coll.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if full.Items[0].InLibrary {
		t.Errorf("InLibrary: got true, want false")
	}
	if full.Missing != 1 {
		t.Errorf("Missing: got %d, want 1", full.Missing)
	}
}

func TestDelete(t *testing.T) {
	svc, _, _, ctx := newServices(t)

	c, err := svc.Create(ctx, 525, "director")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	colls, _ := svc.List(ctx)
	if len(colls) != 0 {
		t.Errorf("List after delete: got %d, want 0", len(colls))
	}
}

func TestDelete_notFound(t *testing.T) {
	svc, _, _, ctx := newServices(t)
	err := svc.Delete(ctx, "nonexistent-id")
	if !errors.Is(err, collection.ErrNotFound) {
		t.Errorf("Delete nonexistent: got %v, want ErrNotFound", err)
	}
}
