package tmdb

// SearchResult is a single result from the TMDB movie search endpoint.
type SearchResult struct {
	ID            int
	Title         string
	OriginalTitle string
	Overview      string
	ReleaseDate   string // "YYYY-MM-DD" or empty
	Year          int    // parsed from ReleaseDate; 0 if unavailable
	PosterPath    string // TMDB path; prefix with image base URL before use
	BackdropPath  string
	Popularity    float64
}

// Person is summary info for a TMDB person.
type Person struct {
	ID                 int
	Name               string
	ProfilePath        string
	KnownForDepartment string
}

// PersonSearchResult is a single result from the TMDB person search endpoint.
type PersonSearchResult struct {
	ID                 int
	Name               string
	ProfilePath        string
	KnownForDepartment string
}

// FilmographyItem is one film entry in a person's filmography.
type FilmographyItem struct {
	TMDBID     int
	Title      string
	Year       int
	PosterPath string
	Order      int // cast billing order (0 for directors)
}

// MovieDetail is the full response from the TMDB /movie/{id} endpoint.
type MovieDetail struct {
	ID             int
	IMDBId         string
	Title          string
	OriginalTitle  string
	Overview       string
	ReleaseDate    string
	Year           int // parsed from ReleaseDate; 0 if unavailable
	RuntimeMinutes int
	Genres         []string
	PosterPath     string
	BackdropPath   string
	// Status is mapped from the TMDB status string to our internal values.
	// TMDB "Released" → "released"; "In Production" / "Post Production" / anything else → "announced".
	Status string
}
