package plugins

import (
	"fmt"
	"strings"

	db "github.com/zkyrnx11/mack/src/sql"
)

func init() {
	Register(&Command{
		Pattern:  "antistatus",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Msg.Chat.String()
			sub := strings.ToLower(strings.TrimSpace(ctx.Text))
			switch sub {
			case "on":
				db.SetAntistatusEnabled(chatJID, true)
				ctx.Reply(T().AntistatusOn)
			case "off":
				db.SetAntistatusEnabled(chatJID, false)
				ctx.Reply(T().AntistatusOff)
			default:
				status := "off"
				if db.GetAntistatusEnabled(chatJID) {
					status = "on"
				}
				ctx.Reply(fmt.Sprintf("Antistatus is currently: *%s*\nUsage: .antistatus on|off", status))
			}
			return nil
		},
	})
}
