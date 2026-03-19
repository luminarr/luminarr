package autosearch

import "github.com/luminarr/luminarr/pkg/plugin"

// SkipReason identifies why a release candidate was not grabbed.
type SkipReason string

const (
	ReasonGrabbed              SkipReason = "grabbed"
	ReasonBlocklisted          SkipReason = "blocklisted"
	ReasonCFScoreBelowMinimum  SkipReason = "cf_score_below_minimum"
	ReasonQualityNotAllowed    SkipReason = "quality_not_in_profile"
	ReasonNoUpgradeNeeded      SkipReason = "no_upgrade_needed"
	ReasonUpgradeDisabled      SkipReason = "upgrade_disabled"
	ReasonEditionNotPreferred  SkipReason = "edition_not_preferred"
	ReasonDownloadClientReject SkipReason = "download_client_rejected"
	ReasonAlreadyDownloading   SkipReason = "already_downloading"
)

// ReleaseDecision records the outcome for one candidate release during search.
type ReleaseDecision struct {
	Title          string                 `json:"title"`
	GUID           string                 `json:"guid"`
	Outcome        string                 `json:"outcome"` // "grabbed", "skipped"
	Reason         SkipReason             `json:"reason"`
	Explanation    string                 `json:"explanation"`
	QualityScore   int                    `json:"quality_score"`
	CFScore        int                    `json:"cf_score"`
	MatchedFormats []string               `json:"matched_formats,omitempty"`
	Breakdown      *plugin.ScoreBreakdown `json:"breakdown,omitempty"`
}
