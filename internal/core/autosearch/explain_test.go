package autosearch

import (
	"strings"
	"testing"
)

func TestExplain_AllReasons(t *testing.T) {
	t.Parallel()

	ctx := ExplainContext{
		ProfileName:    "HD-1080p",
		MinCFScore:     0,
		CurrentFile:    "1080p BluRay",
		CFScore:        -150,
		MatchedFormats: []string{"x265 (HD)"},
		QualityName:    "720p WEB-DL",
		QualityScore:   72,
		TotalScore:     92,
	}

	tests := []struct {
		reason   SkipReason
		contains string
	}{
		{ReasonGrabbed, "Grabbed: best candidate"},
		{ReasonBlocklisted, "blocklist"},
		{ReasonCFScoreBelowMinimum, "custom format score"},
		{ReasonQualityNotAllowed, "not in the allowed set"},
		{ReasonNoUpgradeNeeded, "already meets or exceeds"},
		{ReasonUpgradeDisabled, "upgrading is disabled"},
		{ReasonEditionNotPreferred, "preferred edition"},
		{ReasonDownloadClientReject, "download client rejected"},
		{ReasonAlreadyDownloading, "already active"},
	}

	for _, tc := range tests {
		t.Run(string(tc.reason), func(t *testing.T) {
			t.Parallel()
			text := Explain(tc.reason, ctx)
			if text == "" {
				t.Fatal("explanation is empty")
			}
			if !strings.Contains(strings.ToLower(text), strings.ToLower(tc.contains)) {
				t.Errorf("explanation %q does not contain %q", text, tc.contains)
			}
		})
	}
}

func TestExplain_CFScoreIncludesFormats(t *testing.T) {
	t.Parallel()
	ctx := ExplainContext{
		ProfileName:    "Test",
		MinCFScore:     100,
		CFScore:        -50,
		MatchedFormats: []string{"Bad Groups", "x265 HD"},
	}
	text := Explain(ReasonCFScoreBelowMinimum, ctx)
	if !strings.Contains(text, "Bad Groups") {
		t.Errorf("expected matched formats in explanation: %q", text)
	}
}

func TestExplain_CFScoreNoFormats(t *testing.T) {
	t.Parallel()
	ctx := ExplainContext{
		ProfileName: "Test",
		MinCFScore:  100,
		CFScore:     0,
	}
	text := Explain(ReasonCFScoreBelowMinimum, ctx)
	if !strings.Contains(text, "none") {
		t.Errorf("expected 'none' for empty formats: %q", text)
	}
}
