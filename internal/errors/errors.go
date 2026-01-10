// Package errors provides error classification and handling for AWS operations.
package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/smithy-go"

	"github.com/clawscli/claws/internal/log"
)

// Kind represents the classification of an error.
type Kind int

const (
	Unknown    Kind = iota // Unknown or unclassified error
	Auth                   // Authentication/authorization errors (AccessDenied, Forbidden)
	Throttling             // Rate limiting errors (TooManyRequests)
	NotFound               // Resource not found errors
	InUse                  // Resource in use / dependency errors
	Validation             // Input validation errors
)

// String returns the string representation of the error kind.
func (k Kind) String() string {
	switch k {
	case Auth:
		return "Auth"
	case Throttling:
		return "Throttling"
	case NotFound:
		return "NotFound"
	case InUse:
		return "InUse"
	case Validation:
		return "Validation"
	default:
		return "Unknown"
	}
}

// Classify returns the Kind of the given error.
func Classify(err error) Kind {
	if err == nil {
		return Unknown
	}
	switch {
	case IsNotFound(err):
		return NotFound
	case IsAccessDenied(err):
		return Auth
	case IsThrottling(err):
		return Throttling
	case IsResourceInUse(err):
		return InUse
	case IsValidationError(err):
		return Validation
	default:
		return Unknown
	}
}

// Common AWS error codes
const (
	ErrCodeNotFound             = "NotFound"
	ErrCodeResourceNotFound     = "ResourceNotFoundException"
	ErrCodeNoSuchEntity         = "NoSuchEntity"
	ErrCodeAccessDenied         = "AccessDenied"
	ErrCodeUnauthorized         = "UnauthorizedAccess"
	ErrCodeForbidden            = "Forbidden"
	ErrCodeThrottling           = "Throttling"
	ErrCodeTooManyRequests      = "TooManyRequestsException"
	ErrCodeRequestLimitExceeded = "RequestLimitExceeded"
	ErrCodeResourceInUse        = "ResourceInUseException"
	ErrCodeDependencyViolation  = "DependencyViolation"
	ErrCodeValidationError      = "ValidationError"
	ErrCodeInvalidParameter     = "InvalidParameterException"
)

// IsNotFound returns true if the error indicates the resource was not found.
func IsNotFound(err error) bool {
	return hasErrorCode(err,
		ErrCodeNotFound,
		ErrCodeResourceNotFound,
		ErrCodeNoSuchEntity,
		"404",
		"NoSuchBucket",
		"NoSuchKey",
		"NotFoundException",
		"ResourceNotFoundFault",
		"not found", // Match lowercase pattern from DAO Get() methods
	)
}

// IsAccessDenied returns true if the error indicates an access/permission issue.
func IsAccessDenied(err error) bool {
	return hasErrorCode(err,
		ErrCodeAccessDenied,
		ErrCodeUnauthorized,
		ErrCodeForbidden,
		"403",
		"AccessDeniedException",
		"AuthorizationError",
		"UnauthorizedException",
	)
}

// IsThrottling returns true if the error indicates rate limiting.
func IsThrottling(err error) bool {
	return hasErrorCode(err,
		ErrCodeThrottling,
		ErrCodeTooManyRequests,
		ErrCodeRequestLimitExceeded,
		"429",
		"ThrottlingException",
		"ProvisionedThroughputExceededException",
		"SlowDown",
	)
}

// IsResourceInUse returns true if the error indicates the resource is in use.
func IsResourceInUse(err error) bool {
	return hasErrorCode(err,
		ErrCodeResourceInUse,
		ErrCodeDependencyViolation,
		"ResourceInUse",
		"DeleteConflict",
		"HasAttachedResources",
	)
}

// IsValidationError returns true if the error indicates invalid input.
func IsValidationError(err error) bool {
	return hasErrorCode(err,
		ErrCodeValidationError,
		ErrCodeInvalidParameter,
		"InvalidParameterValue",
		"MalformedInput",
		"InvalidInput",
	)
}

// hasErrorCode checks if the error matches any of the given error codes.
func hasErrorCode(err error, codes ...string) bool {
	if err == nil {
		return false
	}

	// Check smithy-go APIError
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		for _, c := range codes {
			if code == c {
				return true
			}
		}
	}

	// Fallback: check error message for codes
	errStr := err.Error()
	for _, code := range codes {
		if strings.Contains(errStr, code) {
			return true
		}
	}

	return false
}

// GetErrorCode extracts the AWS error code from an error, if available.
func GetErrorCode(err error) string {
	if err == nil {
		return ""
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode()
	}

	return ""
}

// GetErrorMessage extracts the AWS error message from an error.
func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorMessage()
	}

	return err.Error()
}

func Wrap(err error, operation string, attrs ...any) error {
	if err == nil {
		return nil
	}
	log.Warn(operation, append(attrs, "error", err)...)
	return fmt.Errorf("%s: %w", operation, err)
}

func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, args...)
	log.Warn(msg, "error", err)
	return fmt.Errorf("%s: %w", msg, err)
}
