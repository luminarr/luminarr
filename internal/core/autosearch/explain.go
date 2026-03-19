package autosearch

import "fmt"

// ExplainContext provides the details needed to generate human-readable explanations.
type ExplainContext struct {
	ProfileName    string
	MinCFScore     int
	CurrentFile    string // e.g. "1080p BluRay"
	CFScore        int
	MatchedFormats []string
	QualityName    string // e.g. "720p WEB-DL"
	QualityScore   int
	TotalScore     int
}

// Explain generates a human-readable explanation for a skip or grab decision.
func Explain(reason SkipReason, ctx ExplainContext) string {
	switch reason {
	case ReasonGrabbed:
		cf := ""
		if ctx.CFScore != 0 {
			cf = fmt.Sprintf(" + CF %d", ctx.CFScore)
		}
		return fmt.Sprintf("Grabbed: best candidate scoring %d (quality %d%s). Passed all profile checks.",
			ctx.TotalScore, ctx.QualityScore, cf)

	case ReasonBlocklisted:
		return "Skipped: this release is on your blocklist."

	case ReasonCFScoreBelowMinimum:
		matched := "none"
		if len(ctx.MatchedFormats) > 0 {
			matched = fmt.Sprintf("%v", ctx.MatchedFormats)
		}
		return fmt.Sprintf("Skipped: custom format score %d is below the minimum of %d required by profile %q. Matched formats: %s.",
			ctx.CFScore, ctx.MinCFScore, ctx.ProfileName, matched)

	case ReasonQualityNotAllowed:
		return fmt.Sprintf("Skipped: quality %q is not in the allowed set for profile %q.",
			ctx.QualityName, ctx.ProfileName)

	case ReasonNoUpgradeNeeded:
		return fmt.Sprintf("Skipped: your current file (%s) already meets or exceeds this release (%s).",
			ctx.CurrentFile, ctx.QualityName)

	case ReasonUpgradeDisabled:
		return fmt.Sprintf("Skipped: your current file meets the cutoff and upgrading is disabled in profile %q.",
			ctx.ProfileName)

	case ReasonEditionNotPreferred:
		return "Skipped: release did not match the preferred edition and quality profile rejected it."

	case ReasonDownloadClientReject:
		return "Skipped: download client rejected this release. It was auto-blocklisted."

	case ReasonAlreadyDownloading:
		return "Skipped: another grab for this movie is already active."

	default:
		return fmt.Sprintf("Skipped: %s", string(reason))
	}
}
