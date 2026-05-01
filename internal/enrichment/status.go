package enrichment

import apperrors "github.com/clawscli/claws/internal/errors"

// Status describes whether optional resource details were fetched and why they
// may be unavailable.
type Status string

const (
	Unknown       Status = ""
	Fetched       Status = "fetched"
	Configured    Status = "configured"
	NotConfigured Status = "not_configured"
	AccessDenied  Status = "access_denied"
	FetchFailed   Status = "fetch_failed"
)

func FailureStatus(err error) Status {
	if apperrors.IsAccessDenied(err) {
		return AccessDenied
	}
	return FetchFailed
}

func IsFailure(status Status) bool {
	return status == AccessDenied || status == FetchFailed
}

func Display(status Status) string {
	switch status {
	case AccessDenied:
		return "Unknown (access denied)"
	case FetchFailed:
		return "Unknown (fetch failed)"
	case NotConfigured:
		return "Not configured"
	default:
		return "Unknown"
	}
}
