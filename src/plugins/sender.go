package plugins

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

const sendTimeout = 20 * time.Second

type sendTask struct {
	client   *whatsmeow.Client
	to       types.JID
	msg      *waProto.Message
	id       types.MessageID
	queuedAt time.Time
}

var sendQueue = make(chan sendTask, 512)

// Concurrency=2 pipelines encrypt+network without SQLite lock contention.
var sendSem = make(chan struct{}, 2)

func init() {
	go sendWorker()
}

func sendWorker() {
	for task := range sendQueue {
		sendSem <- struct{}{}
		go func(t sendTask) {
			defer func() { <-sendSem }()
			_, err := t.client.SendMessage(
				context.Background(),
				t.to,
				t.msg,
				whatsmeow.SendRequestExtra{
					ID:      t.id,
					Timeout: sendTimeout,
				},
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[Send ERROR] %s → %s: %v\n", t.id, t.to, err)
			}
		}(task)
	}
}

func sendMention(ctx *Context, text string, jids []string) {
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: jids,
			},
		},
	}
	queueMsg(ctx.Client, ctx.Msg.Chat, msg)
}

// queueMsg is a shorthand to enqueue a message for sending.
func queueMsg(client *whatsmeow.Client, to types.JID, msg *waProto.Message) {
	sendQueue <- sendTask{
		client:   client,
		to:       to,
		msg:      msg,
		id:       client.GenerateMessageID(),
		queuedAt: time.Now(),
	}
}
