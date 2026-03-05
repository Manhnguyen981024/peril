package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType string
type AckType string

const (
	DurableType   SimpleQueueType = "durable"
	TransientType SimpleQueueType = "transient"
)

const (
	Ack         AckType = "ack"
	NackRequeue AckType = "nack_requeue"
	NackDiscard AckType = "nack_discard"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonData, err := json.Marshal(val)
	if err != nil {
		return err
	}

	errPub := ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonData,
		},
	)
	if errPub != nil {
		return errPub
	}
	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	isDurable := queueType == DurableType
	table := amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	}
	queue, err := channel.QueueDeclare(queueName, isDurable, !isDurable, !isDurable, false, table)
	if err != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	errBinding := channel.QueueBind(queue.Name, key, exchange, false, nil)
	if errBinding != nil {
		return &amqp.Channel{}, amqp.Queue{}, err
	}

	return channel, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	err := subscribe(conn, exchange, queueName, key, queueType, handler, func(data []byte) (T, error) {
		var val T
		err := json.Unmarshal(data, &val)
		if err != nil {
			log.Printf("Error when unmarshal the data: %v", err)
		}
		return val, err
	})
	if err != nil {
		log.Printf("Error when subscribe json: %v", err)
		return err
	}
	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(val)
	if err != nil {
		return err
	}

	errPub := ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/gob",
			Body:        buf.Bytes(),
		},
	)
	if errPub != nil {
		return errPub
	}
	return nil
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	err := subscribe(conn, exchange, queueName, key, queueType, handler, func(data []byte) (T, error) {
		var val T
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		err := dec.Decode(&val)
		return val, err
	})
	if err != nil {
		log.Printf("Error when subscribe gob: %v", err)
		return err
	}
	return nil
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	decodeFunc func([]byte) (T, error),
) error {
	channel, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		log.Println("Error when binding the queue")
		return err
	}

	chanDeliver, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Println("Error when consuming the queue")
		return err
	}

	// Start a goroutine
	go func() {
		for delivery := range chanDeliver {
			var data T
			data, err := decodeFunc(delivery.Body)
			if err != nil {
				log.Fatal("Error when decodeFunc the data")
			}
			acktype := handler(data)
			switch acktype {
			case Ack:
				delivery.Ack(false)
			case NackRequeue:
				delivery.Nack(false, true)
			case NackDiscard:
				delivery.Nack(false, false)
			}
		}
	}()
	return nil
}
