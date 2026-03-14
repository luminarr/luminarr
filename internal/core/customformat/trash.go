package customformat

import (
	"context"
	"encoding/json"
	"fmt"
)

// trashFormat is the JSON structure used by TRaSH Guides for custom format distribution.
type trashFormat struct {
	TrashID                         string         `json:"trash_id"`
	TrashScores                     map[string]int `json:"trash_scores,omitempty"`
	Name                            string         `json:"name"`
	IncludeCustomFormatWhenRenaming bool           `json:"includeCustomFormatWhenRenaming"`
	Specifications                  []trashSpec    `json:"specifications"`
}

type trashSpec struct {
	Name           string            `json:"name"`
	Implementation string            `json:"implementation"`
	Negate         bool              `json:"negate"`
	Required       bool              `json:"required"`
	Fields         map[string]string `json:"fields"`
}

// TRaSH → Luminarr implementation name mapping.
var trashToLuminarr = map[string]string{
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

// Luminarr → TRaSH implementation name mapping (reverse).
var luminarrToTrash = func() map[string]string {
	m := make(map[string]string, len(trashToLuminarr))
	for k, v := range trashToLuminarr {
		m[v] = k
	}
	return m
}()

// parseTrashJSON parses TRaSH-format JSON (single format or array) into trashFormat structs.
func parseTrashJSON(data []byte, out *[]trashFormat) error {
	if err := json.Unmarshal(data, out); err != nil {
		var single trashFormat
		if err2 := json.Unmarshal(data, &single); err2 != nil {
			return fmt.Errorf("invalid TRaSH JSON: %w", err2)
		}
		*out = []trashFormat{single}
	}
	return nil
}

// Import parses TRaSH-format JSON (single format or array) and creates custom formats.
// Returns the created formats.
func (s *Service) Import(ctx context.Context, data []byte) ([]CustomFormat, error) {
	var formats []trashFormat
	if err := parseTrashJSON(data, &formats); err != nil {
		return nil, err
	}

	var created []CustomFormat
	for _, tf := range formats {
		specs := make([]Specification, 0, len(tf.Specifications))
		for _, ts := range tf.Specifications {
			impl, ok := trashToLuminarr[ts.Implementation]
			if !ok {
				impl = ts.Implementation // pass through unknown types
			}
			specs = append(specs, Specification{
				Name:           ts.Name,
				Implementation: impl,
				Negate:         ts.Negate,
				Required:       ts.Required,
				Fields:         ts.Fields,
			})
		}

		cf, err := s.Create(ctx, CreateRequest{
			Name:                tf.Name,
			IncludeWhenRenaming: tf.IncludeCustomFormatWhenRenaming,
			Specifications:      specs,
		})
		if err != nil {
			return created, fmt.Errorf("importing %q: %w", tf.Name, err)
		}
		created = append(created, cf)
	}

	return created, nil
}

// Export serializes the given custom format IDs as TRaSH-compatible JSON.
// If ids is empty, all formats are exported.
func (s *Service) Export(ctx context.Context, ids []string) ([]byte, error) {
	var formats []CustomFormat
	var err error

	if len(ids) == 0 {
		formats, err = s.List(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		formats = make([]CustomFormat, 0, len(ids))
		for _, id := range ids {
			cf, err := s.Get(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("exporting format %q: %w", id, err)
			}
			formats = append(formats, cf)
		}
	}

	trash := make([]trashFormat, len(formats))
	for i, cf := range formats {
		specs := make([]trashSpec, len(cf.Specifications))
		for j, s := range cf.Specifications {
			impl, ok := luminarrToTrash[s.Implementation]
			if !ok {
				impl = s.Implementation
			}
			specs[j] = trashSpec{
				Name:           s.Name,
				Implementation: impl,
				Negate:         s.Negate,
				Required:       s.Required,
				Fields:         s.Fields,
			}
		}
		trash[i] = trashFormat{
			TrashID:                         cf.ID,
			Name:                            cf.Name,
			IncludeCustomFormatWhenRenaming: cf.IncludeWhenRenaming,
			Specifications:                  specs,
		}
	}

	return json.MarshalIndent(trash, "", "  ")
}
