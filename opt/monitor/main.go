package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	binance "github.com/fetchtrades/binance"
	bitfinex "github.com/fetchtrades/bitfinex"
	bybit "github.com/fetchtrades/bybit"

	// bittrex "github.com/fetchtrades/bittrex"
	hitbtc "github.com/fetchtrades/hitbtc"
	kraken "github.com/fetchtrades/kraken"
	okex "github.com/fetchtrades/okex"
)

var mu sync.Mutex
var count int

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Panic in main()! Error: %s (%T) \n", err, err)
		}
	}()
	env_binance := os.Getenv("BINANCE")
	env_bitfinex := os.Getenv("BITFINEX")
	//env_bittrex := os.Getenv("BITTREX")
	env_bybit := os.Getenv("BYBIT")
	env_hitbtc := os.Getenv("HITBTC")
	env_kraken := os.Getenv("KRAKEN")
	env_okex := os.Getenv("OKEX")

	if env_binance == "true" {
		var binance binance.WebSock
		go binance.StartWS()
	}
	if env_bitfinex == "true" {
		go bitfinex.StartWS()
	}
	// if env_bittrex == "true" {
	// 	var bittrex bittrex.WebSock
	// 	go bittrex.StartWS()
	// }
	if env_bybit == "true" {
		var bybit bybit.WebSock
		go bybit.StartWS()
	}
	if env_hitbtc == "true" {
		var hitbtc hitbtc.WebSock
		go hitbtc.StartWS()
	}
	if env_kraken == "true" {
		var kraken kraken.WebSock
		go kraken.StartWS()
	}
	if env_okex == "true" {
		var okex okex.WebSock
		go okex.StartWS()
	}

	for {
		time.Sleep(time.Second)
	}
}
