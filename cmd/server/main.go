package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		panic("Failed to connect to RabbitMQ: " + err.Error())
	}
	defer conn.Close()
	channel, queue, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", 0)
	if err != nil {
		panic("Failed to declare and bind queue: " + err.Error())
	}
	_ = queue
	err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
	if err != nil {
		panic("Failed to publish message: " + err.Error())
	}
	fmt.Println("Connected to RabbitMQ")
	gamelogic.PrintServerHelp()
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		if input[0] == "pause" {
			log.Printf("Pausing game...")
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
			if err != nil {
				panic("Failed to publish message: " + err.Error())
			}
		}
		if input[0] == "resume" {
			log.Printf("Resuming game...")
			err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
			if err != nil {
				panic("Failed to publish message: " + err.Error())
			}
		}
		if input[0] == "quit" {
			log.Printf("Quitting game...")
			break
		}
		log.Printf("Unknown command: %s", input[0])
	}
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
