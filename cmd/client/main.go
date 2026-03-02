package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	fmt.Println("Peril game client connected to RabbitMQ!")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}

	game := gamelogic.NewGameState(username)

	errSub := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.TransientType,
		handlerPause(game),
	)

	if errSub != nil {
		log.Fatalf("Error when subcribe a message %v", errSub)
	}

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error when creating a channel: %v", err)
	}

	errSubMove := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+username,
		routing.ArmyMovesPrefix+".*",
		pubsub.TransientType,
		handlerMove(game, channel),
	)
	if errSubMove != nil {
		log.Fatalf("Error when subcribe a exchange topic a message %v", errSubMove)
	}

	for {
		words := gamelogic.GetInput()
		cmd := words[0]
		switch cmd {
		case "spawn":
			log.Println("starting spawn")
			err := game.CommandSpawn(words)
			if err != nil {
				log.Println(err)
			}
		case "move":
			armyMove, err := game.CommandMove(words)
			if err != nil {
				log.Println(err)
			}
			pubsub.PublishJSON(
				channel,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+username,
				armyMove,
			)
		case "status":
			game.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			log.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			os.Exit(0)
		default:
			log.Println("Command not found!!")
		}
	}
}
