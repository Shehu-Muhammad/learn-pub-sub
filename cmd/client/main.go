package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	fmt.Println("Peril game client connected to RabbitMQ!")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not prompt for a username: %v", err)
	}

	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	ch, q, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.QueueTransient)

	if err != nil {
		log.Fatalf("declare/bind failed: %v", err)
	}
	defer ch.Close()

	fmt.Printf("queue ready: %s\n", q.Name)
	// keep the client alive for now (e.g., read from stdin or block)

	gamelogic.PrintClientHelp()
	gamestate := gamelogic.NewGameState(username)

	qName := fmt.Sprintf("pause.%s", username)
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, qName, routing.PauseKey, pubsub.QueueTransient, handlerPause(gamestate))
	if err != nil {
		log.Fatalf("subscribe failed: %v", err)
	}

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		cmd := words[0]
		switch cmd {
		case "spawn":
			fmt.Println("player is attempting to spawn a new unit")
			if err := gamestate.CommandSpawn(words); err != nil {
				fmt.Printf("spawn error: %v\n", err)
				continue
			}

		case "move":
			mv, err := gamestate.CommandMove(words)
			if err != nil {
				fmt.Printf("move error: %v\n", err)
				continue
			}
			fmt.Printf("move successful: %d unit(s) to %s\n", len(mv.Units), mv.ToLocation)

		case "status":
			gamestate.CommandStatus()

		case "help":
			gamelogic.PrintClientHelp()

		case "spam":
			fmt.Println("Spamming not allowed yet!")

		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("unknown command:", cmd)
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}
