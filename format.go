package rtx5sdk

import (
	"strconv"
	"strings"
)

func formatAmount(value float64) string {
	s := strconv.FormatFloat(value, 'f', 8, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if s == "" || s == "-" || s == "-0" {
		return "0"
	}
	return s
}
