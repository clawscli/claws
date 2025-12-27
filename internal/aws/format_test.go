package aws

import (
	"testing"
)

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		currency string
		want     string
	}{
		{"USD explicit", 123.45, "USD", "$123.45"},
		{"empty currency defaults to USD", 99.99, "", "$99.99"},
		{"EUR currency", 50.00, "EUR", "50.00 EUR"},
		{"JPY currency", 1000.00, "JPY", "1000.00 JPY"},
		{"zero value", 0.00, "USD", "$0.00"},
		{"negative value", -10.50, "USD", "-$10.50"},
		{"large value", 1234567.89, "USD", "$1234567.89"},
		{"small decimals", 0.01, "", "$0.01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatMoney(tt.value, tt.currency); got != tt.want {
				t.Errorf("FormatMoney(%f, %q) = %q, want %q", tt.value, tt.currency, got, tt.want)
			}
		})
	}
}
