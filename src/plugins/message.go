package plugins

import (
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// Message is a lightweight wrapper around a WhatsApp message event.
// It pre-extracts the fields that plugins need, avoiding repeated
// traversal of the heavy protobuf tree.
type Message struct {
	ID         types.MessageID
	Chat       types.JID
	Sender     types.JID
	SenderAlt  types.JID
	IsGroup    bool
	IsFromMe   bool
	Timestamp  time.Time
	Text       string
	PushName   string
	RawMessage *waProto.Message // kept for cache/forward needs
}

// QuotedInfo holds information about a quoted (replied-to) message.
type QuotedInfo struct {
	StanzaID    string
	Participant string
	Message     *waProto.Message
}

// NewMessage constructs a lightweight Message from a raw event.
func NewMessage(evt *events.Message) *Message {
	m := &Message{
		ID:         evt.Info.ID,
		Chat:       evt.Info.Chat,
		Sender:     evt.Info.Sender,
		SenderAlt:  evt.Info.SenderAlt,
		IsGroup:    evt.Info.Chat.Server == types.GroupServer,
		IsFromMe:   evt.Info.IsFromMe,
		Timestamp:  evt.Info.Timestamp,
		RawMessage: evt.Message,
	}
	m.Text = extractMsgTextFromProto(evt.Message)
	m.PushName = evt.Info.PushName
	return m
}

// Quoted returns information about the quoted message, or nil if there is none.
func (m *Message) Quoted() *QuotedInfo {
	ci := m.RawMessage.GetExtendedTextMessage().GetContextInfo()
	if ci == nil {
		return nil
	}
	qm := ci.GetQuotedMessage()
	if qm == nil {
		return nil
	}
	return &QuotedInfo{
		StanzaID:    ci.GetStanzaID(),
		Participant: ci.GetParticipant(),
		Message:     qm,
	}
}

// QuotedContextInfo returns the raw ContextInfo from the message.
// Useful for pin/star/del commands that need stanza ID and participant
// even when there is no quoted message body.
func (m *Message) QuotedContextInfo() (stanzaID, participant string) {
	ci := m.RawMessage.GetExtendedTextMessage().GetContextInfo()
	if ci == nil {
		return "", ""
	}
	return ci.GetStanzaID(), ci.GetParticipant()
}

// extractMsgTextFromProto extracts the user-visible text from a raw protobuf message.
func extractMsgTextFromProto(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}
	if t := msg.GetConversation(); t != "" {
		return t
	}
	if t := msg.GetExtendedTextMessage().GetText(); t != "" {
		return t
	}
	if t := msg.GetImageMessage().GetCaption(); t != "" {
		return t
	}
	if t := msg.GetVideoMessage().GetCaption(); t != "" {
		return t
	}
	return ""
}

// SenderUser returns the User part of the sender JID,
// stripping the device suffix if present.
func (m *Message) SenderUser() string {
	u := m.Sender.User
	if i := strings.IndexByte(u, '.'); i != -1 {
		return u[:i]
	}
	return u
}
