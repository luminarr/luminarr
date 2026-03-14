package radarrimport

import (
	"testing"
)

func TestQualities_LeafAllowed(t *testing.T) {
	item := radarrProfileItem{
		Quality: radarrProfileQuality{ID: 7, Name: "Bluray-1080p"},
		Allowed: true,
	}
	got := item.qualities()
	if len(got) != 1 || got[0].Name != "Bluray-1080p" {
		t.Fatalf("expected [Bluray-1080p], got %v", got)
	}
}

func TestQualities_LeafNotAllowed_StillIncluded(t *testing.T) {
	item := radarrProfileItem{
		Quality: radarrProfileQuality{ID: 7, Name: "Bluray-1080p"},
		Allowed: false,
	}
	got := item.qualities()
	if len(got) != 1 || got[0].Name != "Bluray-1080p" {
		t.Fatalf("expected [Bluray-1080p] even when Allowed=false, got %v", got)
	}
}

func TestQualities_GroupPlaceholder_Skipped(t *testing.T) {
	// Radarr groups have a placeholder quality with ID=0 at the group level.
	item := radarrProfileItem{
		Quality: radarrProfileQuality{ID: 0, Name: ""},
		Allowed: false,
	}
	got := item.qualities()
	if len(got) != 0 {
		t.Fatalf("expected empty (ID=0 placeholder), got %v", got)
	}
}

func TestQualities_GroupRecursesChildren(t *testing.T) {
	group := radarrProfileItem{
		Quality: radarrProfileQuality{ID: 0, Name: ""},
		Allowed: true,
		Items: []radarrProfileItem{
			{Quality: radarrProfileQuality{ID: 4, Name: "HDTV-720p"}, Allowed: true},
			{Quality: radarrProfileQuality{ID: 5, Name: "WEBDL-720p"}, Allowed: false},
			{Quality: radarrProfileQuality{ID: 14, Name: "WEBRip-720p"}, Allowed: true},
		},
	}
	got := group.qualities()
	if len(got) != 3 {
		t.Fatalf("expected 3 qualities from group (all children regardless of Allowed), got %d: %v", len(got), got)
	}
	names := make([]string, len(got))
	for i, q := range got {
		names[i] = q.Name
	}
	expected := []string{"HDTV-720p", "WEBDL-720p", "WEBRip-720p"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("quality[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestMapProfile_AllItemsIncluded(t *testing.T) {
	// Simulate Radarr's "Any" profile: items at top level, some allowed, some not.
	profile := radarrProfile{
		ID:             1,
		Name:           "Any",
		UpgradeAllowed: false,
		Cutoff:         7,
		Items: []radarrProfileItem{
			{Quality: radarrProfileQuality{ID: 1, Name: "SDTV"}, Allowed: false},
			{Quality: radarrProfileQuality{ID: 7, Name: "Bluray-1080p"}, Allowed: true},
			{Quality: radarrProfileQuality{ID: 3, Name: "WEBDL-1080p"}, Allowed: false},
		},
	}
	req := mapProfile(profile)
	if len(req.Qualities) != 3 {
		t.Fatalf("expected 3 qualities (all items regardless of Allowed), got %d", len(req.Qualities))
	}
}

func TestMapProfile_NestedGroups(t *testing.T) {
	// Simulate Radarr's "HD" profile with grouped items.
	profile := radarrProfile{
		ID:             2,
		Name:           "HD",
		UpgradeAllowed: true,
		Cutoff:         4,
		Items: []radarrProfileItem{
			// A group containing 720p qualities
			{
				Quality: radarrProfileQuality{ID: 0, Name: ""},
				Allowed: true,
				Items: []radarrProfileItem{
					{Quality: radarrProfileQuality{ID: 4, Name: "HDTV-720p"}, Allowed: true},
					{Quality: radarrProfileQuality{ID: 5, Name: "WEBDL-720p"}, Allowed: false},
				},
			},
			// A standalone item
			{Quality: radarrProfileQuality{ID: 7, Name: "Bluray-1080p"}, Allowed: false},
		},
	}
	req := mapProfile(profile)
	if len(req.Qualities) != 3 {
		t.Fatalf("expected 3 qualities (2 from group + 1 standalone), got %d", len(req.Qualities))
	}
}
