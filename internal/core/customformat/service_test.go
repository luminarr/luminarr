package customformat_test

import (
	"context"
	"testing"

	"github.com/luminarr/luminarr/internal/core/customformat"
	dbsqlite "github.com/luminarr/luminarr/internal/db/generated/sqlite"
	"github.com/luminarr/luminarr/internal/testutil"
)

func newTestService(t *testing.T) *customformat.Service {
	t.Helper()
	q := testutil.NewTestDB(t)
	return customformat.NewService(q)
}

func newTestServiceWithDB(t *testing.T) (*customformat.Service, *dbsqlite.Queries) {
	t.Helper()
	q := testutil.NewTestDB(t)
	return customformat.NewService(q), q
}

func sampleSpecs() []customformat.Specification {
	return []customformat.Specification{
		{
			Name:           "TrueHD",
			Implementation: customformat.ImplReleaseTitle,
			Fields:         map[string]string{"value": `(?i)\bTrueHD\b`},
		},
	}
}

func TestService_CRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Create
	cf, err := svc.Create(ctx, customformat.CreateRequest{
		Name:                "TrueHD ATMOS",
		IncludeWhenRenaming: true,
		Specifications:      sampleSpecs(),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cf.ID == "" {
		t.Fatal("Create: expected non-empty ID")
	}
	if cf.Name != "TrueHD ATMOS" {
		t.Errorf("Create: name = %q, want %q", cf.Name, "TrueHD ATMOS")
	}
	if !cf.IncludeWhenRenaming {
		t.Error("Create: include_when_renaming should be true")
	}
	if len(cf.Specifications) != 1 {
		t.Fatalf("Create: expected 1 spec, got %d", len(cf.Specifications))
	}
	if cf.Specifications[0].Implementation != customformat.ImplReleaseTitle {
		t.Errorf("Create: spec[0].implementation = %q, want %q", cf.Specifications[0].Implementation, customformat.ImplReleaseTitle)
	}

	// Get
	got, err := svc.Get(ctx, cf.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != cf.Name {
		t.Errorf("Get: name = %q, want %q", got.Name, cf.Name)
	}

	// List
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List: expected 1, got %d", len(list))
	}

	// Update
	updated, err := svc.Update(ctx, cf.ID, customformat.UpdateRequest{
		Name:                "TrueHD Atmos v2",
		IncludeWhenRenaming: false,
		Specifications: []customformat.Specification{
			{Name: "TrueHD", Implementation: customformat.ImplReleaseTitle, Fields: map[string]string{"value": `(?i)\bTrueHD\b`}},
			{Name: "Atmos", Implementation: customformat.ImplReleaseTitle, Fields: map[string]string{"value": `(?i)\bAtmos\b`}},
		},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "TrueHD Atmos v2" {
		t.Errorf("Update: name = %q, want %q", updated.Name, "TrueHD Atmos v2")
	}
	if updated.IncludeWhenRenaming {
		t.Error("Update: include_when_renaming should be false")
	}
	if len(updated.Specifications) != 2 {
		t.Errorf("Update: expected 2 specs, got %d", len(updated.Specifications))
	}

	// Delete
	if err := svc.Delete(ctx, cf.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify deleted
	list, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List after delete: expected 0, got %d", len(list))
	}
}

func TestService_CreateDuplicateName(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, customformat.CreateRequest{
		Name:           "DTS-X",
		Specifications: sampleSpecs(),
	})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err = svc.Create(ctx, customformat.CreateRequest{
		Name:           "DTS-X",
		Specifications: sampleSpecs(),
	})
	if err == nil {
		t.Fatal("expected error on duplicate name, got nil")
	}
}

func TestService_Scores(t *testing.T) {
	svc, q := newTestServiceWithDB(t)
	ctx := context.Background()

	// Create two formats to use as score targets.
	cf1, err := svc.Create(ctx, customformat.CreateRequest{
		Name:           "Format A",
		Specifications: sampleSpecs(),
	})
	if err != nil {
		t.Fatalf("Create cf1: %v", err)
	}
	cf2, err := svc.Create(ctx, customformat.CreateRequest{
		Name:           "Format B",
		Specifications: sampleSpecs(),
	})
	if err != nil {
		t.Fatalf("Create cf2: %v", err)
	}

	// Create a quality profile so FK constraints are satisfied.
	profileID := "test-profile-id"
	_, err = q.CreateQualityProfile(ctx, dbsqlite.CreateQualityProfileParams{
		ID:            profileID,
		Name:          "Test Profile",
		CutoffJson:    `{"resolution":"1080p","source":"bluray","codec":"x264","hdr":"none","name":"Bluray-1080p"}`,
		QualitiesJson: `[{"resolution":"1080p","source":"bluray","codec":"x264","hdr":"none","name":"Bluray-1080p"}]`,
		CreatedAt:     "2025-01-01T00:00:00Z",
		UpdatedAt:     "2025-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("CreateQualityProfile: %v", err)
	}

	// Initially no scores.
	scores, err := svc.ListScores(ctx, profileID)
	if err != nil {
		t.Fatalf("ListScores: %v", err)
	}
	if len(scores) != 0 {
		t.Errorf("ListScores: expected 0, got %d", len(scores))
	}

	// Set scores.
	err = svc.SetScores(ctx, profileID, map[string]int{
		cf1.ID: 1750,
		cf2.ID: -500,
	})
	if err != nil {
		t.Fatalf("SetScores: %v", err)
	}

	scores, err = svc.ListScores(ctx, profileID)
	if err != nil {
		t.Fatalf("ListScores after set: %v", err)
	}
	if len(scores) != 2 {
		t.Fatalf("ListScores: expected 2, got %d", len(scores))
	}
	if scores[cf1.ID] != 1750 {
		t.Errorf("score[cf1] = %d, want 1750", scores[cf1.ID])
	}
	if scores[cf2.ID] != -500 {
		t.Errorf("score[cf2] = %d, want -500", scores[cf2.ID])
	}

	// Overwrite scores — should replace, not accumulate.
	err = svc.SetScores(ctx, profileID, map[string]int{
		cf1.ID: 100,
	})
	if err != nil {
		t.Fatalf("SetScores overwrite: %v", err)
	}

	scores, err = svc.ListScores(ctx, profileID)
	if err != nil {
		t.Fatalf("ListScores after overwrite: %v", err)
	}
	if len(scores) != 1 {
		t.Fatalf("ListScores: expected 1 after overwrite, got %d", len(scores))
	}
	if scores[cf1.ID] != 100 {
		t.Errorf("score[cf1] = %d, want 100", scores[cf1.ID])
	}
}

func TestService_ImportExport(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	trashJSON := `{
		"trash_id": "abc123",
		"trash_scores": {"default": 1750},
		"name": "TrueHD ATMOS",
		"includeCustomFormatWhenRenaming": true,
		"specifications": [
			{
				"name": "TrueHD",
				"implementation": "ReleaseTitleSpecification",
				"negate": false,
				"required": true,
				"fields": {"value": "(?i)\\bTrueHD\\b"}
			}
		]
	}`

	// Import
	created, err := svc.Import(ctx, []byte(trashJSON))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("Import: expected 1, got %d", len(created))
	}
	if created[0].Name != "TrueHD ATMOS" {
		t.Errorf("Import: name = %q, want %q", created[0].Name, "TrueHD ATMOS")
	}
	if !created[0].IncludeWhenRenaming {
		t.Error("Import: include_when_renaming should be true")
	}
	// Implementation should be converted from TRaSH to Luminarr format.
	if created[0].Specifications[0].Implementation != customformat.ImplReleaseTitle {
		t.Errorf("Import: spec[0].implementation = %q, want %q",
			created[0].Specifications[0].Implementation, customformat.ImplReleaseTitle)
	}

	// Export all
	data, err := svc.Export(ctx, nil)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Export: expected non-empty output")
	}

	// Export by ID
	data, err = svc.Export(ctx, []string{created[0].ID})
	if err != nil {
		t.Fatalf("Export by ID: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Export by ID: expected non-empty output")
	}
}

func TestService_ImportArray(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	trashJSON := `[
		{
			"trash_id": "a",
			"name": "Format A",
			"specifications": [
				{"name": "x265", "implementation": "ReleaseTitleSpecification", "fields": {"value": "x265"}}
			]
		},
		{
			"trash_id": "b",
			"name": "Format B",
			"specifications": [
				{"name": "remux", "implementation": "QualityModifierSpecification", "fields": {"value": "remux"}}
			]
		}
	]`

	created, err := svc.Import(ctx, []byte(trashJSON))
	if err != nil {
		t.Fatalf("Import array: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("Import array: expected 2, got %d", len(created))
	}
}
