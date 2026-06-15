package rtx5sdk

import (
	"encoding/json"
	"fmt"
	"math"
)

type Value = any

type ManagerLoginRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Server   string `json:"server"`
	BrokerID string `json:"broker_id,omitempty"`
}

type AccountMode string

const (
	AccountModeHedging AccountMode = "hedging"
	AccountModeNetting AccountMode = "netting"
)

type CreateAccountRequest struct {
	MasterPassword            string
	Group                     string
	InvestorPassword          string
	Leverage                  *uint32
	FirstName                 string
	LastName                  string
	Email                     string
	Phone                     string
	Country                   string
	City                      string
	Address                   string
	ZipCode                   string
	Company                   string
	Comment                   string
	Currency                  string
	Credit                    *float64
	Status                    string
	AccountMode               AccountMode
	MarginCallLevel           *float64
	StopOutLevel              *float64
	CommissionRate            *float64
	SwapFree                  *bool
	NegativeBalanceProtection *bool
	MQID                      string
}

type CreateAccountAndDepositRequest struct {
	Account CreateAccountRequest
	Amount  float64
}

type OrderSendRequest struct {
	Login      int64
	Symbol     string
	Operation  string
	Lots       float64
	Price      *float64
	StopLoss   *float64
	TakeProfit *float64
}

type OrderCloseRequest struct {
	Ticket int64
	Lots   *float64
}

type OrderModifyRequest struct {
	Ticket int64
	// Price is not supported by the current manager /OrderModify endpoint.
	// Set only StopLoss and TakeProfit. Use 0 explicitly to clear either value.
	Price      *float64
	StopLoss   *float64
	TakeProfit *float64
}

type BalanceAction string

const (
	BalanceActionDeposit    BalanceAction = "deposit"
	BalanceActionWithdraw   BalanceAction = "withdraw"
	BalanceActionCredit     BalanceAction = "credit"
	BalanceActionCorrection BalanceAction = "correction"
	BalanceActionBonus      BalanceAction = "bonus"
)

type BalanceAdjustmentRequest struct {
	Login   int64
	Amount  float64
	Action  BalanceAction
	Comment string
}

type TimeRangeRequest struct {
	From string
	To   string
}

type LoginTimeRangeRequest struct {
	Login int64
	From  string
	To    string
}

type GroupTimeRangeRequest struct {
	Group string
	From  string
	To    string
}

const MaxGroupLeverage uint32 = 10000

type GroupConfig struct {
	Name            string         `json:"name"`
	Currency        string         `json:"currency,omitempty"`
	Leverage        *uint32        `json:"leverage,omitempty"`
	Spread          *float64       `json:"spreadMarkup,omitempty"`
	ContractSize    *float64       `json:"contractSize,omitempty"`
	MarginCallLevel *float64       `json:"marginCallLevel,omitempty"`
	StopOutLevel    *float64       `json:"stopOutLevel,omitempty"`
	Extra           map[string]any `json:"-"`
}

func NewGroupConfig(name string) GroupConfig {
	return GroupConfig{Name: name}
}

func (g GroupConfig) Validate() error {
	if trimEmpty(g.Name) {
		return InvalidInputError{Message: "group name is required"}
	}
	if g.Leverage != nil && (*g.Leverage == 0 || *g.Leverage > MaxGroupLeverage) {
		return InvalidInputError{Message: fmt.Sprintf("leverage must be between 1 and %d, got %d", MaxGroupLeverage, *g.Leverage)}
	}
	if err := checkFiniteMin("spread", g.Spread, 0); err != nil {
		return err
	}
	if err := checkPositiveOptional("contractSize", g.ContractSize); err != nil {
		return err
	}
	if err := checkFiniteMin("marginCallLevel", g.MarginCallLevel, 0); err != nil {
		return err
	}
	if err := checkFiniteMin("stopOutLevel", g.StopOutLevel, 0); err != nil {
		return err
	}
	if g.MarginCallLevel != nil && g.StopOutLevel != nil && *g.StopOutLevel > *g.MarginCallLevel {
		return InvalidInputError{Message: fmt.Sprintf("stopOutLevel (%v) must not exceed marginCallLevel (%v)", *g.StopOutLevel, *g.MarginCallLevel)}
	}
	return nil
}

func (g GroupConfig) body() (map[string]any, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	body := map[string]any{}
	for key, value := range g.Extra {
		body[key] = value
	}
	body["name"] = g.Name
	if g.Currency != "" {
		body["currency"] = g.Currency
	}
	if g.Leverage != nil {
		body["leverage"] = *g.Leverage
	}
	insertFloat(body, "spreadMarkup", g.Spread)
	insertFloat(body, "contractSize", g.ContractSize)
	insertFloat(body, "marginCallLevel", g.MarginCallLevel)
	insertFloat(body, "stopOutLevel", g.StopOutLevel)
	return body, nil
}

func Ptr[T any](v T) *T {
	return &v
}

func MarshalValue(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func checkFinite(field string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return InvalidInputError{Message: fmt.Sprintf("%s must be a finite number, got %v", field, value)}
	}
	return nil
}

func requirePositiveAmount(field string, value float64) error {
	if err := checkFinite(field, value); err != nil {
		return err
	}
	if value <= 0 {
		return InvalidInputError{Message: fmt.Sprintf("%s must be greater than zero, got %v", field, value)}
	}
	return nil
}

func requireOptionalFinite(field string, value *float64) error {
	if value == nil {
		return nil
	}
	return checkFinite(field, *value)
}

func checkFiniteMin(field string, value *float64, min float64) error {
	if value == nil {
		return nil
	}
	if err := checkFinite(field, *value); err != nil {
		return err
	}
	if *value < min {
		return InvalidInputError{Message: fmt.Sprintf("%s must be >= %v, got %v", field, min, *value)}
	}
	return nil
}

func checkPositiveOptional(field string, value *float64) error {
	if value == nil {
		return nil
	}
	if err := checkFinite(field, *value); err != nil {
		return err
	}
	if *value <= 0 {
		return InvalidInputError{Message: fmt.Sprintf("%s must be a finite value > 0, got %v", field, *value)}
	}
	return nil
}

func insertFloat(body map[string]any, key string, value *float64) {
	if value != nil {
		body[key] = *value
	}
}
