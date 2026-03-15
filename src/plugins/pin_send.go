package plugins

import (
	"context"
	"fmt"
	"reflect"
	"unsafe"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

//go:linkname encryptMessageForDevices go.mau.fi/whatsmeow.(*Client).encryptMessageForDevices
func encryptMessageForDevices(cli *whatsmeow.Client, ctx context.Context, allDevices []types.JID, id string, msgPlaintext, dsmPlaintext []byte, extraAttrs waBinary.Attrs) ([]waBinary.Node, bool, error)

//go:linkname makeDeviceIdentityNode go.mau.fi/whatsmeow.(*Client).makeDeviceIdentityNode
func makeDeviceIdentityNode(cli *whatsmeow.Client) waBinary.Node

func sendPinMessage(ctx context.Context, cli *whatsmeow.Client, to types.JID, msg *waE2E.Message) error {
	// 1. Prepare plaintext (replicated from whatsmeow/send.go marshalMessage)
	plaintext, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	var dsmPlaintext []byte
	if to.Server != types.GroupServer && to.Server != types.NewsletterServer {
		dsmPlaintext, err = proto.Marshal(&waE2E.Message{
			DeviceSentMessage: &waE2E.DeviceSentMessage{
				DestinationJID: proto.String(to.String()),
				Message:        msg,
			},
			MessageContextInfo: msg.MessageContextInfo,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal message (for own devices): %w", err)
		}
	}

	// 2. Get devices for encryption
	devices, err := cli.GetUserDevicesContext(ctx, []types.JID{to})
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	msgID := cli.GenerateMessageID()
	encAttrs := waBinary.Attrs{}

	// 3. Call internal encryption via go:linkname
	participantNodes, includeIdentity, err := encryptMessageForDevices(cli, ctx, devices, msgID, plaintext, dsmPlaintext, encAttrs)
	if err != nil {
		return fmt.Errorf("internal encryption failed: %w", err)
	}

	// 4. Construct the message content (replicated from whatsmeow/send.go getMessageContent)
	participantNode := waBinary.Node{
		Tag:     "participants",
		Content: participantNodes,
	}

	content := []waBinary.Node{participantNode}
	if includeIdentity {
		content = append(content, makeDeviceIdentityNode(cli))
	}

	// 5. Construct the final message node
	node := waBinary.Node{
		Tag: "message",
		Attrs: waBinary.Attrs{
			"id":   msgID,
			"type": "text", // Pins are sent as "text" type nodes with edit="2"
			"to":   to,
			"edit": "2",
		},
		Content: content,
	}

	// 6. Send the node directly via internal socket (access via unsafe reflection)
	vCli := reflect.ValueOf(cli).Elem()
	fSocket := vCli.FieldByName("socket")
	if !fSocket.IsValid() {
		return fmt.Errorf("could not find unexported socket field in Client")
	}

	// Bypass unexported field restriction
	ptrSocket := reflect.NewAt(fSocket.Type(), unsafe.Pointer(fSocket.UnsafeAddr())).Elem()

	payload, err := waBinary.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node: %w", err)
	}

	mSendFrame := ptrSocket.MethodByName("SendFrame")
	if !mSendFrame.IsValid() {
		return fmt.Errorf("could not find SendFrame method on socket")
	}

	results := mSendFrame.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(payload),
	})

	if !results[0].IsNil() {
		return results[0].Interface().(error)
	}

	return nil
}
