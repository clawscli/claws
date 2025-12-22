package certificates

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
)

func TestNewCertificateResource(t *testing.T) {
	notAfter := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	cert := &types.CertificateDetail{
		CertificateArn: aws.String("arn:aws:acm:us-east-1:123456789012:certificate/abc123"),
		DomainName:     aws.String("example.com"),
		Status:         types.CertificateStatusIssued,
		Type:           types.CertificateTypeAmazonIssued,
		KeyAlgorithm:   types.KeyAlgorithmRsa2048,
		NotAfter:       &notAfter,
		InUseBy:        []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/123"},
	}

	resource := NewCertificateResource(cert)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"GetID", resource.GetID(), "arn:aws:acm:us-east-1:123456789012:certificate/abc123"},
		{"GetName", resource.GetName(), "example.com"},
		{"GetARN", resource.GetARN(), "arn:aws:acm:us-east-1:123456789012:certificate/abc123"},
		{"DomainName", resource.DomainName(), "example.com"},
		{"Status", resource.Status(), "ISSUED"},
		{"Type", resource.Type(), "AMAZON_ISSUED"},
		{"KeyAlgorithm", resource.KeyAlgorithm(), "RSA_2048"},
		{"NotAfter", resource.NotAfter(), "2025-12-31"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Test InUseBy
	inUseBy := resource.InUseBy()
	if len(inUseBy) != 1 {
		t.Errorf("InUseBy() len = %d, want 1", len(inUseBy))
	}

	// Test IsInUse
	isInUse := resource.IsInUse()
	if isInUse == nil || !*isInUse {
		t.Error("IsInUse() should be true")
	}
}

func TestNewCertificateResourceFromSummary(t *testing.T) {
	notAfter := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	inUse := true
	summary := types.CertificateSummary{
		CertificateArn:     aws.String("arn:aws:acm:us-east-1:123456789012:certificate/abc123"),
		DomainName:         aws.String("example.com"),
		Status:             types.CertificateStatusIssued,
		Type:               types.CertificateTypeAmazonIssued,
		KeyAlgorithm:       types.KeyAlgorithmRsa2048,
		NotAfter:           &notAfter,
		InUse:              &inUse,
		RenewalEligibility: types.RenewalEligibilityEligible,
	}

	resource := NewCertificateResourceFromSummary(summary)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"GetID", resource.GetID(), "arn:aws:acm:us-east-1:123456789012:certificate/abc123"},
		{"GetName", resource.GetName(), "example.com"},
		{"DomainName", resource.DomainName(), "example.com"},
		{"Status", resource.Status(), "ISSUED"},
		{"Type", resource.Type(), "AMAZON_ISSUED"},
		{"KeyAlgorithm", resource.KeyAlgorithm(), "RSA_2048"},
		{"NotAfter", resource.NotAfter(), "2025-12-31"},
		{"RenewalEligibility", resource.RenewalEligibility(), "ELIGIBLE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}

	// Summary doesn't have InUseBy list, only InUse bool
	inUseBy := resource.InUseBy()
	if inUseBy != nil {
		t.Errorf("InUseBy() should be nil for Summary, got %v", inUseBy)
	}

	// But IsInUse should work
	isInUse := resource.IsInUse()
	if isInUse == nil || !*isInUse {
		t.Error("IsInUse() should be true")
	}
}

func TestCertificateResource_IsInUse(t *testing.T) {
	tests := []struct {
		name     string
		resource *CertificateResource
		expected *bool
	}{
		{
			name: "Item with InUseBy",
			resource: &CertificateResource{
				Item: &types.CertificateDetail{
					InUseBy: []string{"arn:aws:elasticloadbalancing:..."},
				},
			},
			expected: aws.Bool(true),
		},
		{
			name: "Item without InUseBy",
			resource: &CertificateResource{
				Item: &types.CertificateDetail{
					InUseBy: []string{},
				},
			},
			expected: aws.Bool(false),
		},
		{
			name: "Summary InUse true",
			resource: &CertificateResource{
				Summary: &types.CertificateSummary{
					InUse: aws.Bool(true),
				},
			},
			expected: aws.Bool(true),
		},
		{
			name: "Summary InUse false",
			resource: &CertificateResource{
				Summary: &types.CertificateSummary{
					InUse: aws.Bool(false),
				},
			},
			expected: aws.Bool(false),
		},
		{
			name:     "Neither Item nor Summary",
			resource: &CertificateResource{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.resource.IsInUse()
			if tt.expected == nil {
				if got != nil {
					t.Errorf("IsInUse() = %v, want nil", *got)
				}
			} else if got == nil {
				t.Errorf("IsInUse() = nil, want %v", *tt.expected)
			} else if *got != *tt.expected {
				t.Errorf("IsInUse() = %v, want %v", *got, *tt.expected)
			}
		})
	}
}

func TestCertificateResource_NilFields(t *testing.T) {
	// Test with completely nil Item
	resource := &CertificateResource{}

	if resource.DomainName() != "" {
		t.Errorf("DomainName() = %q, want empty", resource.DomainName())
	}
	if resource.Status() != "" {
		t.Errorf("Status() = %q, want empty", resource.Status())
	}
	if resource.NotAfter() != "" {
		t.Errorf("NotAfter() = %q, want empty", resource.NotAfter())
	}
	if resource.InUseBy() != nil {
		t.Errorf("InUseBy() = %v, want nil", resource.InUseBy())
	}
}
