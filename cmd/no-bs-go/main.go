package main

import (
	"log"
	"os"

	"net/http"
	_ "net/http/pprof"

	internal_bot "github.com/etra0/no-bs-go/internal"
	tbot "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	go func() {
		log.Println("Starting pprof server on localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	bot, err := tbot.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatal("Something went wrong: ", err)
	}

	u := tbot.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	bot_handler := internal_bot.NewBot("https://co.wuk.sh/")

	go bot_handler.RunDispatcher(bot)

	log.Println("Starting to get messages.")
	for update := range updates {
		if update.Message == nil {
			continue
		}

		go bot_handler.HandleMessage(update.Message)
	}
}
