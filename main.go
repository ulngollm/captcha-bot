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
	bot.Handle(tele.OnChatMember, onJoin)

	bot.Start()
}

func onJoin(c tele.Context) error {
	// technically it is available to kick user from channel. Avoid this
	if c.Chat().Type != tele.ChatSuperGroup && c.Chat().Type != tele.ChatGroup {
		return nil
	}
	//only if user join. Exclude left
	// также может быть restricted. Можно дать ему шанс ответить еще раз
	newRole := c.Update().ChatMember.NewChatMember.Role
	if newRole != tele.Member && newRole != tele.Restricted {
		return nil
	}
	oldRole := c.Update().ChatMember.OldChatMember.Role
	// этот же хендлер срабатывает, если с пользователя были сняты ограничения
	// не присылать ничего в таком случае
	if newRole != tele.Member && oldRole != tele.Restricted {
		return nil
	}

	designButton := tele.InlineButton{Text: "design", Data: answerOk}
	spamButton := tele.InlineButton{Text: "spam", Data: answerSpam}
	markup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{designButton, spamButton},
		},
	}

	// use link to name tg://user?id=<user_id>
	if err := c.Send(
		fmt.Sprintf(
			"Привет, [%s](tg://user?id=%d)\\! Выбери, зачем пришел",
			c.Sender().FirstName,
			c.Sender().ID,
		), &tele.SendOptions{ParseMode: tele.ModeMarkdownV2}, markup); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	// Start a goroutine to handle the timeout
	go func(c tele.Context) {
		time.Sleep(answerTimeout)
		// проверить, что пользователь еще состоит в чате
		// не банить пользователя, если он сам ушел
		member := c.ChatMember().NewChatMember
		m, err := c.Bot().ChatMemberOf(c.Chat(), member.User)
		if err != nil {
			return
		}
		if m.Role != tele.Member {
			return
		}

		// хак, как понять, что пользователь не ответил:
		// если ответил - сообщение удалится. Если оно еще осталось - значит пользователь не ответил и будет забанен
		// todo обработать кейсы, когда сообщение не удалилось по ошибке
		member.RestrictedUntil = time.Now().Add(1 * time.Hour).Unix() // в таком случае блочим только на час
		if err := c.Bot().Restrict(c.Chat(), member); err != nil {
			log.Printf("Failed to ban user after timeout: %v", err)
		}
		//	todo удалить сообщение. Для этого надо прокинуть id ?
	}(c)

	return nil
}

func onAnswer(c tele.Context) error {
	//todo игнорить, если кнопку нажал другой пользователь
	userToAsk := c.Callback().Message.Entities[0].User.ID
	if c.Callback().Sender.ID != userToAsk {
		return c.Respond(&tele.CallbackResponse{Text: "Это не вам."})
	}
	switch c.Data() {
	case answerSpam:
		r := &tele.CallbackResponse{Text: "you are banned"}
		if err := c.Respond(r); err != nil {
			return fmt.Errorf("respond: %w", err)
		}
		if err := c.Bot().Restrict(c.Chat(), &tele.ChatMember{User: c.Callback().Sender}); err != nil {
			return fmt.Errorf("bot.Restrict: %w", err)
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
