// Package edition detects movie editions (Director's Cut, Extended, IMAX, etc.)
// from scene release titles and filenames. It is a pure, stateless parser with
// no external dependencies.
package edition

import (
	"regexp"
	"strings"
)

// Edition represents a detected movie edition.
type Edition struct {
	// Name is the canonical edition label, e.g. "Director's Cut", "Extended".
	Name string
	// Raw is the matched token from the input, e.g. "Directors.Cut", "EXTENDED".
	Raw string
}

// All regexps are compiled once at package init — never inside Parse.
// The (?i) flag makes every pattern case-insensitive.
// Patterns use word boundaries (\b) where needed to avoid false positives.
//
// Order matters: more specific patterns (e.g. "Final Cut") must come before
// broader ones (e.g. "Theatrical Cut" before bare "Theatrical").

type editionRule struct {
	name string
	re   *regexp.Regexp
}

var rules = []editionRule{
	// ── Multi-word editions (check first to avoid partial matches) ────────
	{name: "Director's Cut", re: regexp.MustCompile(`(?i)\bdirector'?s[\s._-]?(?:cut|edition)\b`)},
	{name: "Extended", re: regexp.MustCompile(`(?i)\bextended[\s._-]?(?:cut|edition|version)\b`)},
	{name: "Theatrical", re: regexp.MustCompile(`(?i)\btheatrical[\s._-]?(?:cut|edition|release)\b`)},
	{name: "Unrated", re: regexp.MustCompile(`(?i)\b(?:unrated[\s._-]?(?:cut|edition)?|uncensored)\b`)},
	{name: "Ultimate", re: regexp.MustCompile(`(?i)\bultimate[\s._-]?(?:cut|edition|collector'?s?)\b`)},
	{name: "Special Edition", re: regexp.MustCompile(`(?i)\bspecial[\s._-]?edition\b`)},
	{name: "Criterion", re: regexp.MustCompile(`(?i)\bcriterion[\s._-]?(?:collection)?\b`)},
	{name: "IMAX", re: regexp.MustCompile(`(?i)\bimax[\s._-]?(?:edition)?\b`)},
	{name: "Final Cut", re: regexp.MustCompile(`(?i)\bfinal[\s._-]?cut\b`)},
	{name: "Open Matte", re: regexp.MustCompile(`(?i)\bopen[\s._-]?matte\b`)},
	{name: "Rogue Cut", re: regexp.MustCompile(`(?i)\brogue[\s._-]?cut\b`)},
	{name: "Black and Chrome", re: regexp.MustCompile(`(?i)\bblack[\s._-]?and[\s._-]?chrome\b`)},

	// ── Remastered variants ──────────────────────────────────────────────
	{name: "Remastered", re: regexp.MustCompile(`(?i)\b(?:4k[\s._-]?)?(?:digitally[\s._-]?)?remastered\b`)},

	// ── Anniversary (with optional Nth prefix) ──────────────────────────
	{name: "Anniversary", re: regexp.MustCompile(`(?i)\b(?:\d+(?:st|nd|rd|th)[\s._-]?)?anniversary[\s._-]?(?:edition)?\b`)},

	// ── Single-word editions (check last to avoid false positives) ───────
	{name: "Extended", re: regexp.MustCompile(`(?i)\bextended\b`)},
	{name: "Theatrical", re: regexp.MustCompile(`(?i)\btheatrical\b`)},
	{name: "Redux", re: regexp.MustCompile(`(?i)\bredux\b`)},
}

// Canonical returns the list of canonical edition names that the parser
// recognises. Useful for populating UI dropdowns.
func Canonical() []string {
	return []string{
		"Theatrical",
		"Director's Cut",
		"Extended",
		"Unrated",
		"Ultimate",
		"Special Edition",
		"Criterion",
		"IMAX",
		"Remastered",
		"Anniversary",
		"Final Cut",
		"Redux",
		"Rogue Cut",
		"Black and Chrome",
		"Open Matte",
	}
}

// EditionBonus is the score bonus awarded when a release matches the movie's
// preferred edition. Additive to the 0–100 quality score.
const EditionBonus = 30

// Bonus returns EditionBonus when the release edition matches the movie's
// preferred edition, 0 otherwise. Both values are compared case-insensitively.
// An empty preferred or release edition never earns a bonus.
func Bonus(preferred, release string) int {
	if preferred == "" || release == "" {
		return 0
	}
	if strings.EqualFold(preferred, release) {
		return EditionBonus
	}
	return 0
}

// Parse extracts edition information from a release title or filename.
// Returns nil if no edition is detected. The absence of an edition tag
// means "unknown/default", NOT "Theatrical".
//
// Parse is a pure function with no external dependencies.
func Parse(title string) *Edition {
	// Normalise: replace dots and underscores with spaces so all patterns
	// see a consistent word-separated string. The original casing is preserved
	// because all regexps carry (?i).
	norm := strings.NewReplacer(".", " ", "_", " ").Replace(title)

	for _, rule := range rules {
		if loc := rule.re.FindStringIndex(norm); loc != nil {
			return &Edition{
				Name: rule.name,
				Raw:  strings.TrimSpace(norm[loc[0]:loc[1]]),
			}
		}
	}

	return nil
}
