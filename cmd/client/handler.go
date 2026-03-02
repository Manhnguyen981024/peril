package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	defer fmt.Print("> ")
	return func(ps routing.PlayingState) pubsub.AckType {
		gs.HandlePause(ps)
		fmt.Println("=== handler pause ack the message ===")
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, channel *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	defer fmt.Print("> ")
	return func(am gamelogic.ArmyMove) pubsub.AckType {
		mvOutCome := gs.HandleMove(am)
		if mvOutCome == gamelogic.MoveOutcomeMakeWar {
			err := pubsub.PublishJSON(
				channel,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.Player.Username,
				gamelogic.RecognitionOfWar{
					Attacker: am.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Printf("Error when publish recognition of war: %v", err)
			}
			fmt.Println("=== handler move ack the message ===")

			return pubsub.NackRequeue
		}
		if mvOutCome == gamelogic.MoveOutComeSafe {
			fmt.Println("=== handler move ack the message ===")
			return pubsub.Ack
		}
		fmt.Println("=== handler move nack discard the message ===")
		return pubsub.NackDiscard
	}
}
