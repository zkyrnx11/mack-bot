package plugins

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var antivvOn int32

func init() {
	Register(&Command{
		Pattern:  "antivv",
		IsSudo:   true,
		Category: "utility",
		Func: func(ctx *Context) error {
			sub := strings.ToLower(strings.TrimSpace(ctx.Text))
			switch sub {
			case "on":
				atomic.StoreInt32(&antivvOn, 1)
				ctx.Reply("Anti-viewonce enabled. View-once media will be forwarded to your DM.")
			case "off":
				atomic.StoreInt32(&antivvOn, 0)
				ctx.Reply("Anti-viewonce disabled.")
			default:
				status := "off"
				if atomic.LoadInt32(&antivvOn) == 1 {
					status = "on"
				}
				ctx.Reply(fmt.Sprintf("Anti-viewonce is currently: *%s*\nUsage: .antivv on|off", status))
			}
			return nil
		},
	})

	Register(&Command{
		Pattern:  "vv",
		Category: "utility",
		Func: func(ctx *Context) error {
			quoted := ctx.Quoted()
			if quoted == nil || quoted.Message == nil {
				ctx.Reply("Reply to a view-once message to reveal it.")
				return nil
			}

			fwd := revealViewOnce(quoted.Message, quoted.StanzaID, quoted.Participant, ctx.Msg.Chat.String())
			if fwd == nil {
				ctx.Reply("The quoted message is not a view-once media.")
				return nil
			}

			ctx.Client.SendMessage(context.Background(), ctx.Msg.Chat, fwd)
			return nil
		},
	})

	RegisterModerationHook(antivvHook)
}

func antivvHook(client *whatsmeow.Client, evt *events.Message) {
	if atomic.LoadInt32(&antivvOn) == 0 {
		return
	}
	if ownerPhone == "" || evt.Info.IsFromMe {
		return
	}

	msg := evt.Message
	fwd := revealViewOnce(msg, string(evt.Info.ID), evt.Info.Sender.String(), evt.Info.Chat.String())
	if fwd == nil {
		return
	}

	ownerJID := types.NewJID(ownerPhone, types.DefaultUserServer)
	if _, err := client.SendMessage(context.Background(), ownerJID, fwd); err != nil {
		fmt.Printf("[ANTIVV] send error: %v\n", err)
	}
}

func revealViewOnce(msg *waProto.Message, msgID, participant, remoteJID string) *waProto.Message {
	ci := &waProto.ContextInfo{
		StanzaID:      proto.String(msgID),
		Participant:   proto.String(participant),
		QuotedMessage: msg,
		RemoteJID:     proto.String(remoteJID),
	}

	if img := msg.GetImageMessage(); img != nil && img.GetViewOnce() {
		copied := proto.Clone(img).(*waProto.ImageMessage)
		copied.ViewOnce = proto.Bool(false)
		copied.ContextInfo = ci
		return &waProto.Message{ImageMessage: copied}
	}
	if vid := msg.GetVideoMessage(); vid != nil && vid.GetViewOnce() {
		copied := proto.Clone(vid).(*waProto.VideoMessage)
		copied.ViewOnce = proto.Bool(false)
		copied.ContextInfo = ci
		return &waProto.Message{VideoMessage: copied}
	}
	if aud := msg.GetAudioMessage(); aud != nil && aud.GetViewOnce() {
		copied := proto.Clone(aud).(*waProto.AudioMessage)
		copied.ViewOnce = proto.Bool(false)
		copied.ContextInfo = ci
		return &waProto.Message{AudioMessage: copied}
	}
	return nil
}
