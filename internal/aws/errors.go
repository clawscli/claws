package aws

import (
	"errors"
	"strings"

	"github.com/aws/smithy-go"
)

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
