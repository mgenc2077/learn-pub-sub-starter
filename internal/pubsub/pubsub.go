package pubsub

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	valjson, err := json.Marshal(val)
	if err != nil {
		return err
	}
	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        valjson,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	durable, autoDelete, exclusive := true, false, false
	switch simpleQueueType {
	case 0: // durable
		durable = true
		autoDelete = false
		exclusive = false
	case 1: // transient
		durable = false
		autoDelete = true
		exclusive = true
	}
	queue, err := ch.QueueDeclare(
		queueName,
		durable,
		autoDelete,
		exclusive,
		false,
		amqp.Table{"x-dead-letter-exchange": "peril_dlx"},
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}
	retrnch, err := ch.Consume(queueName, "", false, false, false, false, nil)
	go func() {
		for i := range retrnch {
			var msg T
			json.Unmarshal(i.Body, &msg)
			handlerreturn := handler(msg)
			switch handlerreturn {
			case Ack:
				log.Printf("Received Ack")
				err = i.Ack(false)
				if err != nil {
					return
				}
			case NackRequeue:
				log.Printf("Received NackRequeue")
				i.Nack(false, true)
				if err != nil {
					return
				}
			case NackDiscard:
				log.Printf("Received NackDiscard")
				i.Nack(false, false)
				if err != nil {
					return
				}
			}
		}
	}()
	return nil
}
