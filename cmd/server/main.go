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
	fmt.Println("Peril game server connected to RabbitMQ!")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not open channel: %v", err)
	}
	defer ch.Close()

	err = pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.QueueDurable,
		handlerLogs(),
	)
	if err != nil {
		log.Fatalf("could not start consuming logs: %v", err)
	}

	gamelogic.PrintServerHelp()
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		cmd := words[0]
		switch cmd {
		case "pause":
			log.Println("sending pause message")
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true}); err != nil {
				log.Fatalf("could not send message to exchange: %v", err)
			}
		case "resume":
			log.Println("sending resume message")
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false}); err != nil {
				log.Fatalf("could not send message to exchange: %v", err)
			}
		case "quit":
			log.Println("exiting")
			return
		default:
			log.Println("unknown command:", cmd)
		}
	}
}

func handlerLogs() func(routing.GameLog) pubsub.Acktype {
	return func(l routing.GameLog) pubsub.Acktype {
		// 1. defer re-printing the prompt
		defer fmt.Print("> ")
		// 2. call gamelogic.WriteLog(l)
		err := gamelogic.WriteLog(l)
		// 3. return Ack or some Nack on error
		if err != nil {
			return pubsub.NackDiscard
		}
		return pubsub.Ack

	}
}
