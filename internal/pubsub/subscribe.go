package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Acktype int

const (
	Ack Acktype = iota
	NackDiscard
	NackRequeue
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("declare/bind: %w", err)
	}

	deliveries, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	go func() {
		for d := range deliveries {
			var msg T
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				fmt.Println("could not unmarshal message:", err)
				_ = d.Nack(false, false)
				fmt.Println("NackDiscard (unmarshal error)")
				continue
			}

			ackType := handler(msg)

			switch ackType {
			case Ack:
				_ = d.Ack(false)
				fmt.Println("Ack")
			case NackRequeue:
				_ = d.Nack(false, true)
				fmt.Println("NackRequeue")
			case NackDiscard:
				_ = d.Nack(false, false)
				fmt.Println("NackDiscard")
			}
		}
	}()

	return nil
}
