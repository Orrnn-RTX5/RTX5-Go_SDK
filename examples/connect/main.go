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

	session, err := client.Connect(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Printf("connected token_len=%d\n", len(session.Token))

	groups, err := client.Groups().List(context.Background())
	if err != nil {
		panic(err)
	}
	printJSON(groups)
}

func env(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(key + " is required")
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
