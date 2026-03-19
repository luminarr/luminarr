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

func TestExplain_GrabbedWithZeroCFScore(t *testing.T) {
	t.Parallel()
	ctx := ExplainContext{
		ProfileName:  "HD-1080p",
		CFScore:      0,
		QualityScore: 72,
		TotalScore:   72,
	}
	text := Explain(ReasonGrabbed, ctx)
	// With CF score of 0, the CF text should not appear.
	if strings.Contains(text, "CF") {
		t.Errorf("expected no CF mention when CFScore is 0, got %q", text)
	}
	if !strings.Contains(text, "Grabbed") {
		t.Errorf("expected 'Grabbed' in explanation, got %q", text)
	}
}

func TestExplain_GrabbedWithPositiveCFScore(t *testing.T) {
	t.Parallel()
	ctx := ExplainContext{
		ProfileName:  "HD-1080p",
		CFScore:      25,
		QualityScore: 72,
		TotalScore:   97,
	}
	text := Explain(ReasonGrabbed, ctx)
	if !strings.Contains(text, "CF 25") {
		t.Errorf("expected 'CF 25' in explanation, got %q", text)
	}
}

func TestExplain_EverySkipReasonNonEmpty(t *testing.T) {
	t.Parallel()
	allReasons := []SkipReason{
		ReasonGrabbed,
		ReasonBlocklisted,
		ReasonCFScoreBelowMinimum,
		ReasonQualityNotAllowed,
		ReasonNoUpgradeNeeded,
		ReasonUpgradeDisabled,
		ReasonEditionNotPreferred,
		ReasonDownloadClientReject,
		ReasonAlreadyDownloading,
	}
	ctx := ExplainContext{
		ProfileName: "TestProfile",
		CurrentFile: "1080p BluRay",
		QualityName: "720p WEB-DL",
	}
	for _, reason := range allReasons {
		text := Explain(reason, ctx)
		if text == "" {
			t.Errorf("Explain(%q) returned empty string", reason)
		}
	}
}

func TestExplain_UnknownReasonStillProducesOutput(t *testing.T) {
	t.Parallel()
	text := Explain("totally_unknown_reason", ExplainContext{})
	if text == "" {
		t.Fatal("expected non-empty output for unknown reason")
	}
	if !strings.Contains(text, "totally_unknown_reason") {
		t.Errorf("expected unknown reason string in output, got %q", text)
	}
}

func TestExplain_ProfileNameInQualityNotAllowed(t *testing.T) {
	t.Parallel()
	ctx := ExplainContext{
		ProfileName: "Ultra HD 4K",
		QualityName: "720p WEB-DL",
	}
	text := Explain(ReasonQualityNotAllowed, ctx)
	if !strings.Contains(text, "Ultra HD 4K") {
		t.Errorf("expected profile name in quality_not_in_profile explanation, got %q", text)
	}
}

func TestExplain_ProfileNameInUpgradeDisabled(t *testing.T) {
	t.Parallel()
	ctx := ExplainContext{
		ProfileName: "Standard Def",
	}
	text := Explain(ReasonUpgradeDisabled, ctx)
	if !strings.Contains(text, "Standard Def") {
		t.Errorf("expected profile name in upgrade_disabled explanation, got %q", text)
	}
}

func TestExplain_CurrentFileLabelInNoUpgradeNeeded(t *testing.T) {
	t.Parallel()
	ctx := ExplainContext{
		CurrentFile: "2160p Remux",
		QualityName: "1080p WEB-DL",
	}
	text := Explain(ReasonNoUpgradeNeeded, ctx)
	if !strings.Contains(text, "2160p Remux") {
		t.Errorf("expected current file label in no_upgrade_needed explanation, got %q", text)
	}
}
