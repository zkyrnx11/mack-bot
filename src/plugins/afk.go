package plugins

import (
	"fmt"
	"strings"

	db "github.com/zkyrnx11/mack/src/sql"
)

func init() {
	Register(&Command{
		Pattern:  "afk",
		Category: "utility",
		Func: func(ctx *Context) error {
			args := ctx.Args
			if len(args) == 0 {
				ctx.Reply(menuHeader("afk") + "on — enable AFK\noff — disable AFK\nset <message> — set custom away message")
				return nil
			}
			sub := strings.ToLower(args[0])

			userKey := ownerPhone
			if userKey == "" {
				userKey = ctx.Msg.Sender.User
			}
			switch sub {
			case "on":
				existing := db.GetAFK(userKey)
				msg := ""
				if existing != nil {
					msg = existing.Message
				}
				db.SetAFK(userKey, msg)
				ctx.Reply(T().AFKEnabled)
			case "off":
				if db.GetAFK(userKey) == nil {
					ctx.Reply(T().AFKNotActive)
					return nil
				}
				db.ClearAFK(userKey)
				ctx.Reply(T().AFKOff)
			case "set":
				msg := strings.TrimSpace(strings.TrimPrefix(ctx.Text, args[0]))
				if msg == "" {
					ctx.Reply(T().AFKSetUsage)
					return nil
				}
				db.SetAFK(userKey, msg)
				ctx.Reply(fmt.Sprintf("%s\nMessage: %s", T().AFKEnabled, msg))
			default:
				ctx.Reply(menuHeader("afk") + "on — enable AFK\noff — disable AFK\nset <message> — set custom away message")
			}
			return nil
		},
	})
}
