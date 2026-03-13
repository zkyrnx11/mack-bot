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

func isGroupStatusMsg(evt *events.Message) bool {
	m := evt.Message
	if m.GetGroupStatusMentionMessage() != nil {
		return true
	}
	if m.GetGroupStatusMessage() != nil {
		return true
	}
	if m.GetGroupStatusMessageV2() != nil {
		return true
	}

	if ci := m.GetExtendedTextMessage().GetContextInfo(); ci != nil && ci.GetIsGroupStatus() {
		return true
	}
	if pm := m.GetProtocolMessage(); pm != nil {
		if pm.GetType() == waProto.ProtocolMessage_STATUS_MENTION_MESSAGE {
			return true
		}
	}
	if m.GetGroupInviteMessage() != nil {
		return true
	}
	return false
}

func revokeMsg(client *whatsmeow.Client, chat, sender types.JID, msgID string) {
	revoke := client.BuildRevoke(chat, sender, types.MessageID(msgID))
	queueMsg(client, chat, revoke)
}

func menuHeader(name string) string {
	return fmt.Sprintf("*.%s*\n-------------------\n", name)
}

func sendMentionToChat(client *whatsmeow.Client, chat types.JID, text string, jids []string) {
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: jids,
			},
		},
	}
	queueMsg(client, chat, msg)
}

func init() {
	RegisterModerationHook(moderationHook)

	// Periodically prune spam/afk cache tables.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-10 * time.Minute).Unix()
			db.PruneSpamEvents(cutoff)
			db.PruneDMSpam(cutoff)
			db.PruneAFKCooldown()
		}
	}()
}

func moderationHook(client *whatsmeow.Client, evt *events.Message) {

	myPhone := ""
	myLID := ""
	if client.Store.ID != nil {
		myPhone = strings.SplitN(client.Store.ID.User, ".", 2)[0]
	}
	myLID = client.Store.LID.User

	su := evt.Info.Sender.User
	sa := evt.Info.SenderAlt.User
	isFromMe := false
	if myPhone != "" && (su == myPhone || sa == myPhone) {
		isFromMe = true
	}
	if !isFromMe && myLID != "" && (su == myLID || sa == myLID) {
		isFromMe = true
	}

	if myPhone == "" && myLID == "" {
		isFromMe = evt.Info.IsFromMe
	}

	if isFromMe {
		if ownerPhone != "" {
			text := extractMsgText(evt)
			_, name, _, ok := parseCommand(text, BotSettings.GetPrefixes())
			if !ok || name != "afk" {
				db.ClearAFK(ownerPhone)
			}
		}
		return
	}

	chatJID := evt.Info.Chat.String()
	senderUser := evt.Info.Sender.User
	senderAlt := evt.Info.SenderAlt.User
	isGroup := evt.Info.Chat.Server == types.GroupServer
	msgText := extractMsgText(evt)

	chatServer := evt.Info.Chat.Server
	if chatServer == types.BroadcastServer || chatServer == types.NewsletterServer {
		return
	}

	// ── AFK auto-reply ──────────────────────────────────────────────────
	if ownerPhone != "" {
		if status := db.GetAFK(ownerPhone); status != nil {
			shouldReply := false
			if !isGroup {
				shouldReply = true
			} else {
				participant := evt.Message.GetExtendedTextMessage().GetContextInfo().GetParticipant()
				mentionedJIDs := evt.Message.GetExtendedTextMessage().GetContextInfo().GetMentionedJID()
				if participant != "" {
					partUser := strings.Split(participant, "@")[0]
					if partUser == ownerPhone || (myLID != "" && partUser == myLID) {
						shouldReply = true
					}
				}
				if !shouldReply {
					for _, jid := range mentionedJIDs {
						if strings.HasPrefix(jid, ownerPhone+"@") || (myLID != "" && strings.HasPrefix(jid, myLID+"@")) {
							shouldReply = true
							break
						}
					}
				}
			}
			if shouldReply {
				cooldownKey := chatJID + ":" + senderUser
				if !db.IsAFKCooldownActive(cooldownKey) {
					db.SetAFKCooldown(cooldownKey, time.Now().Add(30*time.Second))
					lastSeen := status.SetAt.Format("3:04PM, 02 Jan 2006")
					reply := fmt.Sprintf(T().AFKAutoReply, lastSeen)
					if status.Message != "" {
						reply += "\n\n" + status.Message
					}
					reply += "\n\n" + T().AFKDefaultMsg
					queueMsg(client, evt.Info.Chat,
						&waProto.Message{Conversation: proto.String(reply)})
				}
			}
		}
	}

	// ── DM spam detection (SQLite-backed) ────────────────────────────────
	if !isGroup {

		if time.Since(evt.Info.Timestamp) > 30*time.Second {
			return
		}
		db.RecordDMSpamEvent(senderUser)
		count := db.CountRecentDMSpam(senderUser, 5)

		if count > 3 {
			if db.IsDMSpamWarned(senderUser) {
				senderJID := types.NewJID(senderUser, types.DefaultUserServer)
				if senderAlt != "" {
					senderJID = types.NewJID(senderAlt, types.DefaultUserServer)
				}
				client.UpdateBlocklist(context.Background(), senderJID, events.BlocklistChangeActionBlock)
				db.SetDMSpamWarned(senderUser, false)
			} else {
				queueMsg(client, evt.Info.Chat,
					&waProto.Message{Conversation: proto.String(T().AntispamWarn)})
				db.SetDMSpamWarned(senderUser, true)
			}
		}
		return
	}

	// ── Group moderation ─────────────────────────────────────────────────
	var (
		participants    []types.GroupParticipant
		groupInfoLoaded bool
	)
	botJID := client.Store.ID.ToNonAD()

	loadGroup := func() {
		if !groupInfoLoaded {
			groupInfoLoaded = true
			if gi, err := client.GetGroupInfo(context.Background(), evt.Info.Chat); err == nil {
				participants = gi.Participants
			}
		}
	}

	isBotAdmin := func() bool {
		loadGroup()
		return botIsAdmin(participants, ownerPhone, botJID.User)
	}

	isSenderAdmin := func() bool {
		loadGroup()
		p := findParticipant(participants, senderUser, "")
		if p == nil && senderAlt != "" {
			p = findParticipant(participants, senderAlt, "")
		}
		return p != nil && (p.IsAdmin || p.IsSuperAdmin)
	}

	if db.GetAntistatusEnabled(chatJID) && isBotAdmin() && !isSenderAdmin() {
		if isGroupStatusMsg(evt) {
			revokeMsg(client, evt.Info.Chat, evt.Info.Sender, string(evt.Info.ID))
			senderJIDStr := evt.Info.Sender.ToNonAD().String()
			notify := fmt.Sprintf(T().AntistatusNotify, senderUser)
			sendMentionToChat(client, evt.Info.Chat, notify, []string{senderJIDStr})
			return
		}
	}

	if db.IsShhed(chatJID, senderUser) && isBotAdmin() {
		revokeMsg(client, evt.Info.Chat, evt.Info.Sender, string(evt.Info.ID))
		return
	}

	if mode := db.GetAntilinkMode(chatJID); mode != "off" && msgText != "" {
		if isBotAdmin() && !isSenderAdmin() {
			if urlRegex.MatchString(msgText) {
				revokeMsg(client, evt.Info.Chat, evt.Info.Sender, string(evt.Info.ID))
				senderJIDStr := evt.Info.Sender.ToNonAD().String()
				notify := fmt.Sprintf(T().AntilinkNotify, senderUser)
				sendMentionToChat(client, evt.Info.Chat, notify, []string{senderJIDStr})
				if mode == "kick" {
					client.UpdateGroupParticipants(context.Background(), evt.Info.Chat,
						[]types.JID{evt.Info.Sender.ToNonAD()}, whatsmeow.ParticipantChangeRemove)
				}
				return
			}
		}
	}

	if words := db.GetAntiwords(chatJID); len(words) > 0 && msgText != "" {
		if isBotAdmin() && !isSenderAdmin() {
			lower := strings.ToLower(msgText)
			for _, w := range words {
				if strings.Contains(lower, strings.ToLower(w)) {
					revokeMsg(client, evt.Info.Chat, evt.Info.Sender, string(evt.Info.ID))
					return
				}
			}
		}
	}

	// ── Group spam detection (SQLite-backed) ─────────────────────────────
	if db.GetAntispamMode(chatJID) != "off" {

		if time.Since(evt.Info.Timestamp) <= 30*time.Second && !db.IsAntispamWhitelisted(chatJID, senderUser) {
			spamKey := chatJID + ":" + senderUser
			db.RecordSpamEvent(spamKey, senderUser, string(evt.Info.ID))
			count := db.CountRecentSpam(spamKey, senderUser, 5)

			if count > 3 && isBotAdmin() {
				msgIDs := db.GetRecentSpamMsgIDs(spamKey, senderUser, 5)
				for _, msgID := range msgIDs {
					revokeMsg(client, evt.Info.Chat, evt.Info.Sender, msgID)
				}
				client.UpdateGroupParticipants(context.Background(), evt.Info.Chat,
					[]types.JID{evt.Info.Sender.ToNonAD()}, whatsmeow.ParticipantChangeRemove)
			}
		}
	}
}
