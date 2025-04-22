package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
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

type SimpleQueueType int

const (
	DurableQueue SimpleQueueType = iota
	TransientQueue
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

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(val)
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
			ContentType:     "application/gob",
			ContentEncoding: "binary",
			Body:            network.Bytes(),
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
	simpleQueueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	durable, autoDelete, exclusive := true, false, false
	switch simpleQueueType {
	case DurableQueue: // durable
		durable = true
		autoDelete = false
		exclusive = false
	case TransientQueue: // transient
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

//func SubscribeJSON[T any](
//	conn *amqp.Connection,
//	exchange,
//	queueName,
//	key string,
//	simpleQueueType SimpleQueueType, // an enum to represent "durable" or "transient"
//	handler func(T) AckType,
//) error {
//	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
//	if err != nil {
//		return err
//	}
//	retrnch, err := ch.Consume(queueName, "", false, false, false, false, nil)
//	go func() {
//		for i := range retrnch {
//			var msg T
//			json.Unmarshal(i.Body, &msg)
//			handlerreturn := handler(msg)
//			switch handlerreturn {
//			case Ack:
//				log.Printf("Received Ack")
//				err = i.Ack(false)
//				if err != nil {
//					return
//				}
//			case NackRequeue:
//				log.Printf("Received NackRequeue")
//				i.Nack(false, true)
//				if err != nil {
//					return
//				}
//			case NackDiscard:
//				log.Printf("Received NackDiscard")
//				i.Nack(false, false)
//				if err != nil {
//					return
//				}
//			}
//		}
//	}()
//	return nil
//}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}
	retrnch, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for i := range retrnch {
			val, err := unmarshaller(i.Body)
			handlerreturn := handler(val)
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

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, simpleQueueType, handler, unmarshalGob[T])
}

func unmarshalGob[T any](data []byte) (T, error) {
	var val T
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&val)
	if err != nil {
		return val, err
	}
	return val, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	return subscribe(conn, exchange, queueName, key, simpleQueueType, handler, unmarshalJSON[T])
}

func unmarshalJSON[T any](data []byte) (T, error) {
	var val T
	err := json.Unmarshal(data, &val)
	if err != nil {
		return val, err
	}
	return val, nil
}
