package main

import "testing"

func TestResolveStartupService(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantService  string
		wantResource string
		wantErr      bool
	}{
		{
			name:         "ec2 defaults to instances (not alphabetically first)",
			input:        "ec2",
			wantService:  "ec2",
			wantResource: "instances",
			wantErr:      false,
		},
		{
			name:         "rds defaults to instances",
			input:        "rds",
			wantService:  "rds",
			wantResource: "instances",
			wantErr:      false,
		},
		{
			name:         "service/resource syntax",
			input:        "ec2/volumes",
			wantService:  "ec2",
			wantResource: "volumes",
			wantErr:      false,
		},
		{
			name:         "alias resolves to service",
			input:        "cfn",
			wantService:  "cloudformation",
			wantResource: "stacks",
			wantErr:      false,
		},
		{
			name:         "alias with resource resolves",
			input:        "sg",
			wantService:  "ec2",
			wantResource: "security-groups",
			wantErr:      false,
		},
		{
			name:         "logs alias",
			input:        "logs",
			wantService:  "cloudwatch",
			wantResource: "log-groups",
			wantErr:      false,
		},
		{
			name:    "unknown service fails",
			input:   "nonexistent",
			wantErr: true,
		},
		{
			name:    "known service unknown resource fails",
			input:   "ec2/nonexistent",
			wantErr: true,
		},
		{
			name:         "alias with explicit resource override",
			input:        "cfn/resources",
			wantService:  "cloudformation",
			wantResource: "resources",
			wantErr:      false,
		},
		{
			name:    "empty string fails",
			input:   "",
			wantErr: true,
		},
		{
			name:         "trailing slash uses default resource",
			input:        "ec2/",
			wantService:  "ec2",
			wantResource: "instances",
			wantErr:      false,
		},
		{
			name:    "multiple slashes rejected",
			input:   "ec2/volumes/extra",
			wantErr: true,
		},
		{
			name:    "leading slash fails",
			input:   "/instances",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, resourceType, err := resolveStartupService(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if service != tt.wantService {
				t.Errorf("service = %q, want %q", service, tt.wantService)
			}
			if resourceType != tt.wantResource {
				t.Errorf("resourceType = %q, want %q", resourceType, tt.wantResource)
			}
		})
	}
}
