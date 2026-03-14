package customformat

import (
	"testing"
)

func TestMatchRelease(t *testing.T) {
	tests := []struct {
		name    string
		formats []CustomFormat
		release ReleaseInfo
		want    []string
	}{
		{
			name:    "no formats",
			formats: nil,
			release: ReleaseInfo{Title: "anything"},
			want:    nil,
		},
		{
			name: "empty specifications never match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Empty", Specifications: nil},
			},
			release: ReleaseInfo{Title: "anything"},
			want:    nil,
		},
		{
			name: "single release_title regex match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "TrueHD", Specifications: []Specification{
					{Name: "TrueHD", Implementation: ImplReleaseTitle, Fields: map[string]string{"value": `(?i)\bTrueHD\b`}},
				}},
			},
			release: ReleaseInfo{Title: "Movie.2024.1080p.BluRay.TrueHD.Atmos.x265-GRP"},
			want:    []string{"cf1"},
		},
		{
			name: "release_title regex no match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "TrueHD", Specifications: []Specification{
					{Name: "TrueHD", Implementation: ImplReleaseTitle, Fields: map[string]string{"value": `(?i)\bTrueHD\b`}},
				}},
			},
			release: ReleaseInfo{Title: "Movie.2024.1080p.BluRay.DTS.x265-GRP"},
			want:    nil,
		},
		{
			name: "negated condition inverts match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Not x265", Specifications: []Specification{
					{Name: "x265", Implementation: ImplReleaseTitle, Negate: true, Fields: map[string]string{"value": `(?i)x265`}},
				}},
			},
			release: ReleaseInfo{Title: "Movie.2024.1080p.BluRay.x264-GRP"},
			want:    []string{"cf1"},
		},
		{
			name: "negated condition — match means fail",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Not x265", Specifications: []Specification{
					{Name: "x265", Implementation: ImplReleaseTitle, Negate: true, Fields: map[string]string{"value": `(?i)x265`}},
				}},
			},
			release: ReleaseInfo{Title: "Movie.2024.1080p.BluRay.x265-GRP"},
			want:    nil,
		},
		{
			name: "required conditions must all pass",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Both required", Specifications: []Specification{
					{Name: "BluRay", Implementation: ImplSource, Required: true, Fields: map[string]string{"value": "bluray"}},
					{Name: "1080p", Implementation: ImplResolution, Required: true, Fields: map[string]string{"value": "1080p"}},
				}},
			},
			release: ReleaseInfo{Source: "bluray", Resolution: "1080p"},
			want:    []string{"cf1"},
		},
		{
			name: "required conditions — one fails",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Both required", Specifications: []Specification{
					{Name: "BluRay", Implementation: ImplSource, Required: true, Fields: map[string]string{"value": "bluray"}},
					{Name: "1080p", Implementation: ImplResolution, Required: true, Fields: map[string]string{"value": "1080p"}},
				}},
			},
			release: ReleaseInfo{Source: "bluray", Resolution: "720p"},
			want:    nil,
		},
		{
			name: "optional conditions in same group — OR logic",
			formats: []CustomFormat{
				{ID: "cf1", Name: "HD sources", Specifications: []Specification{
					{Name: "BluRay", Implementation: ImplSource, Fields: map[string]string{"value": "bluray"}},
					{Name: "WEBDL", Implementation: ImplSource, Fields: map[string]string{"value": "webdl"}},
				}},
			},
			release: ReleaseInfo{Source: "webdl"},
			want:    []string{"cf1"},
		},
		{
			name: "optional conditions in same group — none match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "HD sources", Specifications: []Specification{
					{Name: "BluRay", Implementation: ImplSource, Fields: map[string]string{"value": "bluray"}},
					{Name: "WEBDL", Implementation: ImplSource, Fields: map[string]string{"value": "webdl"}},
				}},
			},
			release: ReleaseInfo{Source: "hdtv"},
			want:    nil,
		},
		{
			name: "mixed required + optional in same group",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Mixed", Specifications: []Specification{
					{Name: "BluRay req", Implementation: ImplSource, Required: true, Fields: map[string]string{"value": "bluray"}},
					{Name: "1080p opt", Implementation: ImplResolution, Fields: map[string]string{"value": "1080p"}},
					{Name: "2160p opt", Implementation: ImplResolution, Fields: map[string]string{"value": "2160p"}},
				}},
			},
			release: ReleaseInfo{Source: "bluray", Resolution: "2160p"},
			want:    []string{"cf1"},
		},
		{
			name: "indexer flag match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Freeleech", Specifications: []Specification{
					{Name: "FL", Implementation: ImplIndexerFlag, Fields: map[string]string{"value": "freeleech"}},
				}},
			},
			release: ReleaseInfo{IndexerFlags: []string{"freeleech", "internal"}},
			want:    []string{"cf1"},
		},
		{
			name: "indexer flag no match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Freeleech", Specifications: []Specification{
					{Name: "FL", Implementation: ImplIndexerFlag, Fields: map[string]string{"value": "freeleech"}},
				}},
			},
			release: ReleaseInfo{IndexerFlags: []string{"internal"}},
			want:    nil,
		},
		{
			name: "size within range",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Small", Specifications: []Specification{
					{Name: "size", Implementation: ImplSize, Fields: map[string]string{"min": "1", "max": "5"}},
				}},
			},
			release: ReleaseInfo{SizeBytes: 3 * 1024 * 1024 * 1024}, // 3 GB
			want:    []string{"cf1"},
		},
		{
			name: "size outside range",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Small", Specifications: []Specification{
					{Name: "size", Implementation: ImplSize, Fields: map[string]string{"min": "1", "max": "5"}},
				}},
			},
			release: ReleaseInfo{SizeBytes: 10 * 1024 * 1024 * 1024}, // 10 GB
			want:    nil,
		},
		{
			name: "size unbounded max",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Large", Specifications: []Specification{
					{Name: "size", Implementation: ImplSize, Fields: map[string]string{"min": "50"}},
				}},
			},
			release: ReleaseInfo{SizeBytes: 80 * 1024 * 1024 * 1024}, // 80 GB
			want:    []string{"cf1"},
		},
		{
			name: "year within range",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Recent", Specifications: []Specification{
					{Name: "year", Implementation: ImplYear, Fields: map[string]string{"min": "2020", "max": "2025"}},
				}},
			},
			release: ReleaseInfo{Year: 2023},
			want:    []string{"cf1"},
		},
		{
			name: "year outside range",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Recent", Specifications: []Specification{
					{Name: "year", Implementation: ImplYear, Fields: map[string]string{"min": "2020", "max": "2025"}},
				}},
			},
			release: ReleaseInfo{Year: 2019},
			want:    nil,
		},
		{
			name: "release group regex match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "HQ Groups", Specifications: []Specification{
					{Name: "grp", Implementation: ImplReleaseGroup, Fields: map[string]string{"value": `(?i)^(FraMeSToR|BHDStudio|hallowed)$`}},
				}},
			},
			release: ReleaseInfo{ReleaseGroup: "FraMeSToR"},
			want:    []string{"cf1"},
		},
		{
			name: "release group no match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "HQ Groups", Specifications: []Specification{
					{Name: "grp", Implementation: ImplReleaseGroup, Fields: map[string]string{"value": `(?i)^(FraMeSToR|BHDStudio)$`}},
				}},
			},
			release: ReleaseInfo{ReleaseGroup: "YIFY"},
			want:    nil,
		},
		{
			name: "edition regex match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Extended", Specifications: []Specification{
					{Name: "ext", Implementation: ImplEdition, Fields: map[string]string{"value": `(?i)\bextended\b`}},
				}},
			},
			release: ReleaseInfo{Edition: "Extended Cut"},
			want:    []string{"cf1"},
		},
		{
			name: "language match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "English", Specifications: []Specification{
					{Name: "lang", Implementation: ImplLanguage, Fields: map[string]string{"value": "english"}},
				}},
			},
			release: ReleaseInfo{Languages: []string{"English", "French"}},
			want:    []string{"cf1"},
		},
		{
			name: "quality modifier match",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Remux", Specifications: []Specification{
					{Name: "mod", Implementation: ImplQualityModifier, Fields: map[string]string{"value": "remux"}},
				}},
			},
			release: ReleaseInfo{Modifier: "remux"},
			want:    []string{"cf1"},
		},
		{
			name: "multiple formats — some match some don't",
			formats: []CustomFormat{
				{ID: "cf1", Name: "x265", Specifications: []Specification{
					{Name: "x265", Implementation: ImplReleaseTitle, Fields: map[string]string{"value": `(?i)x265`}},
				}},
				{ID: "cf2", Name: "DTS", Specifications: []Specification{
					{Name: "DTS", Implementation: ImplReleaseTitle, Fields: map[string]string{"value": `(?i)\bDTS\b`}},
				}},
				{ID: "cf3", Name: "FLAC", Specifications: []Specification{
					{Name: "FLAC", Implementation: ImplReleaseTitle, Fields: map[string]string{"value": `(?i)\bFLAC\b`}},
				}},
			},
			release: ReleaseInfo{Title: "Movie.2024.1080p.BluRay.DTS.x265-GRP"},
			want:    []string{"cf1", "cf2"},
		},
		{
			name: "invalid regex returns false",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Bad", Specifications: []Specification{
					{Name: "bad", Implementation: ImplReleaseTitle, Fields: map[string]string{"value": `[invalid`}},
				}},
			},
			release: ReleaseInfo{Title: "anything"},
			want:    nil,
		},
		{
			name: "cross-group required + optional",
			formats: []CustomFormat{
				{ID: "cf1", Name: "Complex", Specifications: []Specification{
					{Name: "BluRay", Implementation: ImplSource, Required: true, Fields: map[string]string{"value": "bluray"}},
					{Name: "x265", Implementation: ImplReleaseTitle, Fields: map[string]string{"value": `(?i)x265`}},
					{Name: "x264", Implementation: ImplReleaseTitle, Fields: map[string]string{"value": `(?i)x264`}},
				}},
			},
			release: ReleaseInfo{Source: "bluray", Title: "Movie.x264"},
			want:    []string{"cf1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchRelease(tt.formats, tt.release)
			if len(got) != len(tt.want) {
				t.Fatalf("MatchRelease() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("MatchRelease()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestScoreRelease(t *testing.T) {
	tests := []struct {
		name          string
		matched       []string
		profileScores map[string]int
		want          int
	}{
		{
			name:          "no matches",
			matched:       nil,
			profileScores: map[string]int{"cf1": 100},
			want:          0,
		},
		{
			name:          "single match",
			matched:       []string{"cf1"},
			profileScores: map[string]int{"cf1": 1750},
			want:          1750,
		},
		{
			name:          "multiple matches sum scores",
			matched:       []string{"cf1", "cf2"},
			profileScores: map[string]int{"cf1": 1750, "cf2": -500, "cf3": 100},
			want:          1250,
		},
		{
			name:          "matched format not in profile scores",
			matched:       []string{"cf1", "cf_missing"},
			profileScores: map[string]int{"cf1": 500},
			want:          500,
		},
		{
			name:          "negative scores",
			matched:       []string{"cf1"},
			profileScores: map[string]int{"cf1": -10000},
			want:          -10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreRelease(tt.matched, tt.profileScores)
			if got != tt.want {
				t.Errorf("ScoreRelease() = %d, want %d", got, tt.want)
			}
		})
	}
}
