package plugin

// ScoreBreakdown records how a release was evaluated against a quality profile.
// Each dimension is independently scored; Total is the sum.
type ScoreBreakdown struct {
	Total      int              `json:"total"`
	Dimensions []ScoreDimension `json:"dimensions"`
	// CustomFormatScore is the sum of matched custom format scores for the
	// quality profile. It sits alongside (not inside) Total because CF scoring
	// is an independent dimension used for separate thresholds.
	CustomFormatScore int      `json:"custom_format_score"`
	MatchedFormats    []string `json:"matched_formats,omitempty"`
	// EditionBonus is the bonus points awarded when the release edition
	// matches the movie's preferred edition (+30 pts). It is additive
	// to Total and reflected in the Dimensions list.
	EditionBonus int `json:"edition_bonus"`
}

// ScoreDimension is one component of a ScoreBreakdown.
type ScoreDimension struct {
	Name    string `json:"name"`    // "resolution", "source", "codec", "hdr"
	Score   int    `json:"score"`   // points awarded for this dimension
	Max     int    `json:"max"`     // maximum possible for this dimension
	Matched bool   `json:"matched"` // did it meet the profile requirement?
	Got     string `json:"got"`     // what we found (e.g. "x264")
	Want    string `json:"want"`    // what the profile requires (e.g. "x265")
}
