package plugins

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var heartEmojis = []string{"❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "🤍", "🤎", "💗", "💓", "💞", "💕", "❤️‍🔥"}

func randomHeart() string {
	return heartEmojis[rand.Intn(len(heartEmojis))]
}

func init() {
	Register(&Command{
		Pattern:  "autosavestatus",
		IsSudo:   true,
		Category: "utility",
		Func: func(ctx *Context) error {
			sub := strings.ToLower(strings.TrimSpace(ctx.Text))
			switch sub {
			case "on":
				BotSettings.mu.Lock()
				BotSettings.AutoSaveStatus = true
				BotSettings.mu.Unlock()
				_ = SaveSettings()
				ctx.Reply("Auto-save status enabled. Incoming status media will be forwarded to your DM.")
			case "off":
				BotSettings.mu.Lock()
				BotSettings.AutoSaveStatus = false
				BotSettings.mu.Unlock()
				_ = SaveSettings()
				ctx.Reply("Auto-save status disabled.")
			default:
				BotSettings.mu.RLock()
				on := BotSettings.AutoSaveStatus
				BotSettings.mu.RUnlock()
				status := "off"
				if on {
					status = "on"
				}
				ctx.Reply(fmt.Sprintf("Auto-save status is currently: *%s*\nUsage: .autosavestatus on|off", status))
			}
			return nil
		},
	})

	Register(&Command{
		Pattern:  "autolikestatus",
		IsSudo:   true,
		Category: "utility",
		Func: func(ctx *Context) error {
			sub := strings.ToLower(strings.TrimSpace(ctx.Text))
			switch sub {
			case "on":
				BotSettings.mu.Lock()
				BotSettings.AutoLikeStatus = true
				BotSettings.mu.Unlock()
				_ = SaveSettings()
				ctx.Reply("Auto-like status enabled. All statuses will be reacted with a random heart.")
			case "off":
				BotSettings.mu.Lock()
				BotSettings.AutoLikeStatus = false
				BotSettings.mu.Unlock()
				_ = SaveSettings()
				ctx.Reply("Auto-like status disabled.")
			default:
				BotSettings.mu.RLock()
				on := BotSettings.AutoLikeStatus
				BotSettings.mu.RUnlock()
				status := "off"
				if on {
					status = "on"
				}
				ctx.Reply(fmt.Sprintf("Auto-like status is currently: *%s*\nUsage: .autolikestatus on|off", status))
			}
			return nil
		},
	})

	RegisterModerationHook(autoStatusHook)
}

func autoStatusHook(client *whatsmeow.Client, evt *events.Message) {
	if evt.Info.Chat != types.StatusBroadcastJID || evt.Info.IsFromMe {
		return
	}

	BotSettings.mu.RLock()
	saveOn := BotSettings.AutoSaveStatus
	likeOn := BotSettings.AutoLikeStatus
	BotSettings.mu.RUnlock()

	if !saveOn && !likeOn {
		return
	}

	if likeOn {
		reaction := client.BuildReaction(
			types.StatusBroadcastJID,
			evt.Info.Sender,
			evt.Info.ID,
			randomHeart(),
		)
		if _, err := client.SendMessage(context.Background(), types.StatusBroadcastJID, reaction); err != nil {
			fmt.Printf("[AUTOLIKE] send error: %v\n", err)
		}
	}

	if saveOn {
		go forwardStatusToOwner(client, evt)
	}
}

func forwardStatusToOwner(client *whatsmeow.Client, evt *events.Message) {
	if ownerPhone == "" {
		return
	}

	msg := evt.Message
	img := msg.GetImageMessage()
	vid := msg.GetVideoMessage()
	aud := msg.GetAudioMessage()
	txt := msg.GetExtendedTextMessage()
	conv := msg.GetConversation()

	if img == nil && vid == nil && aud == nil && txt == nil && conv == "" {
		return
	}

	ci := &waProto.ContextInfo{
		StanzaID:      proto.String(string(evt.Info.ID)),
		Participant:   proto.String(evt.Info.Sender.String()),
		QuotedMessage: msg,
		RemoteJID:     proto.String(types.StatusBroadcastJID.String()),
	}

	var fwd *waProto.Message
	switch {
	case img != nil:
		copied := proto.Clone(img).(*waProto.ImageMessage)
		copied.ViewOnce = proto.Bool(false)
		copied.ContextInfo = ci
		if copied.GetCaption() == "" {
			copied.Caption = nil
		}
		fwd = &waProto.Message{ImageMessage: copied}
	case vid != nil:
		copied := proto.Clone(vid).(*waProto.VideoMessage)
		copied.ViewOnce = proto.Bool(false)
		copied.ContextInfo = ci
		if copied.GetCaption() == "" {
			copied.Caption = nil
		}
		fwd = &waProto.Message{VideoMessage: copied}
	case aud != nil:
		copied := proto.Clone(aud).(*waProto.AudioMessage)
		copied.ContextInfo = ci
		fwd = &waProto.Message{AudioMessage: copied}
	case txt != nil:
		copied := proto.Clone(txt).(*waProto.ExtendedTextMessage)
		copied.ContextInfo = ci
		fwd = &waProto.Message{ExtendedTextMessage: copied}
	default:
		fwd = &waProto.Message{ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text:        proto.String(conv),
			ContextInfo: ci,
		}}
	}

	ownerJID := types.NewJID(ownerPhone, types.DefaultUserServer)
	if _, err := client.SendMessage(context.Background(), ownerJID, fwd); err != nil {
		fmt.Printf("[AUTOSAVE] forward error: %v\n", err)
	} else {
		fmt.Printf("[AUTOSAVE] forwarded status from %s at %s\n", evt.Info.Sender.User, time.Now().Format("15:04:05"))
	}
}
