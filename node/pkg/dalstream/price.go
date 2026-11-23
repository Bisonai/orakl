package dalstream

import (
	"fmt"
	"strings"
)

// formatPrice scales a raw integer value by decimals and returns it as a
// decimal string without trailing zeros, e.g. ("6141018000000", 8) ->
// "61410.18".
func formatPrice(raw string, decimals int) (string, error) {
	if decimals < 0 || decimals > 77 {
		return "", fmt.Errorf("invalid decimals %d", decimals)
	}
	neg := strings.HasPrefix(raw, "-")
	digits := strings.TrimPrefix(raw, "-")
	if digits == "" {
		return "", fmt.Errorf("invalid value %q", raw)
	}
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return "", fmt.Errorf("invalid value %q", raw)
		}
	}
	if len(digits) <= decimals {
		digits = strings.Repeat("0", decimals-len(digits)+1) + digits
	}
	intPart := strings.TrimLeft(digits[:len(digits)-decimals], "0")
	if intPart == "" {
		intPart = "0"
	}
	frac := strings.TrimRight(digits[len(digits)-decimals:], "0")
	out := intPart
	if frac != "" {
		out += "." + frac
	}
	if neg && out != "0" {
		out = "-" + out
	}
	return out, nil
}
