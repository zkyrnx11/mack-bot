package plugins

import (
	"fmt"
	"strings"

	db "github.com/zkyrnx11/mack/src/sql"
)

func init() {
	Register(&Command{
		Pattern:  "antispam",
		IsGroup:  false,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Msg.Chat.String()
			args := ctx.Args

			if len(args) == 0 {
				mode := db.GetAntispamMode(chatJID)
				ctx.Reply(menuHeader("antispam") + fmt.Sprintf(T().AntispamStatus, mode))
				return nil
			}

			switch strings.ToLower(args[0]) {
			case "on":
				db.SetAntispamMode(chatJID, "on")
				ctx.Reply(T().AntispamOn)
			case "off":
				db.SetAntispamMode(chatJID, "off")
				ctx.Reply(T().AntispamOff)
			case "allow":
				arg := ""
				if len(args) > 1 {
					arg = args[1]
				}
				phone, lid := ResolveTarget(ctx, arg)
				if phone == "" && lid == "" {
					ctx.Reply(T().UserResolveFail)
					return nil
				}
				userID := phone
				if userID == "" {
					userID = lid
				}
				db.SetAntispamWhitelist(chatJID, userID, true)
				ctx.Reply(T().AntispamAllowed)
			default:
				ctx.Reply(T().AntispamUsage)
			}
			return nil
		},
	})
}
