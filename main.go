package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v4"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("godotenv.Load: %s", err)
		return
	}
}

const answerTimeout = 10 * time.Second

const (
	answerSpam = "yes"
	answerOk   = "no"
)

func main() {
	t, ok := os.LookupEnv("BOT_TOKEN")
	if !ok {
		log.Fatalf("bot token is empty")
		return
	}

	pref := tele.Settings{
		Token: t,
		Poller: &tele.LongPoller{
			Timeout: time.Second,
			AllowedUpdates: []string{
				"callback_query",
				"message",
				"chat_member",
			},
		},
	}
	bot, err := tele.NewBot(pref)
	if err != nil {
		log.Fatalf("tele.NewBot: %s", err)
		return
	}

	bot.Handle(tele.OnCallback, onAnswer)
	bot.Handle(tele.OnUserJoined, onJoin)

	bot.Start()
}

func onJoin(c tele.Context) error {
	//	todo add goroutine - if user is not answered after timeout - remove him
	//todo send message with keyboard
	return nil
}

func onAnswer(c tele.Context) error {
	switch c.Data() {
	case answerSpam:
		r := &tele.CallbackResponse{
			CallbackID: c.Callback().ID,
			Text:       "you are banned",
		}
		if err := c.Respond(r); err != nil {
			return fmt.Errorf("respond: %w", err)
		}
		if err := c.Bot().Ban(c.Chat(), &tele.ChatMember{User: c.Message().Sender}); err != nil {
			return fmt.Errorf("bot.Ban: %w", err)
		}
	case answerOk:
		r := &tele.CallbackResponse{
			CallbackID: c.Callback().ID,
			Text:       "you are ok",
		}
		if err := c.Respond(r); err != nil {
			return fmt.Errorf("respond: %w", err)
		}
	}
	if err := c.Delete(); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}
