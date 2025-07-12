package main

import (
	"context"
	"os"
	"os/signal"

	"go.uber.org/zap"

	"github.com/dafraer/create-video-note-bot/bot"
)

func main() {
	if len(os.Args) < 2 {
		panic("telegram bot token must be passed as arguments")
	}
	token := os.Args[1]

	//Declare context that is marked Done when os.Interrupt is called
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	//Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	sugar := logger.Sugar()

	//Create bot
	myBot, err := bot.New(token, sugar)
	if err != nil {
		panic(err)
	}

	//If webhook flag specified run bot using webhook
	webhook := len(os.Args) == 3 && os.Args[2] == "-w"
	if webhook {
		if err := myBot.RunWebhook(ctx, ":8080"); err != nil {
			panic(err)
		}
		return
	}
	myBot.Run(ctx)
}
