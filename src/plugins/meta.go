package plugins

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	db "github.com/zkyrnx11/mack/src/sql"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var MetaJID = types.NewMetaAIJID

var metaMu sync.Mutex

type metaPendingReply struct {
	chatJID   types.JID
	msgID     types.MessageID
	senderJID string
}

var pendingReplies = make(map[string]metaPendingReply)

var lastProcessedResponse = make(map[string]string)

var sentMessageIDs = make(map[string]types.MessageID)

const metaMapMaxSize = 50

func init() {
	// Periodically sweep unbounded meta maps to prevent memory leaks.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			metaMu.Lock()
			if len(pendingReplies) > metaMapMaxSize {
				pendingReplies = make(map[string]metaPendingReply)
			}
			if len(lastProcessedResponse) > metaMapMaxSize {
				lastProcessedResponse = make(map[string]string)
			}
			if len(sentMessageIDs) > metaMapMaxSize {
				sentMessageIDs = make(map[string]types.MessageID)
			}
			metaMu.Unlock()
		}
	}()
}

func buildMetaQuery(senderID, pushName, pastContext, userQuery string) string {
	if pastContext != "" {
		return fmt.Sprintf(
			"User ID: %s, Their Name: %s, Past Context — You Meta AI: %s, Their reply to your message: %s",
			senderID, pushName, pastContext, userQuery,
		)
	}
	return fmt.Sprintf("User ID: %s, Their Name: %s, Query: %s", senderID, pushName, userQuery)
}

func resolveParticipant(participant string) (id, name string) {
	if participant == "" {
		return "unknown", "unknown"
	}
	jid, err := types.ParseJID(participant)
	if err != nil {
		return participant, participant
	}
	id = jid.User
	if strings.HasSuffix(participant, "@lid") {
		name = db.GetContactNameByLID(jid.User)
	} else {
		name = db.GetContactName(participant)
	}
	if name == "" {
		name = id
	}
	return id, name
}

func senderPushName(evt *events.Message) string {
	if evt.Info.PushName != "" {
		return evt.Info.PushName
	}
	return evt.Info.Sender.User
}

func HandleMetaAIResponse(client *whatsmeow.Client, v *events.Message) {

	if v.Info.IsGroup {
		return
	}
	var responseText string
	resID := v.Message.GetMessageContextInfo().GetBotMetadata().GetBotResponseID()

	if img := v.Message.GetImageMessage(); img != nil {
		metaMu.Lock()
		reply, ok := pendingReplies[v.Info.Sender.String()]
		metaMu.Unlock()
		if ok {
			client.SendMessage(context.Background(), reply.chatJID, &waProto.Message{ImageMessage: img})
		}
		return
	}

	if v.Message.Conversation != nil {
		responseText = v.Message.GetConversation()
	} else if v.Message.ExtendedTextMessage != nil {
		responseText = v.Message.GetExtendedTextMessage().GetText()
	} else if v.Message.ProtocolMessage != nil &&
		v.Message.ProtocolMessage.GetType() == waProto.ProtocolMessage_MESSAGE_EDIT {
		edit := v.Message.ProtocolMessage.EditedMessage
		if edit != nil {
			if edit.Conversation != nil {
				responseText = edit.GetConversation()
			} else if edit.ExtendedTextMessage != nil {
				responseText = edit.ExtendedTextMessage.GetText()
			}
		}
	}

	if responseText == "" || resID == "" {
		return
	}

	metaMu.Lock()
	defer metaMu.Unlock()

	if lastText, seen := lastProcessedResponse[resID]; seen && len(responseText) <= len(lastText) {
		return
	}

	reply, ok := pendingReplies[v.Info.Sender.String()]
	if !ok {
		return
	}
	targetJID := reply.chatJID

	if msgID, exists := sentMessageIDs[resID]; exists {
		editMsg := client.BuildEdit(targetJID, msgID, &waProto.Message{
			Conversation: proto.String(responseText),
		})
		if _, err := client.SendMessage(context.Background(), targetJID, editMsg); err == nil {
			lastProcessedResponse[resID] = responseText
			db.UpdateMetaMessageText(string(msgID), responseText)
		}
	} else {

		outMsg := &waProto.Message{
			ExtendedTextMessage: &waProto.ExtendedTextMessage{
				Text: proto.String(responseText),
				ContextInfo: &waProto.ContextInfo{
					StanzaID:    proto.String(string(reply.msgID)),
					Participant: proto.String(reply.senderJID),
					QuotedMessage: &waProto.Message{
						Conversation: proto.String(""),
					},
				},
			},
		}
		if resp, err := client.SendMessage(context.Background(), targetJID, outMsg); err == nil {
			sentMessageIDs[resID] = resp.ID
			lastProcessedResponse[resID] = responseText
			db.SaveMetaMessage(string(resp.ID), targetJID.String(), responseText)
		}
	}
}

func handleMetaAutoReply(client *whatsmeow.Client, evt *events.Message) {

	ext := evt.Message.GetExtendedTextMessage()
	if ext == nil {
		return
	}
	ci := ext.GetContextInfo()
	if ci == nil {
		return
	}
	stanzaID := ci.GetStanzaID()
	if stanzaID == "" {
		return
	}

	if BotSettings.GetMode() == ModePrivate && !BotSettings.IsSudo(evt.Info.Sender.User) {
		return
	}

	pastResponse, found := db.GetMetaMessageText(stanzaID)
	if !found {
		return
	}

	replyText := ext.GetText()
	if replyText == "" {
		return
	}

	for _, p := range BotSettings.GetPrefixes() {
		if p != "" && strings.HasPrefix(strings.ToLower(replyText), strings.ToLower(p)) {
			return
		}
	}

	query := buildMetaQuery(evt.Info.Sender.User, senderPushName(evt), pastResponse, replyText)
	if _, err := client.SendMessage(context.Background(), MetaJID, &waProto.Message{
		Conversation: proto.String(query),
	}); err != nil {
		return
	}

	metaMu.Lock()
	pendingReplies[MetaJID.String()] = metaPendingReply{
		chatJID:   evt.Info.Chat,
		msgID:     evt.Info.ID,
		senderJID: evt.Info.Sender.String(),
	}
	metaMu.Unlock()
}

func init() {
	RegisterModerationHook(handleMetaAutoReply)

	Register(&Command{
		Pattern:  "meta",
		Category: "ai",
		Func: func(ctx *Context) error {
			query := ctx.Text
			senderID := ctx.Msg.Sender.User
			pushName := ctx.Msg.PushName
			if pushName == "" {
				pushName = ctx.Msg.Sender.User
			}

			var outMsg *waProto.Message
			nonTextQuoted := false

			ci := ctx.Msg.RawMessage.GetExtendedTextMessage().GetContextInfo()
			if ci != nil {
				quoted := ci.GetQuotedMessage()
				stanzaID := ci.GetStanzaID()

				if quoted != nil {

					if stanzaID != "" {
						if pastResponse, found := db.GetMetaMessageText(stanzaID); found {
							if query == "" {
								ctx.Reply(T().MetaUsage)
								return nil
							}
							q := buildMetaQuery(senderID, pushName, pastResponse, query)
							outMsg = &waProto.Message{Conversation: proto.String(q)}
						}
					}

					if outMsg == nil {
						quotedText := quoted.GetConversation()
						if quotedText == "" {
							quotedText = quoted.GetExtendedTextMessage().GetText()
						}
						if quotedText != "" {
							participant := ci.GetParticipant()
							quotedUserID, quotedUserName := resolveParticipant(participant)
							var q string
							if query == "" {
								q = fmt.Sprintf(
									"User ID: %s, Name: %s — Context: they quoted a message from User ID: %s, Name: %s, who said: %q",
									senderID, pushName, quotedUserID, quotedUserName, quotedText,
								)
							} else {
								q = fmt.Sprintf(
									"User ID: %s, Name: %s, Query: %s — Context: they quoted a message from User ID: %s, Name: %s, who said: %q",
									senderID, pushName, query, quotedUserID, quotedUserName, quotedText,
								)
							}
							outMsg = &waProto.Message{Conversation: proto.String(q)}
						} else {

							nonTextQuoted = true
							if query == "" {
								ctx.Reply(T().MetaUsage)
								return nil
							}
						}
					}
				}
			}

			if outMsg == nil {
				if img := ctx.Msg.RawMessage.GetImageMessage(); img != nil {
					img.Caption = proto.String(query)
					outMsg = &waProto.Message{ImageMessage: img}
				} else if vid := ctx.Msg.RawMessage.GetVideoMessage(); vid != nil {
					vid.Caption = proto.String(query)
					outMsg = &waProto.Message{VideoMessage: vid}
				}
			}

			if outMsg == nil {
				if query == "" {
					ctx.Reply(T().MetaUsage)
					return nil
				}
				q := buildMetaQuery(senderID, pushName, "", query)
				outMsg = &waProto.Message{Conversation: proto.String(q)}
			}

			if nonTextQuoted {
				ctx.Reply(T().MetaNonTextWarning)
			}

			resp, err := ctx.Client.SendMessage(context.Background(), MetaJID, outMsg)
			if err != nil {
				return err
			}
			_ = resp

			metaMu.Lock()
			pendingReplies[MetaJID.String()] = metaPendingReply{
				chatJID:   ctx.Msg.Chat,
				msgID:     ctx.Msg.ID,
				senderJID: ctx.Msg.Sender.String(),
			}
			metaMu.Unlock()
			return nil
		},
	})
}
