package mock

import (
	"context"

	"github.com/davidfic/luminarr/internal/metadata/tmdb"
)

// CollectionProvider is a configurable mock of collection.MetadataProvider.
type CollectionProvider struct {
	GetPersonFunc            func(ctx context.Context, personID int) (*tmdb.Person, error)
	GetPersonFilmographyFunc func(ctx context.Context, personID int, personType string) ([]tmdb.FilmographyItem, error)
	SearchPeopleFunc         func(ctx context.Context, query string) ([]tmdb.PersonSearchResult, error)
	SearchFranchisesFunc     func(ctx context.Context, query string) ([]tmdb.FranchiseSearchResult, error)
	GetFranchiseFunc         func(ctx context.Context, collectionID int) (*tmdb.FranchiseDetail, error)
}

func (m *CollectionProvider) GetPerson(ctx context.Context, personID int) (*tmdb.Person, error) {
	if m.GetPersonFunc != nil {
		return m.GetPersonFunc(ctx, personID)
	}
	return &tmdb.Person{ID: personID, Name: "Christopher Nolan"}, nil
}

func (m *CollectionProvider) GetPersonFilmography(ctx context.Context, personID int, personType string) ([]tmdb.FilmographyItem, error) {
	if m.GetPersonFilmographyFunc != nil {
		return m.GetPersonFilmographyFunc(ctx, personID, personType)
	}
	return []tmdb.FilmographyItem{
		{TMDBID: 27205, Title: "Inception", Year: 2010},
		{TMDBID: 49026, Title: "The Dark Knight Rises", Year: 2012},
	}, nil
}

func (m *CollectionProvider) SearchPeople(ctx context.Context, query string) ([]tmdb.PersonSearchResult, error) {
	if m.SearchPeopleFunc != nil {
		return m.SearchPeopleFunc(ctx, query)
	}
	return []tmdb.PersonSearchResult{
		{ID: 525, Name: "Christopher Nolan", KnownForDepartment: "Directing"},
	}, nil
}

func (m *CollectionProvider) SearchFranchises(ctx context.Context, query string) ([]tmdb.FranchiseSearchResult, error) {
	if m.SearchFranchisesFunc != nil {
		return m.SearchFranchisesFunc(ctx, query)
	}
	return []tmdb.FranchiseSearchResult{}, nil
}

func (m *CollectionProvider) GetFranchise(ctx context.Context, collectionID int) (*tmdb.FranchiseDetail, error) {
	if m.GetFranchiseFunc != nil {
		return m.GetFranchiseFunc(ctx, collectionID)
	}
	return &tmdb.FranchiseDetail{
		ID:   collectionID,
		Name: "Alien Collection",
		Parts: []tmdb.FilmographyItem{
			{TMDBID: 348, Title: "Alien", Year: 1979},
			{TMDBID: 679, Title: "Aliens", Year: 1986},
		},
	}, nil
}
