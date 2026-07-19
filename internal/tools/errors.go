package tools

import "fmt"

type ErrorCode string

const (
	CodeBrowserDisconnected       ErrorCode = "BROWSER_DISCONNECTED"
	CodeDiscoveryNotFound         ErrorCode = "DISCOVERY_NOT_FOUND"
	CodeDiscoveryStale            ErrorCode = "DISCOVERY_STALE"
	CodeProtocolVersionMismatch   ErrorCode = "PROTOCOL_VERSION_MISMATCH"
	CodeUpstreamUnavailable       ErrorCode = "UPSTREAM_UNAVAILABLE"
	CodeAuthenticationFailed      ErrorCode = "AUTHENTICATION_FAILED"
	CodeInvalidDuration           ErrorCode = "INVALID_DURATION"
	CodeTabNotFound               ErrorCode = "TAB_NOT_FOUND"
	CodeGroupNotFound             ErrorCode = "GROUP_NOT_FOUND"
	CodeTabNotOperable            ErrorCode = "TAB_NOT_OPERABLE"
	CodePlanInvalid               ErrorCode = "PLAN_INVALID"
	CodePlanStale                 ErrorCode = "PLAN_STALE"
	CodeContentPermissionRequired ErrorCode = "CONTENT_PERMISSION_REQUIRED"
	CodeContentNotAccessible      ErrorCode = "CONTENT_NOT_ACCESSIBLE"
	CodeContentExtractionFailed   ErrorCode = "CONTENT_EXTRACTION_FAILED"
	CodeContentStale              ErrorCode = "CONTENT_STALE"
	CodeUndoUnavailable           ErrorCode = "UNDO_UNAVAILABLE"
	CodeApplyFailedRolledBack     ErrorCode = "APPLY_FAILED_ROLLED_BACK"
	CodeApplyPartial              ErrorCode = "APPLY_PARTIAL"
	CodeInvalidArgument           ErrorCode = "INVALID_ARGUMENT"
	CodeCrossWindowGroup          ErrorCode = "CROSS_WINDOW_GROUP"
	CodePreviewExpired            ErrorCode = "PREVIEW_EXPIRED"
	CodePreviewNotFound           ErrorCode = "PREVIEW_NOT_FOUND"
	CodeConfirmationRequired      ErrorCode = "CONFIRMATION_REQUIRED"
	CodeTabCloseFailed            ErrorCode = "TAB_CLOSE_FAILED"
)

type Error struct {
	Code      ErrorCode      `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: message, Retryable: IsRetryable(code)}
}

func IsRetryable(code ErrorCode) bool {
	switch code {
	case CodeBrowserDisconnected, CodeDiscoveryNotFound, CodeDiscoveryStale, CodeUpstreamUnavailable,
		CodePlanStale, CodeContentStale, CodePreviewExpired:
		return true
	default:
		return false
	}
}

func (err *Error) Error() string {
	if err.Code == CodeContentPermissionRequired {
		origin, _ := err.Details["origin"].(string)
		action, _ := err.Details["action"].(string)
		if origin != "" || action != "" {
			return fmt.Sprintf("%s: %s (origin: %s; action: %s)", err.Code, err.Message, origin, action)
		}
	}
	return fmt.Sprintf("%s: %s", err.Code, err.Message)
}
