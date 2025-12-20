package aws

import (
	"testing"
)

func TestCommonRegions(t *testing.T) {
	if len(CommonRegions) == 0 {
		t.Error("CommonRegions should not be empty")
	}

	// Check some expected regions are present
	expectedRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-northeast-1"}
	for _, expected := range expectedRegions {
		found := false
		for _, region := range CommonRegions {
			if region == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("CommonRegions should contain %q", expected)
		}
	}
}
