package rules

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

func TestNewRuleResourceUsesBusQualifiedID(t *testing.T) {
	rule := types.Rule{
		Name:         aws.String("nightly"),
		EventBusName: aws.String("custom-bus"),
	}

	resource := NewRuleResource(rule)

	if resource.GetID() != "custom-bus/nightly" {
		t.Fatalf("GetID() = %q, want %q", resource.GetID(), "custom-bus/nightly")
	}
	if resource.GetName() != "nightly" {
		t.Fatalf("GetName() = %q, want %q", resource.GetName(), "nightly")
	}
}

func TestParseRuleID(t *testing.T) {
	tests := []struct {
		id       string
		name     string
		eventBus string
	}{
		{id: "nightly", name: "nightly", eventBus: ""},
		{id: "custom-bus/nightly", name: "nightly", eventBus: "custom-bus"},
		{id: "aws.partner/example.com/account/nightly", name: "nightly", eventBus: "aws.partner/example.com/account"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			name, eventBus := parseRuleID(tt.id)
			if name != tt.name || eventBus != tt.eventBus {
				t.Fatalf("parseRuleID(%q) = (%q, %q), want (%q, %q)", tt.id, name, eventBus, tt.name, tt.eventBus)
			}
		})
	}
}
