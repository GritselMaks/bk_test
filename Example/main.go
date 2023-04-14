package main

import (
	"fmt"
	"os"
	"os/signal"

	asendexClient "github.com/GritselMaks/bk_test"
)

var wsUrl = "wss://ascendex.com/0/api/pro/v1/stream"
var subChannel = "BTC_USDT"

func main() {
	client := asendexClient.NewClient(wsUrl)
	err := client.Connection()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}
	if err = client.SubscribeToChannel(subChannel); err != nil {
		fmt.Printf("Error Subscribe: %s\n", err.Error())
		return
	}

	ch := make(chan asendexClient.BestOrderBook)
	go func() {
		client.ReadMessagesFromChannel(ch)
	}()

	interrupt := make(chan os.Signal, 5)
	signal.Notify(interrupt, os.Interrupt)
	for {
		select {
		case book := <-ch:
			fmt.Println(book)
		case <-interrupt:
			client.Disconnect()
			close(ch)
			fmt.Println("Shutdown")
			return
		}
	}

}
