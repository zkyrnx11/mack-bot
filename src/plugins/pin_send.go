package plugins

import (
	"bytes"
	"context"
	"fmt"

	"go.mau.fi/libsignal/groups"
	"go.mau.fi/libsignal/protocol"
	"go.mau.fi/libsignal/session"
	"go.mau.fi/util/random"
	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func sendPinMessage(ctx context.Context, cli *whatsmeow.Client, to types.JID, msg *waE2E.Message) error {
	id := cli.GenerateMessageID()

	plaintext, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	dsmPlaintext, err := proto.Marshal(&waE2E.Message{
		DeviceSentMessage: &waE2E.DeviceSentMessage{
			DestinationJID: proto.String(to.String()),
			Message:        msg,
		},
	})
	if err != nil {
		return fmt.Errorf("marshal dsm: %w", err)
	}

	internals := cli.DangerousInternals()
	ownJID := internals.GetOwnID()
	ownLID := internals.GetOwnLID()
	pbSer := store.SignalProtobufSerializer

	perDevicePlaintext := plaintext
	var groupEncNode *waBinary.Node
	var participantJIDs []types.JID

	if to.Server == types.GroupServer {
		senderJID := ownLID
		if senderJID.IsEmpty() {
			senderJID = *cli.Store.ID
		}
		senderKeyName := protocol.NewSenderKeyName(to.String(), senderJID.SignalAddress())
		groupBuilder := groups.NewGroupSessionBuilder(cli.Store, pbSer)
		skdMsg, err := groupBuilder.Create(ctx, senderKeyName)
		if err != nil {
			return fmt.Errorf("create sender key distribution: %w", err)
		}
		skdPlaintext, err := proto.Marshal(&waE2E.Message{
			SenderKeyDistributionMessage: &waE2E.SenderKeyDistributionMessage{
				GroupID:                             proto.String(to.String()),
				AxolotlSenderKeyDistributionMessage: skdMsg.Serialize(),
			},
		})
		if err != nil {
			return fmt.Errorf("marshal skd message: %w", err)
		}
		perDevicePlaintext = skdPlaintext

		groupCipher := groups.NewGroupCipher(groupBuilder, senderKeyName, cli.Store)
		encrypted, err := groupCipher.Encrypt(ctx, padPinMessage(plaintext))
		if err != nil {
			return fmt.Errorf("group encrypt: %w", err)
		}
		skMsg := waBinary.Node{
			Tag:     "enc",
			Content: encrypted.SignedSerialize(),
			Attrs:   waBinary.Attrs{"v": "2", "type": "skmsg"},
		}
		groupEncNode = &skMsg

		groupInfo, err := cli.GetGroupInfo(ctx, to)
		if err != nil {
			return fmt.Errorf("get group info: %w", err)
		}
		for _, p := range groupInfo.Participants {
			participantJIDs = append(participantJIDs, p.JID)
		}
	} else {
		participantJIDs = []types.JID{to, ownJID.ToNonAD()}
	}

	allDevices, err := cli.GetUserDevices(ctx, participantJIDs)
	if err != nil {
		return fmt.Errorf("get user devices: %w", err)
	}

	var participantNodes []waBinary.Node
	for _, jid := range allDevices {
		isOwn := jid.User == ownJID.User || (!ownLID.IsEmpty() && jid.User == ownLID.User)

		devicePlaintext := perDevicePlaintext
		if to.Server != types.GroupServer && isOwn {
			if jid.ToNonAD() == ownJID.ToNonAD() {
				continue
			}
			devicePlaintext = dsmPlaintext
		}

		addr := jid.SignalAddress()
		hasSession, err := cli.Store.ContainsSession(ctx, addr)
		if err != nil || !hasSession {
			continue
		}

		builder := session.NewBuilderFromSignal(cli.Store, addr, pbSer)
		cipher := session.NewCipher(builder, addr)
		ciphertext, err := cipher.Encrypt(ctx, padPinMessage(devicePlaintext))
		if err != nil {
			continue
		}

		encType := "msg"
		if ciphertext.Type() == protocol.PREKEY_TYPE {
			encType = "pkmsg"
		}
		participantNodes = append(participantNodes, waBinary.Node{
			Tag:   "to",
			Attrs: waBinary.Attrs{"jid": jid},
			Content: []waBinary.Node{{
				Tag:     "enc",
				Attrs:   waBinary.Attrs{"v": "2", "type": encType},
				Content: ciphertext.Serialize(),
			}},
		})
	}

	content := []waBinary.Node{{Tag: "participants", Content: participantNodes}}
	if groupEncNode != nil {
		content = append(content, *groupEncNode)
	}

	_, err = internals.SendNodeAndGetData(ctx, waBinary.Node{
		Tag: "message",
		Attrs: waBinary.Attrs{
			"id":   id,
			"type": "text",
			"to":   to,
			"edit": string(types.EditAttributePinInChat),
		},
		Content: content,
	})
	return err
}

func padPinMessage(plaintext []byte) []byte {
	pad := random.Bytes(1)
	pad[0] &= 0xf
	if pad[0] == 0 {
		pad[0] = 0xf
	}
	plaintext = append(plaintext, bytes.Repeat(pad, int(pad[0]))...)
	return plaintext
}
