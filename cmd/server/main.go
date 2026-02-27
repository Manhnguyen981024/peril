package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatal(err)
	}

	channel, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	// errPublish := pubsub.PublishJSON(
	// 	channel,
	// 	routing.ExchangePerilDirect,
	// 	routing.PauseKey,
	// 	routing.PlayingState{IsPaused: true},
	// )

	// if errPublish != nil {
	// 	log.Fatal(errPublish)
	// }

	defer conn.Close()
	fmt.Println("The connection was successful.")

	gamelogic.PrintServerHelp()

	_, _, errDeclare := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.DurableType,
	)
	if errDeclare != nil {
		log.Fatalf("Error to bind the exchange: %v", errDeclare)
	}
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		cmd := strings.ToLower(words[0])
		isPaused := false

		switch cmd {
		case "pause":
			isPaused = true
			log.Println("you're sending a pause message")
		case "resume":
			isPaused = false
			log.Println("you're sending a resume message")
		case "quit":
			log.Println("you're exiting")
		default:
			log.Println("you don't understand the command.")
		}
		if cmd == "quit" {
			break
		}
		errPublish := pubsub.PublishJSON(
			channel,
			routing.ExchangePerilDirect,
			routing.PauseKey,
			routing.PlayingState{IsPaused: isPaused},
		)
		if errPublish != nil {
			log.Fatal(errPublish)
		}
	}
}
