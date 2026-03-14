package plugins

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type Command struct {
	Pattern  string
	Aliases  []string
	IsSudo   bool
	IsAdmin  bool
	IsGroup  bool
	Category string
	Func     func(ctx *Context) error
}

type Context struct {
	Client     *whatsmeow.Client
	Msg        *Message
	Args       []string
	Text       string
	Prefix     string
	Matched    string
	ReceivedAt time.Time
}

// Reply sends a text reply to the chat this message came from.
func (c *Context) Reply(text string) (whatsmeow.SendResponse, error) {
	id := c.Client.GenerateMessageID()
	sendQueue <- sendTask{
		client:   c.Client,
		to:       c.Msg.Chat,
		msg:      &waProto.Message{Conversation: proto.String(text)},
		id:       id,
		queuedAt: time.Now(),
	}
	return whatsmeow.SendResponse{ID: id}, nil
}

// ReplySync sends a text reply synchronously.
func (c *Context) ReplySync(text string) (whatsmeow.SendResponse, error) {
	return c.Client.SendMessage(context.Background(), c.Msg.Chat,
		&waProto.Message{Conversation: proto.String(text)},
		whatsmeow.SendRequestExtra{Timeout: sendTimeout},
	)
}

// QueueEdit edits a previously sent message.
func (c *Context) QueueEdit(originalID types.MessageID, newText string) {
	sendQueue <- sendTask{
		client: c.Client,
		to:     c.Msg.Chat,
		msg: c.Client.BuildEdit(c.Msg.Chat, originalID, &waProto.Message{
			Conversation: proto.String(newText),
		}),
		id:       c.Client.GenerateMessageID(),
		queuedAt: time.Now(),
	}
}

// InteractiveSessions stores callbacks for message replies. Key is the StanzaID.
var InteractiveSessions sync.Map

// SendLoader sends a loading message and updates it continuously until the returned stop function is called.
func (c *Context) SendLoader() (string, func(string)) {
	resp, err := c.ReplySync(fmt.Sprintf(T().LoaderProcessing, "⠋"))
	if err != nil {
		return "", func(string) {}
	}
	id := resp.ID
	stopCh := make(chan string, 1)

	go func() {
		ticker := time.NewTicker(800 * time.Millisecond)
		defer ticker.Stop()
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case finalMsg := <-stopCh:
				if finalMsg != "" {
					c.QueueEdit(id, finalMsg)
				} else {
					c.QueueEdit(id, "✅")
				}
				return
			case <-ticker.C:
				i = (i + 1) % len(frames)
				msg := fmt.Sprintf(T().LoaderProcessing, frames[i])
				c.QueueEdit(id, msg)
			}
		}
	}()

	return id, func(final string) {
		stopCh <- final
	}
}

// Quoted returns information about the quoted/replied-to message, or nil.
func (c *Context) Quoted() *QuotedInfo {
	return c.Msg.Quoted()
}

// SendText queues a text message to the current chat.
func (c *Context) SendText(text string) {
	queueMsg(c.Client, c.Msg.Chat, &waProto.Message{Conversation: proto.String(text)})
}

// SendImage uploads image data and queues the image message.
func (c *Context) SendImage(data []byte, mime, caption string) error {
	resp, err := c.Client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("upload image: %w", err)
	}
	msg := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Mimetype:      proto.String(mime),
			Caption:       proto.String(caption),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(resp.FileLength),
		},
	}
	queueMsg(c.Client, c.Msg.Chat, msg)
	return nil
}

// SendVideo uploads video data and queues the video message.
func (c *Context) SendVideo(data []byte, mime, caption string) error {
	resp, err := c.Client.Upload(context.Background(), data, whatsmeow.MediaVideo)
	if err != nil {
		return fmt.Errorf("upload video: %w", err)
	}
	msg := &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			Mimetype:      proto.String(mime),
			Caption:       proto.String(caption),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(resp.FileLength),
		},
	}
	queueMsg(c.Client, c.Msg.Chat, msg)
	return nil
}

// SendMentionSync sends a text message with mentions synchronously, bypassing the send queue.
// This is useful when an action (like kicking/blocking) must happen strictly after the message is sent.
func (c *Context) SendMentionSync(text string, jids []string) (whatsmeow.SendResponse, error) {
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: jids,
			},
		},
	}
	return c.Client.SendMessage(context.Background(), c.Msg.Chat, msg)
}

// SendMention queues a text message with mentions.
func (c *Context) SendMention(text string, jids []string) {
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: jids,
			},
		},
	}
	queueMsg(c.Client, c.Msg.Chat, msg)
}

// SendAudio uploads audio data and queues the audio message.
func (c *Context) SendAudio(data []byte, mime string) error {
	resp, err := c.Client.Upload(context.Background(), data, whatsmeow.MediaAudio)
	if err != nil {
		return fmt.Errorf("upload audio: %w", err)
	}
	msg := &waProto.Message{
		AudioMessage: &waProto.AudioMessage{
			Mimetype:      proto.String(mime),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(resp.FileLength),
		},
	}
	queueMsg(c.Client, c.Msg.Chat, msg)
	return nil
}

// SendSticker uploads sticker data and queues the sticker message.
func (c *Context) SendSticker(data []byte, mime string) error {
	resp, err := c.Client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("upload sticker: %w", err)
	}
	msg := &waProto.Message{
		StickerMessage: &waProto.StickerMessage{
			Mimetype:      proto.String(mime),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(resp.FileLength),
		},
	}
	queueMsg(c.Client, c.Msg.Chat, msg)
	return nil
}

// SendRawMsg queues an arbitrary protobuf message to the current chat.
func (c *Context) SendRawMsg(msg *waProto.Message) {
	queueMsg(c.Client, c.Msg.Chat, msg)
}

var registry []*Command

var registryMap = make(map[string]*Command)

var categoryMap = make(map[string][]*Command)

func Register(cmd *Command) {
	registry = append(registry, cmd)
	registryMap[strings.ToLower(cmd.Pattern)] = cmd
	for _, alias := range cmd.Aliases {
		registryMap[strings.ToLower(alias)] = cmd
	}
	cat := strings.ToLower(cmd.Category)
	if cat == "" {
		cat = "general"
	}
	categoryMap[cat] = append(categoryMap[cat], cmd)
}

func parseCommand(text string, prefixes []string) (prefix, name, rest string, ok bool) {
	lower := strings.ToLower(text)
	for _, p := range prefixes {
		var afterOrig, afterLower string
		if p == "" {
			afterOrig = text
			afterLower = lower
		} else {
			lp := strings.ToLower(p)
			if !strings.HasPrefix(lower, lp) {
				continue
			}
			afterOrig = text[len(lp):]
			afterLower = lower[len(lp):]
		}
		afterLower = strings.TrimLeft(afterLower, " ")
		if afterLower == "" {
			continue
		}

		trimmed := len(afterOrig) - len(strings.TrimLeft(afterOrig, " "))
		afterOrig = afterOrig[trimmed:]
		if i := strings.IndexByte(afterLower, ' '); i != -1 {
			name = afterLower[:i]
			rest = strings.TrimSpace(afterOrig[i+1:])
		} else {
			name = afterLower
		}
		return p, name, rest, true
	}
	return "", "", "", false
}

func findCommand(name string) *Command {
	return registryMap[name]
}

// extractText extracts user-visible text from an events.Message.
// Kept for backward compat with moderation hooks that receive raw events.
func extractText(evt *events.Message) string {
	return extractMsgTextFromProto(evt.Message)
}

func Dispatch(client *whatsmeow.Client, evt *events.Message) {
	receivedAt := time.Now()
	msg := NewMessage(evt)

	if msg.Text == "" {
		return
	}

	senderID := evt.Info.Sender.User
	chatServer := evt.Info.Chat.Server

	if chatServer == types.BroadcastServer || chatServer == types.NewsletterServer {
		return
	}

	if q := msg.Quoted(); q != nil {
		if handlerAny, ok := InteractiveSessions.Load(q.StanzaID); ok {
			handler := handlerAny.(func(*Context))
			ctx := &Context{
				Client:     client,
				Msg:        msg,
				Args:       strings.Fields(msg.Text),
				Text:       msg.Text,
				Prefix:     "",
				Matched:    "",
				ReceivedAt: receivedAt,
			}
			handler(ctx)
			return
		}
	}

	if msg.IsGroup && BotSettings.IsGCDisabled() {
		return
	}

	prefix, name, rest, ok := parseCommand(msg.Text, BotSettings.GetPrefixes())
	if !ok {
		return
	}

	cmd := findCommand(name)
	if cmd == nil {

		if menu := CategoryMenu(name); menu != "" {
			miniCtx := &Context{Client: client, Msg: msg}
			miniCtx.Reply(menu)
		}
		return
	}

	ctx := &Context{
		Client:     client,
		Msg:        msg,
		Args:       strings.Fields(rest),
		Text:       rest,
		Prefix:     prefix,
		Matched:    name,
		ReceivedAt: receivedAt,
	}

	isSudo := BotSettings.IsSudo(senderID)

	if !isSudo && evt.Info.SenderAlt.User != "" {
		isSudo = BotSettings.IsSudo(evt.Info.SenderAlt.User)
	}

	isBanned := BotSettings.IsBanned(senderID)
	if !isBanned && evt.Info.SenderAlt.User != "" {
		isBanned = BotSettings.IsBanned(evt.Info.SenderAlt.User)
	}
	if isBanned {
		return
	}
	mode := BotSettings.GetMode()

	if mode == ModePrivate && !isSudo {
		return
	}

	if cmd.IsGroup && !msg.IsGroup {
		ctx.Reply(T().GroupOnly)
		return
	}

	if cmd.IsSudo && !isSudo {
		ctx.Reply(T().SudoOnly)
		return
	}

	if cmd.IsAdmin && msg.IsGroup {
		botJID := client.Store.ID.ToNonAD()
		group, err := client.GetGroupInfo(context.Background(), evt.Info.Chat)
		if err == nil {
			if !botIsAdmin(group.Participants, ownerPhone, botJID.User) {
				ctx.Reply(T().BotNotAdmin)
				return
			}
			if !isSudo {
				p := findParticipant(group.Participants, evt.Info.Sender.User, evt.Info.SenderAlt.User)
				if p == nil || (!p.IsAdmin && !p.IsSuperAdmin) {
					ctx.Reply(T().SenderNotAdmin)
					return
				}
			}
		}
	}

	if BotSettings.IsCmdDisabled(name) {
		ctx.Reply(T().CmdIsDisabled)
		return
	}

	_ = cmd.Func(ctx)
}
