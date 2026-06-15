package main

import (
	"context"
	"fmt"
	"os"

	rtx5sdk "github.com/YoForex005/RTX5-Go_SDK"
)

func main() {
	client, err := rtx5sdk.Builder().
		BaseURL(env("RTX5_BASE_URL")).
		BrokerID(env("RTX5_BROKER_ID")).
		ManagerLogin(env("RTX5_MANAGER_LOGIN")).
		ManagerPassword(env("RTX5_MANAGER_PASSWORD")).
		Server(env("RTX5_SERVER")).
		Build()
	if err != nil {
		panic(err)
	}
	if _, err := client.Connect(context.Background()); err != nil {
		panic(err)
	}

	leverage := uint32(100)
	account, err := client.Accounts().CreateTradingAccount(context.Background(), rtx5sdk.CreateAccountRequest{
		MasterPassword: env("RTX5_TRADER_PASSWORD"),
		Group:          env("RTX5_DEFAULT_GROUP"),
		FirstName:      "SDK",
		LastName:       "Client",
		Email:          envOr("RTX5_TEST_EMAIL", "sdk-client@example.com"),
		Leverage:       &leverage,
		Currency:       "USD",
		AccountMode:    rtx5sdk.AccountModeHedging,
		Comment:        "created from rtx5-sdk-go example",
	})
	if err != nil {
		panic(err)
	}
	printJSON(account)
}

func env(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(key + " is required")
	}
	return value
}

func envOr(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func printJSON(value any) {
	out, err := rtx5sdk.MarshalValue(value)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
}
