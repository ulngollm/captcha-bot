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

const answerTimeout = 30 * time.Second

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

	bot.Handle("/start", info)

	bot.Handle(tele.OnCallback, onAnswer)
	bot.Handle(tele.OnChatMember, onJoin, CheckJoin)

	bot.Start()
}

func onJoin(c tele.Context) error {
	msg, err := sendCheckMessage(c, c.ChatMember().NewChatMember.User)
	if err != nil {
		return fmt.Errorf("sendCheckMessage: %w", err)
	}

	_ = time.AfterFunc(answerTimeout, func() {
		b := c.Bot().(*tele.Bot)
		// хак, как понять, что пользователь не ответил:
		// если ответил - сообщение удалится и при удалении будет ошибка
		//Если оно еще осталось - значит пользователь не ответил и будет заблочен
		if err := b.Delete(msg); err != nil {
			b.OnError(fmt.Errorf("afterFunc.delete: %w", err), c)
			return
		}

		// проверить, что пользователь еще состоит в чате. Не банить пользователя, если он сам ушел
		member, err := b.ChatMemberOf(c.Chat(), c.ChatMember().NewChatMember.User) // sender - это не обязательно тот кто вступил. Смотреть надо именно мембера
		if err != nil {
			b.OnError(fmt.Errorf("chatMemberOf: %w", err), c)
		}
		if member.Role != tele.Member {
			return
		}

		if err := b.Restrict(c.Chat(), member); err != nil {
			b.OnError(fmt.Errorf("afterFunc.restrict: %w", err), c)
		}
	})

	return nil
}

func onAnswer(c tele.Context) error {
	//игнорить, если кнопку нажал другой пользователь
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
			//todo bot.Restrict: telegram: Bad Request: method is available only for supergroups (400)
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

func info(c tele.Context) error {
	if c.Chat().Type != tele.ChatPrivate {
		return nil
	}
	if _, err := sendCheckMessage(c, c.Sender()); err != nil {
		return fmt.Errorf("sendCheckMessage: %w", err)
	}
	return c.Send("Это сообщение отправлено ознакомительно")
}

func sendCheckMessage(c tele.Context, user *tele.User) (*tele.Message, error) {
	markup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				tele.InlineButton{Text: "design", Data: answerOk},
				tele.InlineButton{Text: "spam", Data: answerSpam},
			},
		},
	}

	// use link to name tg://user?id=<user_id>
	msg, err := c.Bot().Send(
		c.Chat(),
		fmt.Sprintf(
			"Привет, [%s](tg://user?id=%d)\\! Выбери, зачем пришел",
			user.FirstName,
			user.ID,
		), &tele.SendOptions{ParseMode: tele.ModeMarkdownV2}, markup)
	if err != nil {
		return &tele.Message{}, fmt.Errorf("send: %w", err)
	}
	return msg, nil
}

func CheckJoin(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		// technically it is available to kick user from channel. Avoid this
		if c.Chat().Type != tele.ChatSuperGroup && c.Chat().Type != tele.ChatGroup {
			return nil
		}
		//only if user join. Exclude left
		if c.ChatMember().OldChatMember.Member {
			return nil
		}
		//don't send message if user is already restricted ??
		if c.ChatMember().NewChatMember.Role == tele.Restricted {
			return nil
		}
		//ignore if user added by admin
		admins, err := c.Bot().AdminsOf(c.Chat())
		if err != nil {
			return fmt.Errorf("adminsOf: %w", err)
		}
		for _, admin := range admins {
			if c.ChatMember().Sender.ID == admin.User.ID {
				return nil
			}
		}
		//todo обработка ботов
		return next(c)
	}
}
