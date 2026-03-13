package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	db "github.com/zkyrnx11/mack/src/sql"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const msgCacheTTL = 30 * time.Minute

func init() {

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-msgCacheTTL).Unix()
			db.PruneCache(cutoff)
		}
	}()
}

func cacheMessage(evt *events.Message) {
	if evt.Message.GetProtocolMessage() != nil {
		return
	}
	blob, err := proto.Marshal(evt.Message)
	if err != nil {
		return
	}
	altJID := ""
	if !evt.Info.SenderAlt.IsEmpty() {
		altJID = evt.Info.SenderAlt.String()
	}
	fromMe := 0
	if evt.Info.IsFromMe {
		fromMe = 1
	}
	db.InsertCachedMessage(
		string(evt.Info.ID),
		evt.Info.Chat.String(),
		evt.Info.Sender.String(),
		altJID,
		fromMe,
		evt.Info.Timestamp.Unix(),
		time.Now().Unix(),
		blob,
	)
}

type dbCachedMsg struct {
	ChatJID   string
	SenderJID string
	SenderAlt string
	IsFromMe  bool
	MsgTS     int64
	Message   *waProto.Message
}

func popCachedMessage(msgID string) (*dbCachedMsg, bool) {
	row, ok := db.PopCachedMessage(msgID)
	if !ok {
		return nil, false
	}
	msg := &waProto.Message{}
	if err := proto.Unmarshal(row.Blob, msg); err != nil {
		return nil, false
	}
	return &dbCachedMsg{
		ChatJID:   row.ChatJID,
		SenderJID: row.SenderJID,
		SenderAlt: row.SenderAlt,
		IsFromMe:  row.IsFromMe == 1,
		MsgTS:     row.MsgTS,
		Message:   msg,
	}, true
}

func init() {
	Register(&Command{
		Pattern:  "antidelete",
		IsSudo:   true,
		Category: "utility",
		Func: func(ctx *Context) error {
			sub := strings.ToLower(strings.TrimSpace(ctx.Text))
			BotSettings.mu.Lock()
			switch sub {
			case "on":
				BotSettings.AntiDelete = true
				BotSettings.mu.Unlock()
				SaveSettings()
				ctx.Reply("✅ AntiDelete enabled.")
			case "off":
				BotSettings.AntiDelete = false
				BotSettings.mu.Unlock()
				SaveSettings()
				ctx.Reply("AntiDelete disabled.")
			default:
				status := "off"
				if BotSettings.AntiDelete {
					status = "on"
				}
				BotSettings.mu.Unlock()
				ctx.Reply(fmt.Sprintf("AntiDelete is currently: *%s*\nUsage: .antidelete on|off", status))
			}
			return nil
		},
	})

	RegisterModerationHook(antiDeleteHook)
}

func antiDeleteHook(client *whatsmeow.Client, evt *events.Message) {

	// Only cache messages when antidelete is actually on.
	BotSettings.mu.RLock()
	on := BotSettings.AntiDelete
	BotSettings.mu.RUnlock()
	if on {
		cacheMessage(evt)
	}

	pm := evt.Message.GetProtocolMessage()
	if pm == nil {
		return
	}
	if pm.GetType() != waProto.ProtocolMessage_REVOKE {
		return
	}

	BotSettings.mu.RLock()
	adOn := BotSettings.AntiDelete
	BotSettings.mu.RUnlock()
	if !adOn {
		return
	}

	if evt.Info.IsFromMe {
		return
	}

	deleterUser := evt.Info.Sender.User
	deleterAlt := evt.Info.SenderAlt.User
	if BotSettings.IsSudo(deleterUser) || (deleterAlt != "" && BotSettings.IsSudo(deleterAlt)) {
		return
	}

	isGroup := evt.Info.Chat.Server == types.GroupServer

	if isGroup {
		gi, err := client.GetGroupInfo(context.Background(), evt.Info.Chat)
		if err == nil {
			p := findParticipant(gi.Participants, deleterAlt, deleterUser)
			if p != nil && (p.IsAdmin || p.IsSuperAdmin) {
				return
			}
		}
	}

	deletedID := pm.GetKey().GetID()
	cached, ok := popCachedMessage(deletedID)
	if !ok {
		return
	}

	senderParsed, _ := types.ParseJID(cached.SenderJID)
	chatParsed, _ := types.ParseJID(cached.ChatJID)
	senderUser := senderParsed.User
	ts := time.Unix(cached.MsgTS, 0).Local().Format("15:04:05")

	ci := &waProto.ContextInfo{
		StanzaID:      proto.String(deletedID),
		Participant:   proto.String(cached.SenderJID),
		QuotedMessage: cached.Message,
		RemoteJID:     proto.String(cached.ChatJID),
	}

	var mentionJIDs []string
	if isGroup {
		mentionJIDs = []string{cached.SenderJID, evt.Info.Sender.String()}
	} else {
		mentionJIDs = []string{cached.SenderJID}
	}
	ci.MentionedJID = mentionJIDs

	fakeInfo := types.MessageInfo{MessageSource: types.MessageSource{Chat: chatParsed, Sender: senderParsed}}
	msg := cached.Message
	textContent := extractText(&events.Message{Info: fakeInfo, Message: msg})
	contentDesc := antiDeleteContentDesc(msg, textContent)

	var alertText string
	if isGroup {
		alertText = fmt.Sprintf("*AntiDelete*\n\nMessage: %s\nTime: %s\nSent By: @%s\nDeleted By: @%s",
			contentDesc, ts, senderUser, deleterUser)
	} else {
		alertText = fmt.Sprintf("*AntiDelete*\n\nMessage: %s\nTime: %s\nSent By: @%s",
			contentDesc, ts, senderUser)
	}

	var fwd *waProto.Message
	switch {
	case msg.GetImageMessage() != nil:
		copied := proto.Clone(msg.GetImageMessage()).(*waProto.ImageMessage)
		copied.ViewOnce = proto.Bool(false)
		copied.Caption = proto.String(alertText)
		copied.ContextInfo = ci
		fwd = &waProto.Message{ImageMessage: copied}

	case msg.GetVideoMessage() != nil:
		copied := proto.Clone(msg.GetVideoMessage()).(*waProto.VideoMessage)
		copied.ViewOnce = proto.Bool(false)
		copied.Caption = proto.String(alertText)
		copied.ContextInfo = ci
		fwd = &waProto.Message{VideoMessage: copied}

	case msg.GetAudioMessage() != nil:
		textFwd := &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text:        proto.String(alertText),
				ContextInfo: ci,
			},
		}
		dest := antiDeleteDest(isGroup, evt)
		client.SendMessage(context.Background(), dest, textFwd)
		copied := proto.Clone(msg.GetAudioMessage()).(*waProto.AudioMessage)
		copied.ContextInfo = ci
		fwd = &waProto.Message{AudioMessage: copied}

	default:
		fwd = &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text:        proto.String(alertText),
				ContextInfo: ci,
			},
		}
	}

	dest := antiDeleteDest(isGroup, evt)
	if _, err := client.SendMessage(context.Background(), dest, fwd); err != nil {
		fmt.Printf("[ANTIDELETE] send error: %v\n", err)
	}
}

func antiDeleteDest(isGroup bool, evt *events.Message) types.JID {
	if isGroup {
		return evt.Info.Chat
	}
	return types.NewJID(ownerPhone, types.DefaultUserServer)
}

func antiDeleteContentDesc(msg *waProto.Message, text string) string {
	switch {
	case msg.GetImageMessage() != nil:
		if text != "" {
			return "(image) " + text
		}
		return "(image)"
	case msg.GetVideoMessage() != nil:
		if text != "" {
			return "(video) " + text
		}
		return "(video)"
	case msg.GetAudioMessage() != nil:
		return "(audio)"
	case msg.GetStickerMessage() != nil:
		return "(sticker)"
	default:
		return text
	}
}
