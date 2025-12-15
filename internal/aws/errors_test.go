package aws

import (
	"errors"
	"testing"

	"github.com/aws/smithy-go"
)

// mockAPIError implements smithy.APIError for testing
type mockAPIError struct {
	code    string
	message string
}

func (e *mockAPIError) Error() string                 { return e.message }
func (e *mockAPIError) ErrorCode() string             { return e.code }
func (e *mockAPIError) ErrorMessage() string          { return e.message }
func (e *mockAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultServer }

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"ResourceNotFoundException", &mockAPIError{code: "ResourceNotFoundException"}, true},
		{"NotFound", &mockAPIError{code: "NotFound"}, true},
		{"NoSuchEntity", &mockAPIError{code: "NoSuchEntity"}, true},
		{"NoSuchBucket", &mockAPIError{code: "NoSuchBucket"}, true},
		{"other error", &mockAPIError{code: "SomeOtherError"}, false},
		{"plain error with NotFound in message", errors.New("NotFound: resource"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.expected {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAccessDenied(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"AccessDenied", &mockAPIError{code: "AccessDenied"}, true},
		{"AccessDeniedException", &mockAPIError{code: "AccessDeniedException"}, true},
		{"Forbidden", &mockAPIError{code: "Forbidden"}, true},
		{"other error", &mockAPIError{code: "SomeOtherError"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAccessDenied(tt.err); got != tt.expected {
				t.Errorf("IsAccessDenied() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsThrottling(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"Throttling", &mockAPIError{code: "Throttling"}, true},
		{"TooManyRequestsException", &mockAPIError{code: "TooManyRequestsException"}, true},
		{"RequestLimitExceeded", &mockAPIError{code: "RequestLimitExceeded"}, true},
		{"other error", &mockAPIError{code: "SomeOtherError"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsThrottling(tt.err); got != tt.expected {
				t.Errorf("IsThrottling() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsResourceInUse(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"ResourceInUseException", &mockAPIError{code: "ResourceInUseException"}, true},
		{"DependencyViolation", &mockAPIError{code: "DependencyViolation"}, true},
		{"DeleteConflict", &mockAPIError{code: "DeleteConflict"}, true},
		{"other error", &mockAPIError{code: "SomeOtherError"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsResourceInUse(tt.err); got != tt.expected {
				t.Errorf("IsResourceInUse() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, ""},
		{"API error", &mockAPIError{code: "TestCode"}, "TestCode"},
		{"plain error", errors.New("plain error"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorCode(tt.err); got != tt.expected {
				t.Errorf("GetErrorCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, ""},
		{"API error", &mockAPIError{code: "Code", message: "test message"}, "test message"},
		{"plain error", errors.New("plain error"), "plain error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorMessage(tt.err); got != tt.expected {
				t.Errorf("GetErrorMessage() = %v, want %v", got, tt.expected)
			}
		})
	}
}
