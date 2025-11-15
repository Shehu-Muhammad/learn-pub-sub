package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T),
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
				_ = d.Nack(false, false)
				continue
			}
			handler(msg)
			_ = d.Ack(false)
		}
	}()

	return nil
}
