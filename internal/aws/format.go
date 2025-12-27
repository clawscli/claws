package aws

import "fmt"

// FormatMoney formats a monetary value with its currency symbol.
// If currency is empty or "USD", uses "$" prefix. Otherwise appends the currency code.
func FormatMoney(value float64, currency string) string {
	if currency == "" || currency == "USD" {
		if value < 0 {
			return fmt.Sprintf("-$%.2f", -value)
		}
		return fmt.Sprintf("$%.2f", value)
	}
	return fmt.Sprintf("%.2f %s", value, currency)
}
