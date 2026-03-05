package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

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
	channel, err := conn.Channel()

	if err != nil {
		log.Fatalf("Error when creating a channel: %v", err)
	}

	// subscribe to pause actions
	errSubPause := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.TransientType,
		handlerPause(game),
	)

	if errSubPause != nil {
		log.Fatalf("Error when subcribe a message %v", errSubPause)
	}

	// subscribe to war actions
	errSubWar := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		"war",
		routing.WarRecognitionsPrefix+".*",
		pubsub.DurableType,
		handlerWar(game, channel),
	)

	if errSubWar != nil {
		log.Fatalf("Error when subcribe a message %v", errSubWar)
	}

	// subscribe to move actions
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
			num, err := strconv.Atoi(words[1])
			if err != nil {
				log.Fatal(err)
			}
			for i := 0; i < num; i++ {
				maliciousLog := gamelogic.GetMaliciousLog()
				err := publishGameLog(game, channel, maliciousLog)
				if err != nil {
					log.Printf("Error when publish malicious log: %v", err)
				}
			}
		case "quit":
			gamelogic.PrintQuit()
			os.Exit(0)
		default:
			log.Println("Command not found!!")
		}
	}
}
