package plugins

import (
	"fmt"
	"strings"

	db "github.com/zkyrnx11/mack/src/sql"
)

func init() {
	Register(&Command{
		Pattern:  "antiword",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Msg.Chat.String()
			args := ctx.Args

			if len(args) == 0 {
				ctx.Reply(menuHeader("antiword") + T().AntiwordUsage)
				return nil
			}

			switch strings.ToLower(args[0]) {
			case "list":
				words := db.GetAntiwords(chatJID)
				if len(words) == 0 {
					ctx.Reply(T().AntiwordEmpty)
					return nil
				}
				ctx.Reply(fmt.Sprintf(T().AntiwordList, strings.Join(words, "\n")))
			case "add":
				if len(args) < 2 {
					ctx.Reply(T().AntiwordAddUsage)
					return nil
				}
				word := strings.ToLower(args[1])
				db.AddAntiword(chatJID, word)
				ctx.Reply(fmt.Sprintf(T().AntiwordAdded, word))
			case "remove":
				if len(args) < 2 {
					ctx.Reply(T().AntiwordRemoveUsage)
					return nil
				}
				word := strings.ToLower(args[1])
				db.RemoveAntiword(chatJID, word)
				ctx.Reply(fmt.Sprintf(T().AntiwordRemoved, word))
			default:
				ctx.Reply(T().AntiwordUsage)
			}
			return nil
		},
	})
}
