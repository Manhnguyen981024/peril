package main

import (
	"fmt"
	"time"

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
				return pubsub.NackRequeue
			}
			fmt.Println("=== handler move ack the message ===")

			return pubsub.Ack
		}
		if mvOutCome == gamelogic.MoveOutComeSafe {
			fmt.Println("=== handler move ack the message ===")
			return pubsub.Ack
		}
		fmt.Println("=== handler move nack discard the message ===")
		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState, channel *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	defer fmt.Print("> ")

	outcomeMap := map[gamelogic.WarOutcome]pubsub.AckType{
		gamelogic.WarOutcomeNotInvolved: pubsub.NackRequeue,
		gamelogic.WarOutcomeNoUnits:     pubsub.NackDiscard,
		gamelogic.WarOutcomeOpponentWon: pubsub.Ack,
		gamelogic.WarOutcomeYouWon:      pubsub.Ack,
		gamelogic.WarOutcomeDraw:        pubsub.Ack,
	}
	return func(recog gamelogic.RecognitionOfWar) pubsub.AckType {
		outcome, winner, loser := gs.HandleWar(recog)
		message := ""
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			fmt.Printf("==== War recognition received but not involved ====\n")
		case gamelogic.WarOutcomeNoUnits:
			fmt.Printf("==== War recognition received but no units to fight ====\n")
		case gamelogic.WarOutcomeOpponentWon:
			message = fmt.Sprintf("%v won a war against %v", winner, loser)
		case gamelogic.WarOutcomeYouWon:
			message = fmt.Sprintf("%v won a war against %v", winner, loser)
		case gamelogic.WarOutcomeDraw:
			message = fmt.Sprintf("%v and %v resulted in a draw", winner, loser)
		}

		fmt.Printf("We will store the log with the message is %v \n", message)
		if message != "" {
			err := publishGameLog(gs, channel, message)
			if err != nil {
				fmt.Printf("Error when publish game log: %v", err)
				return pubsub.NackRequeue
			}

			fmt.Println("the message is stored successfully")

		}
		if ackType, ok := outcomeMap[outcome]; ok {
			fmt.Printf("=== handler war %v the message ===", ackType)
			return ackType
		}
		fmt.Println("=== handler war ack the message ===")
		return pubsub.NackDiscard
	}
}

func publishGameLog(gs *gamelogic.GameState, channel *amqp.Channel, message string) error {
	gamelog := routing.GameLog{
		CurrentTime: time.Now().UTC(),
		Message:     message,
		Username:    gs.Player.Username,
	}

	err := pubsub.PublishGob(
		channel,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+gamelog.Username,
		gamelog,
	)

	return err
}
