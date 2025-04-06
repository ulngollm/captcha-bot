package middleware

import (
	"fmt"
	tele "gopkg.in/telebot.v4"
)

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

func IgnoreOtherUsersClick(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		userToAsk := c.Callback().Message.Entities[0].User.ID
		if c.Callback().Sender.ID != userToAsk {
			return c.Respond(&tele.CallbackResponse{Text: "Это не вам."})
		}
		return next(c)
	}
}
