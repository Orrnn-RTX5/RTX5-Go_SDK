package rtx5sdk

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	maxSymbolLen       = 64
	maxCommentLen      = 512
	maxRangeLen        = 64
	maxGroupLen        = 128
	maxPathLen         = 256
	managerTimeLayout  = "2006-01-02T15:04:05"
	compactTimeLayout  = "2006-01-02 15:04:05"
	managerDateLayout  = "2006-01-02"
	epochMillisDivisor = int64(1000)
	epochMicrosDivisor = int64(1000 * 1000)
	epochNanosDivisor  = int64(1000 * 1000 * 1000)
)

var allowedOrderOperations = map[string]struct{}{
	"buy":        {},
	"sell":       {},
	"buy_limit":  {},
	"sell_limit": {},
	"buy_stop":   {},
	"sell_stop":  {},
}

var allowedBalanceActions = map[BalanceAction]struct{}{
	BalanceActionDeposit:    {},
	BalanceActionWithdraw:   {},
	BalanceActionCredit:     {},
	BalanceActionCorrection: {},
	BalanceActionBonus:      {},
}

func requirePositiveInt64(field string, value int64) error {
	if value <= 0 {
		return InvalidInputError{Message: fmt.Sprintf("%s must be greater than zero, got %d", field, value)}
	}
	return nil
}

func requireBoundedText(field, value string, max int) error {
	if strings.TrimSpace(value) == "" {
		return InvalidInputError{Message: field + " is required"}
	}
	if len(value) > max {
		return InvalidInputError{Message: fmt.Sprintf("%s must be at most %d bytes", field, max)}
	}
	return nil
}

func validateOptionalComment(comment string) error {
	if len(comment) > maxCommentLen {
		return InvalidInputError{Message: fmt.Sprintf("comment must be at most %d bytes", maxCommentLen)}
	}
	return nil
}

func validateSymbol(symbol string) error {
	if err := requireBoundedText("symbol", symbol, maxSymbolLen); err != nil {
		return err
	}
	for _, r := range symbol {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case strings.ContainsRune("._-/+#:", r):
		default:
			return InvalidInputError{Message: "symbol contains invalid characters"}
		}
	}
	return nil
}

func validateOrderOperation(operation string) error {
	if err := requireBoundedText("operation", operation, 32); err != nil {
		return err
	}
	if _, ok := allowedOrderOperations[strings.ToLower(operation)]; !ok {
		return InvalidInputError{Message: "operation is not supported by the typed SDK"}
	}
	return nil
}

func validateBalanceAction(action BalanceAction) error {
	if _, ok := allowedBalanceActions[action]; !ok {
		return InvalidInputError{Message: "balance action is not supported by the typed SDK"}
	}
	return nil
}

func normalizeTimeRange(from, to string) (string, string, error) {
	normalizedFrom, fromTime, err := normalizeManagerTime("from", from)
	if err != nil {
		return "", "", err
	}
	normalizedTo, toTime, err := normalizeManagerTime("to", to)
	if err != nil {
		return "", "", err
	}
	if toTime.Before(fromTime) {
		return "", "", InvalidInputError{Message: "to must not be before from"}
	}
	return normalizedFrom, normalizedTo, nil
}

func normalizeManagerTime(field, value string) (string, time.Time, error) {
	if err := requireBoundedText(field, value, maxRangeLen); err != nil {
		return "", time.Time{}, err
	}

	raw := strings.TrimSpace(value)
	if t, ok, err := parseEpochTime(raw); ok || err != nil {
		if err != nil {
			return "", time.Time{}, err
		}
		utc := t.UTC()
		return utc.Format(managerTimeLayout), utc, nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		managerTimeLayout,
		compactTimeLayout,
		managerDateLayout,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			utc := t.UTC()
			return utc.Format(managerTimeLayout), utc, nil
		}
	}

	return "", time.Time{}, InvalidInputError{Message: fmt.Sprintf("%s must be RFC3339, YYYY-MM-DDTHH:MM:SS, YYYY-MM-DD, or a Unix epoch timestamp", field)}
}

func parseEpochTime(raw string) (time.Time, bool, error) {
	if raw == "" {
		return time.Time{}, false, nil
	}
	for _, r := range raw {
		if r < '0' || r > '9' {
			return time.Time{}, false, nil
		}
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, true, InvalidInputError{Message: "timestamp is out of range"}
	}

	switch {
	case len(raw) <= 10:
		return time.Unix(value, 0), true, nil
	case len(raw) <= 13:
		return time.Unix(value/epochMillisDivisor, (value%epochMillisDivisor)*int64(time.Millisecond)), true, nil
	case len(raw) <= 16:
		return time.Unix(value/epochMicrosDivisor, (value%epochMicrosDivisor)*int64(time.Microsecond)), true, nil
	case len(raw) <= 19:
		return time.Unix(value/epochNanosDivisor, value%epochNanosDivisor), true, nil
	default:
		return time.Time{}, true, InvalidInputError{Message: "timestamp is out of range"}
	}
}

func validateGroupName(group string) error {
	return requireBoundedText("group", group, maxGroupLen)
}

func validateRawPath(path string) error {
	if err := requireBoundedText("path", path, maxPathLen); err != nil {
		return err
	}
	if !strings.HasPrefix(path, "/") {
		return InvalidInputError{Message: "path must start with /"}
	}
	if strings.ContainsAny(path, " \t\r\n") {
		return InvalidInputError{Message: "path must not contain whitespace"}
	}
	if strings.Contains(path, "..") {
		return InvalidInputError{Message: "path must not contain traversal segments"}
	}
	if _, err := url.ParseRequestURI(path); err != nil {
		return InvalidInputError{Message: "path is not a valid request URI"}
	}
	return nil
}
