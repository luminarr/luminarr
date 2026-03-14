package tag_test

import (
	"context"
	"testing"

	"github.com/luminarr/luminarr/internal/core/tag"
	"github.com/luminarr/luminarr/internal/testutil"
)

func newSvc(t *testing.T) *tag.Service {
	t.Helper()
	q := testutil.NewTestDB(t)
	return tag.NewService(q)
}

// ── CRUD ────────────────────────────────────────────────────────────────────

func TestCreate(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)

	tg, err := svc.Create(ctx, "action")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tg.ID == "" {
		t.Fatal("Create returned empty ID")
	}
	if tg.Name != "action" {
		t.Errorf("Name = %q, want action", tg.Name)
	}
}

func TestCreate_Duplicate(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)

	if _, err := svc.Create(ctx, "dupe"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := svc.Create(ctx, "dupe")
	if err == nil {
		t.Fatal("expected error on duplicate create, got nil")
	}
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)

	created, _ := svc.Create(ctx, "horror")
	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "horror" {
		t.Errorf("Name = %q, want horror", got.Name)
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)

	svc.Create(ctx, "tag-a")
	svc.Create(ctx, "tag-b")

	tags, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("List returned %d tags, want 2", len(tags))
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)

	created, _ := svc.Create(ctx, "old-name")
	updated, err := svc.Update(ctx, created.ID, "new-name")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "new-name" {
		t.Errorf("Name = %q, want new-name", updated.Name)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)

	created, _ := svc.Create(ctx, "to-delete")
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	tags, _ := svc.List(ctx)
	if len(tags) != 0 {
		t.Errorf("List after delete returned %d tags, want 0", len(tags))
	}
}

// ── Movie tag associations ──────────────────────────────────────────────────

func TestMovieTags(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	svc := tag.NewService(q)
	movie := testutil.SeedMovie(t, q)

	t1, _ := svc.Create(ctx, "tag-1")
	t2, _ := svc.Create(ctx, "tag-2")

	// Set tags.
	if err := svc.SetMovieTags(ctx, movie.ID, []string{t1.ID, t2.ID}); err != nil {
		t.Fatalf("SetMovieTags: %v", err)
	}

	ids, err := svc.MovieTagIDs(ctx, movie.ID)
	if err != nil {
		t.Fatalf("MovieTagIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("MovieTagIDs returned %d, want 2", len(ids))
	}

	// Replace tags — should clear old and set new.
	if err := svc.SetMovieTags(ctx, movie.ID, []string{t1.ID}); err != nil {
		t.Fatalf("SetMovieTags (replace): %v", err)
	}
	ids, _ = svc.MovieTagIDs(ctx, movie.ID)
	if len(ids) != 1 {
		t.Fatalf("MovieTagIDs after replace returned %d, want 1", len(ids))
	}

	// Clear tags.
	if err := svc.SetMovieTags(ctx, movie.ID, []string{}); err != nil {
		t.Fatalf("SetMovieTags (clear): %v", err)
	}
	ids, _ = svc.MovieTagIDs(ctx, movie.ID)
	if len(ids) != 0 {
		t.Errorf("MovieTagIDs after clear returned %d, want 0", len(ids))
	}
}

func TestMovieTags_EmptyByDefault(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	svc := tag.NewService(q)
	movie := testutil.SeedMovie(t, q)

	ids, err := svc.MovieTagIDs(ctx, movie.ID)
	if err != nil {
		t.Fatalf("MovieTagIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("MovieTagIDs for untagged movie returned %d, want 0", len(ids))
	}
}

// ── Usage counts ────────────────────────────────────────────────────────────

func TestList_UsageCounts(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	svc := tag.NewService(q)
	movie := testutil.SeedMovie(t, q)

	tg, _ := svc.Create(ctx, "counted")
	svc.SetMovieTags(ctx, movie.ID, []string{tg.ID})

	tags, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("List returned %d tags, want 1", len(tags))
	}
	if tags[0].MovieCount != 1 {
		t.Errorf("MovieCount = %d, want 1", tags[0].MovieCount)
	}
}

// ── Cascade delete ──────────────────────────────────────────────────────────

func TestDelete_CascadeRemovesAssociations(t *testing.T) {
	ctx := context.Background()
	q := testutil.NewTestDB(t)
	svc := tag.NewService(q)
	movie := testutil.SeedMovie(t, q)

	tg, _ := svc.Create(ctx, "cascade-me")
	svc.SetMovieTags(ctx, movie.ID, []string{tg.ID})

	// Delete the tag — should cascade to movie_tags.
	if err := svc.Delete(ctx, tg.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	ids, _ := svc.MovieTagIDs(ctx, movie.ID)
	if len(ids) != 0 {
		t.Errorf("MovieTagIDs after tag delete returned %d, want 0", len(ids))
	}
}

// ── TagsOverlap ─────────────────────────────────────────────────────────────

func TestTagsOverlap(t *testing.T) {
	tests := []struct {
		name       string
		movieTags  []string
		entityTags []string
		want       bool
	}{
		{"entity untagged", []string{"a"}, []string{}, true},
		{"both untagged", []string{}, []string{}, true},
		{"overlap", []string{"a", "b"}, []string{"b", "c"}, true},
		{"no overlap", []string{"a"}, []string{"b", "c"}, false},
		{"movie untagged entity tagged", []string{}, []string{"a"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tag.TagsOverlap(tt.movieTags, tt.entityTags)
			if got != tt.want {
				t.Errorf("TagsOverlap(%v, %v) = %v, want %v",
					tt.movieTags, tt.entityTags, got, tt.want)
			}
		})
	}
}
