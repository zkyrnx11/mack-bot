package plugins

import (
	"fmt"
	"regexp"
	"strings"

	db "github.com/zkyrnx11/mack/src/sql"
)

var urlRegex = regexp.MustCompile(`(?i)https?://[^\s]+`)

func init() {
	Register(&Command{
		Pattern:  "antilink",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Msg.Chat.String()
			args := ctx.Args

			if len(args) == 0 {
				mode := db.GetAntilinkMode(chatJID)
				ctx.Reply(menuHeader("antilink") + fmt.Sprintf(T().AntilinkStatus, mode))
				return nil
			}

			switch strings.ToLower(args[0]) {
			case "on":
				db.SetAntilinkMode(chatJID, "delete")
				ctx.Reply(T().AntilinkOn)
			case "off":
				db.SetAntilinkMode(chatJID, "off")
				ctx.Reply(T().AntilinkOff)
			case "set":
				if len(args) < 2 {
					ctx.Reply(T().AntilinkSetUsage)
					return nil
				}
				switch strings.ToLower(args[1]) {
				case "delete":
					db.SetAntilinkMode(chatJID, "delete")
					ctx.Reply(fmt.Sprintf(T().AntilinkSet, "delete"))
				case "kick":
					db.SetAntilinkMode(chatJID, "kick")
					ctx.Reply(fmt.Sprintf(T().AntilinkSet, "kick"))
				default:
					ctx.Reply(T().AntilinkUnknownAct)
				}
			default:
				ctx.Reply(T().AntilinkUsage)
			}
			return nil
		},
	})
}
