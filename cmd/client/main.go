package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(routing.PlayingState{IsPaused: true})
	}
}

func main() {
	fmt.Println("Starting Peril client...")
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		panic("Failed to connect to RabbitMQ: " + err.Error())
	}
	defer conn.Close()
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		panic("Failed to get username: " + err.Error())
	}
	ch, queue, err := pubsub.DeclareAndBind(conn, "peril_direct", routing.PauseKey+"."+username, routing.PauseKey, 1)
	if err != nil {
		panic("Failed to declare and bind queue: " + err.Error())
	}
	_, _ = queue, ch
	gstate := gamelogic.NewGameState(username)
	pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, routing.PauseKey+"."+username, routing.PauseKey, 1, handlerPause(gstate))
	pubsub.SubscribeJSON(conn, "peril_topic", routing.ArmyMovesPrefix+"."+username, routing.ArmyMovesPrefix+".*", 0,
		func(m gamelogic.ArmyMove) {
			_ = gstate.HandleMove(m)
			fmt.Print("> ")
		})

outerloop:
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "spawn":
			log.Printf("Spawning unit...")
			err = gstate.CommandSpawn(input)
			if err != nil {
				log.Printf("Failed to spawn unit: " + err.Error())
			}
		case "move":
			log.Printf("moving unit...")
			move, err := gstate.CommandMove(input)
			if err != nil {
				log.Printf("Failed to move unit: " + err.Error())
			}
			pubsub.PublishJSON(ch, "peril_topic", routing.ArmyMovesPrefix+"."+username, move)
			log.Printf("Move was published succesfuly")
		case "status":
			gstate.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			log.Printf("Haven't implamented yet...")
		case "quit":
			gamelogic.PrintQuit()
			break outerloop
		default:
			log.Printf("Unknown command: %s", input[0])
		}
	}
	//signalChan := make(chan os.Signal, 1)
	//signal.Notify(signalChan, os.Interrupt)
	//<-signalChan
}
