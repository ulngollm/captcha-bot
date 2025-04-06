package main

import (
	"captcha-bot/middleware"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v4"
)

const answerTimeoutSec = 30

const (
	answerSpam = "yes"
	answerOk   = "no"
)

var waitList Waitlist

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("godotenv.Load: %s", err)
		return
	}
}

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
	bot.Handle(tele.OnCallback, onAnswer, middleware.IgnoreOtherUsersClick)
	bot.Handle(tele.OnChatMember, onJoin, middleware.CheckJoin)
	bot.Handle(tele.OnText, removeNotApprovedMessages)

	sendStartupMessage(ok, err, bot)
	waitList = NewWaitlist()

	bot.Start()

}

func sendStartupMessage(ok bool, err error, bot *tele.Bot) {
	id, ok := os.LookupEnv("ADMIN_ID")
	if !ok {
		return
	}
	adminID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Fatalf("strconv.ParseInt: %s", err)
		return
	}
	if _, err := bot.Send(tele.ChatID(adminID), "bot started"); err != nil {
		log.Fatalf("send: %s", err)
		return
	}
}

func removeNotApprovedMessages(c tele.Context) error {
	userID := c.Sender().ID
	if !waitList.IsExists(userID) {
		return nil
	}
	return c.Delete()
}

func onJoin(c tele.Context) error {
	msg, err := sendCheckMessage(c, c.ChatMember().NewChatMember.User)
	if err != nil {
		return fmt.Errorf("sendCheckMessage: %w", err)
	}
	waitList.AddToList(c.ChatMember().NewChatMember.User.ID)

	_ = time.AfterFunc(answerTimeoutSec*time.Second, func() {
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
		waitList.RemoveFromList(member.User.ID)
	})

	return nil
}

func onAnswer(c tele.Context) error {
	switch c.Data() {
	case answerSpam:
		r := &tele.CallbackResponse{Text: "you are banned", ShowAlert: true}
		if err := c.Respond(r); err != nil {
			return fmt.Errorf("respond: %w", err)
		}
		if err := c.Bot().Restrict(c.Chat(), &tele.ChatMember{User: c.Callback().Sender}); err != nil {
			//todo bot.Restrict: telegram: Bad Request: method is available only for supergroups (400)
			return fmt.Errorf("bot.Restrict: %w", err)
		}
	case answerOk:
		r := &tele.CallbackResponse{Text: "you are ok", ShowAlert: true}
		if err := c.Respond(r); err != nil {
			return fmt.Errorf("respond: %w", err)
		}
	}
	waitList.RemoveFromList(c.Callback().Sender.ID)
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
	//todo build by config
	markup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				tele.InlineButton{Text: "design", Data: answerOk},
				tele.InlineButton{Text: "spam", Data: answerSpam},
			},
		},
	}

	// use link to name tg://user?id=<user_id>
	txt := fmt.Sprintf(
		"Привет, [%s](tg://user?id=%d)\\! Выбери кнопку\\.\\\n"+
			"У тебя есть %d секунд на ответ\\. Если не ответишь, отправка сообщений будет ограничена",
		user.FirstName,
		user.ID,
		answerTimeoutSec,
	)
	msg, err := c.Bot().Send(c.Chat(), txt, &tele.SendOptions{ParseMode: tele.ModeMarkdownV2}, markup)
	if err != nil {
		return &tele.Message{}, fmt.Errorf("send: %w", err)
	}
	return msg, nil
}
