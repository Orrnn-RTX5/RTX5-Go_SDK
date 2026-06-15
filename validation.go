package rtx5sdk

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	maxSymbolLen  = 64
	maxCommentLen = 512
	maxRangeLen   = 64
	maxGroupLen   = 128
	maxPathLen    = 256
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

func validateTimeRange(from, to string) error {
	if err := requireBoundedText("from", from, maxRangeLen); err != nil {
		return err
	}
	if err := requireBoundedText("to", to, maxRangeLen); err != nil {
		return err
	}
	return nil
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
