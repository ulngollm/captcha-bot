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

const answerTimeout = 5 * time.Second

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
	bot.Handle(tele.OnChatMember, onJoin)

	bot.Start()
}

func onJoin(c tele.Context) error {
	//only if user join. Exclude left
	designButton := tele.InlineButton{Text: "design", Data: answerOk}
	spamButton := tele.InlineButton{Text: "spam", Data: answerSpam}
	markup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{designButton, spamButton},
		},
	}

	// Send the initial message with inline buttons
	if err := c.Send(fmt.Sprintf("Привет, %s! Выбери, зачем пришел", c.Sender().FirstName), markup); err != nil {
		return err
	}

	// Start a goroutine to handle the timeout
	go func(member *tele.ChatMember) {
		time.Sleep(answerTimeout)
		// хак, как понять, что пользователь не ответил:
		// если ответил - сообщение удалится. Если оно еще осталось - значит пользователь не ответил и будет забанен
		// todo обработать кейсы, когда сообщение не удалилось по ошибке
		// todo не банить пользователя, если он сам ушел
		if err := c.Bot().Ban(c.Chat(), member); err != nil {
			log.Printf("Failed to ban user after timeout: %v", err)
		}
	}(c.ChatMember().NewChatMember)

	return nil
}

func onAnswer(c tele.Context) error {
	switch c.Data() {
	case answerSpam:
		r := &tele.CallbackResponse{Text: "you are banned"}
		if err := c.Respond(r); err != nil {
			return fmt.Errorf("respond: %w", err)
		}
		if err := c.Bot().Ban(c.Chat(), &tele.ChatMember{User: c.Callback().Sender}); err != nil {
			return fmt.Errorf("bot.Ban: %w", err)
		}
	case answerOk:
		r := &tele.CallbackResponse{Text: "you are ok"}
		if err := c.Respond(r); err != nil {
			return fmt.Errorf("respond: %w", err)
		}
	}
	if err := c.Delete(); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}
