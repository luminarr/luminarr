package customformat

import (
	"testing"
)

func TestTrashToLuminarrMapping(t *testing.T) {
	// Verify all TRaSH implementation names map to known Luminarr types.
	expected := map[string]string{
		"ReleaseTitleSpecification":    ImplReleaseTitle,
		"EditionSpecification":         ImplEdition,
		"LanguageSpecification":        ImplLanguage,
		"IndexerFlagSpecification":     ImplIndexerFlag,
		"SourceSpecification":          ImplSource,
		"ResolutionSpecification":      ImplResolution,
		"QualityModifierSpecification": ImplQualityModifier,
		"SizeSpecification":            ImplSize,
		"ReleaseGroupSpecification":    ImplReleaseGroup,
		"YearSpecification":            ImplYear,
	}

	for trash, want := range expected {
		got, ok := trashToLuminarr[trash]
		if !ok {
			t.Errorf("trashToLuminarr missing key %q", trash)
			continue
		}
		if got != want {
			t.Errorf("trashToLuminarr[%q] = %q, want %q", trash, got, want)
		}
	}

	if len(trashToLuminarr) != len(expected) {
		t.Errorf("trashToLuminarr has %d entries, expected %d", len(trashToLuminarr), len(expected))
	}
}

func TestLuminarrToTrashRoundTrip(t *testing.T) {
	// Every Luminarr impl should map back to a TRaSH impl and vice versa.
	for trash, luminarr := range trashToLuminarr {
		roundTrip, ok := luminarrToTrash[luminarr]
		if !ok {
			t.Errorf("luminarrToTrash missing key %q (from TRaSH %q)", luminarr, trash)
			continue
		}
		if roundTrip != trash {
			t.Errorf("round trip failed: %q → %q → %q, want %q", trash, luminarr, roundTrip, trash)
		}
	}
}

func TestParseTrashJSON(t *testing.T) {
	// Simulate the parsing that Import does without hitting the DB.
	input := `{
		"trash_id": "abc123",
		"trash_scores": {"default": 1750},
		"name": "TrueHD ATMOS",
		"includeCustomFormatWhenRenaming": false,
		"specifications": [
			{
				"name": "TrueHD ATMOS",
				"implementation": "ReleaseTitleSpecification",
				"negate": false,
				"required": true,
				"fields": {"value": "(?i)\\bTrueHD\\.?\\s?Atmos\\b"}
			},
			{
				"name": "Not DTS",
				"implementation": "ReleaseTitleSpecification",
				"negate": true,
				"required": false,
				"fields": {"value": "(?i)\\bDTS\\b"}
			}
		]
	}`

	var formats []trashFormat
	if err := parseTrashJSON([]byte(input), &formats); err != nil {
		t.Fatalf("parseTrashJSON() error: %v", err)
	}

	if len(formats) != 1 {
		t.Fatalf("expected 1 format, got %d", len(formats))
	}

	tf := formats[0]
	if tf.Name != "TrueHD ATMOS" {
		t.Errorf("name = %q, want %q", tf.Name, "TrueHD ATMOS")
	}
	if tf.TrashID != "abc123" {
		t.Errorf("trash_id = %q, want %q", tf.TrashID, "abc123")
	}
	if len(tf.Specifications) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(tf.Specifications))
	}
	if tf.Specifications[0].Implementation != "ReleaseTitleSpecification" {
		t.Errorf("spec[0].implementation = %q, want %q", tf.Specifications[0].Implementation, "ReleaseTitleSpecification")
	}
	if !tf.Specifications[0].Required {
		t.Error("spec[0].required should be true")
	}
	if !tf.Specifications[1].Negate {
		t.Error("spec[1].negate should be true")
	}
}

func TestParseTrashJSONArray(t *testing.T) {
	input := `[
		{"trash_id": "a", "name": "Format A", "specifications": []},
		{"trash_id": "b", "name": "Format B", "specifications": []}
	]`

	var formats []trashFormat
	if err := parseTrashJSON([]byte(input), &formats); err != nil {
		t.Fatalf("parseTrashJSON() error: %v", err)
	}

	if len(formats) != 2 {
		t.Fatalf("expected 2 formats, got %d", len(formats))
	}
	if formats[0].Name != "Format A" {
		t.Errorf("formats[0].name = %q, want %q", formats[0].Name, "Format A")
	}
	if formats[1].Name != "Format B" {
		t.Errorf("formats[1].name = %q, want %q", formats[1].Name, "Format B")
	}
}
