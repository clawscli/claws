package errors

import (
	"errors"
	"testing"

	"github.com/aws/smithy-go"
)

type mockAPIError struct {
	code    string
	message string
}

func (e *mockAPIError) Error() string                 { return e.message }
func (e *mockAPIError) ErrorCode() string             { return e.code }
func (e *mockAPIError) ErrorMessage() string          { return e.message }
func (e *mockAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultUnknown }

func TestKindString(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{Unknown, "Unknown"},
		{Auth, "Auth"},
		{Throttling, "Throttling"},
		{NotFound, "NotFound"},
		{InUse, "InUse"},
		{Validation, "Validation"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{"nil error", nil, Unknown},
		{"not found", &mockAPIError{code: "ResourceNotFoundException"}, NotFound},
		{"access denied", &mockAPIError{code: "AccessDenied"}, Auth},
		{"throttling", &mockAPIError{code: "Throttling"}, Throttling},
		{"in use", &mockAPIError{code: "ResourceInUseException"}, InUse},
		{"validation", &mockAPIError{code: "ValidationError"}, Validation},
		{"unknown code", &mockAPIError{code: "SomeOtherError"}, Unknown},
		{"plain error", errors.New("some error"), Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.err); got != tt.want {
				t.Errorf("Classify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"ResourceNotFoundException", true},
		{"NotFound", true},
		{"NoSuchEntity", true},
		{"NoSuchBucket", true},
		{"AccessDenied", false},
	}
	for _, tt := range tests {
		if got := IsNotFound(&mockAPIError{code: tt.code}); got != tt.want {
			t.Errorf("IsNotFound(%q) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestIsAccessDenied(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"AccessDenied", true},
		{"Forbidden", true},
		{"AccessDeniedException", true},
		{"NotFound", false},
	}
	for _, tt := range tests {
		if got := IsAccessDenied(&mockAPIError{code: tt.code}); got != tt.want {
			t.Errorf("IsAccessDenied(%q) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestIsThrottling(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"Throttling", true},
		{"TooManyRequestsException", true},
		{"SlowDown", true},
		{"NotFound", false},
	}
	for _, tt := range tests {
		if got := IsThrottling(&mockAPIError{code: tt.code}); got != tt.want {
			t.Errorf("IsThrottling(%q) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestGetErrorCode(t *testing.T) {
	if got := GetErrorCode(nil); got != "" {
		t.Errorf("GetErrorCode(nil) = %q, want empty", got)
	}
	if got := GetErrorCode(&mockAPIError{code: "TestCode"}); got != "TestCode" {
		t.Errorf("GetErrorCode() = %q, want TestCode", got)
	}
	if got := GetErrorCode(errors.New("plain")); got != "" {
		t.Errorf("GetErrorCode(plain) = %q, want empty", got)
	}
}

func TestGetErrorMessage(t *testing.T) {
	if got := GetErrorMessage(nil); got != "" {
		t.Errorf("GetErrorMessage(nil) = %q, want empty", got)
	}
	if got := GetErrorMessage(&mockAPIError{message: "test msg"}); got != "test msg" {
		t.Errorf("GetErrorMessage() = %q, want 'test msg'", got)
	}
	if got := GetErrorMessage(errors.New("plain error")); got != "plain error" {
		t.Errorf("GetErrorMessage(plain) = %q, want 'plain error'", got)
	}
}

func TestWrap(t *testing.T) {
	if got := Wrap(nil, "test"); got != nil {
		t.Errorf("Wrap(nil) = %v, want nil", got)
	}
	err := errors.New("original")
	wrapped := Wrap(err, "operation")
	if wrapped == nil {
		t.Fatal("Wrap() returned nil for non-nil error")
	}
	if got := wrapped.Error(); got != "operation: original" {
		t.Errorf("Wrap() = %q, want 'operation: original'", got)
	}
	if !errors.Is(wrapped, err) {
		t.Error("wrapped error should unwrap to original")
	}
}

func TestWrapf(t *testing.T) {
	if got := Wrapf(nil, "test %s", "arg"); got != nil {
		t.Errorf("Wrapf(nil) = %v, want nil", got)
	}
	err := errors.New("original")
	wrapped := Wrapf(err, "get item %s", "myid")
	if wrapped == nil {
		t.Fatal("Wrapf() returned nil for non-nil error")
	}
	if got := wrapped.Error(); got != "get item myid: original" {
		t.Errorf("Wrapf() = %q, want 'get item myid: original'", got)
	}
	if !errors.Is(wrapped, err) {
		t.Error("wrapped error should unwrap to original")
	}
}
