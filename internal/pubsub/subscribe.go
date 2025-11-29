package pubsub

import (
	"bytes"
	"encoding/gob"
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
	queueType SimpleQueueType,
	handler func(T) Acktype,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshalJSON[T])
}

func unmarshalJSON[T any](data []byte) (T, error) {
	var target T
	err := json.Unmarshal(data, &target)
	return target, err
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) Acktype,
) error {
	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshalGob[T])
}

func unmarshalGob[T any](data []byte) (T, error) {
	var target T
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&target); err != nil {
		var zero T
		return zero, err
	}
	return target, nil
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) Acktype,
	unmarshaller func([]byte) (T, error),
) error {
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("declare/bind: %w", err)
	}

	deliveries, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	go func() {
		defer ch.Close()
		for d := range deliveries {
			msg, err := unmarshaller(d.Body)
			if err != nil {
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
