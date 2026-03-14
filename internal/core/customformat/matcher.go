package customformat

import (
	"regexp"
	"strconv"
	"strings"
)

// ReleaseInfo is the input to the matcher — extracted from release title + indexer metadata.
type ReleaseInfo struct {
	Title        string
	Edition      string
	Languages    []string
	IndexerFlags []string
	Source       string
	Resolution   string
	Modifier     string // "remux", "brdisk", "rawhd", etc.
	SizeBytes    int64
	ReleaseGroup string
	Year         int
}

// MatchRelease evaluates all custom formats against a release.
// Returns the IDs of formats that match.
func MatchRelease(formats []CustomFormat, release ReleaseInfo) []string {
	var matched []string
	for _, cf := range formats {
		if matchFormat(cf, release) {
			matched = append(matched, cf.ID)
		}
	}
	return matched
}

// ScoreRelease computes the total custom format score for a release given
// the matched format IDs and a quality profile's per-format scores.
func ScoreRelease(matched []string, profileScores map[string]int) int {
	total := 0
	for _, id := range matched {
		total += profileScores[id]
	}
	return total
}

// matchFormat evaluates a single custom format against a release.
// Groups conditions by implementation type, then:
//   - All required conditions across all groups must pass.
//   - For each group that has optional conditions, at least one must pass.
func matchFormat(cf CustomFormat, rel ReleaseInfo) bool {
	if len(cf.Specifications) == 0 {
		return false
	}

	// Group specs by implementation type.
	groups := make(map[string][]Specification)
	for _, spec := range cf.Specifications {
		groups[spec.Implementation] = append(groups[spec.Implementation], spec)
	}

	for _, specs := range groups {
		var required, optional []Specification
		for _, s := range specs {
			if s.Required {
				required = append(required, s)
			} else {
				optional = append(optional, s)
			}
		}

		// All required conditions must pass.
		for _, s := range required {
			if !evalSpec(s, rel) {
				return false
			}
		}

		// If optional conditions exist, at least one must pass.
		if len(optional) > 0 {
			anyMatch := false
			for _, s := range optional {
				if evalSpec(s, rel) {
					anyMatch = true
					break
				}
			}
			if !anyMatch {
				return false
			}
		}
	}

	return true
}

// evalSpec evaluates a single specification against a release, respecting negate.
func evalSpec(spec Specification, rel ReleaseInfo) bool {
	result := evalCondition(spec, rel)
	if spec.Negate {
		return !result
	}
	return result
}

// evalCondition dispatches to the appropriate condition evaluator.
func evalCondition(spec Specification, rel ReleaseInfo) bool {
	switch spec.Implementation {
	case ImplReleaseTitle:
		return matchRegex(spec.Fields["value"], rel.Title)
	case ImplEdition:
		return matchRegex(spec.Fields["value"], rel.Edition)
	case ImplLanguage:
		return matchLanguage(spec.Fields["value"], rel.Languages)
	case ImplIndexerFlag:
		return matchIndexerFlag(spec.Fields["value"], rel.IndexerFlags)
	case ImplSource:
		return strings.EqualFold(spec.Fields["value"], rel.Source)
	case ImplResolution:
		return strings.EqualFold(spec.Fields["value"], rel.Resolution)
	case ImplQualityModifier:
		return strings.EqualFold(spec.Fields["value"], rel.Modifier)
	case ImplSize:
		return matchSize(spec.Fields["min"], spec.Fields["max"], rel.SizeBytes)
	case ImplReleaseGroup:
		return matchRegex(spec.Fields["value"], rel.ReleaseGroup)
	case ImplYear:
		return matchYear(spec.Fields["min"], spec.Fields["max"], rel.Year)
	default:
		return false
	}
}

// matchRegex compiles the pattern and tests it against the input.
// Returns false on invalid patterns rather than erroring.
func matchRegex(pattern, input string) bool {
	if pattern == "" {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(input)
}

// matchLanguage checks if the target language appears in the release's languages.
func matchLanguage(target string, languages []string) bool {
	if target == "" {
		return false
	}
	for _, lang := range languages {
		if strings.EqualFold(lang, target) {
			return true
		}
	}
	return false
}

// matchIndexerFlag checks if the target flag appears in the release's indexer flags.
func matchIndexerFlag(target string, flags []string) bool {
	if target == "" {
		return false
	}
	for _, f := range flags {
		if strings.EqualFold(f, target) {
			return true
		}
	}
	return false
}

// matchSize checks if the release size (in bytes) falls within the specified
// range (min/max in GB). An empty bound means unbounded on that side.
func matchSize(minGB, maxGB string, sizeBytes int64) bool {
	if sizeBytes <= 0 {
		return false
	}
	sizeGB := float64(sizeBytes) / (1024 * 1024 * 1024)

	if minGB != "" {
		min, err := strconv.ParseFloat(minGB, 64)
		if err != nil {
			return false
		}
		if sizeGB < min {
			return false
		}
	}
	if maxGB != "" {
		max, err := strconv.ParseFloat(maxGB, 64)
		if err != nil {
			return false
		}
		if sizeGB > max {
			return false
		}
	}
	return true
}

// matchYear checks if the release year falls within [min, max].
// An empty bound means unbounded on that side.
func matchYear(minStr, maxStr string, year int) bool {
	if year <= 0 {
		return false
	}
	if minStr != "" {
		min, err := strconv.Atoi(minStr)
		if err != nil {
			return false
		}
		if year < min {
			return false
		}
	}
	if maxStr != "" {
		max, err := strconv.Atoi(maxStr)
		if err != nil {
			return false
		}
		if year > max {
			return false
		}
	}
	return true
}
