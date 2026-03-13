package plugins

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type LIDResolver interface {
	GetLIDForPN(ctx context.Context, pn types.JID) (types.JID, error)
	GetPNForLID(ctx context.Context, lid types.JID) (types.JID, error)
	PutLIDMapping(ctx context.Context, lid, pn types.JID) error
}

var lidResolver LIDResolver
var ownerPhone string

func InitLIDStore(ls LIDResolver, ownerPN string) {
	lidResolver = ls
	ownerPhone = ownerPN
}

func GetAltID(id string) string {
	if lidResolver == nil {
		return ""
	}
	ctx := context.Background()

	var jid types.JID
	if strings.Contains(id, "@") {
		parsed, err := types.ParseJID(id)
		if err != nil {
			return ""
		}
		jid = parsed
	} else {

		jid = types.NewJID(id, types.DefaultUserServer)
	}

	switch jid.Server {
	case types.DefaultUserServer:
		lid, err := lidResolver.GetLIDForPN(ctx, jid)
		if err != nil || lid.User == "" {
			return ""
		}
		return lid.User
	case types.HiddenUserServer:
		pn, err := lidResolver.GetPNForLID(ctx, jid)
		if err != nil || pn.User == "" {
			return ""
		}
		return pn.User
	}
	return ""
}

func SaveUser(evt *events.Message) {
	if lidResolver == nil {
		return
	}

	ctx := context.Background()
	sender := evt.Info.Sender

	if sender.Server != types.HiddenUserServer {
		return
	}
	senderLID := types.NewJID(sender.User, types.HiddenUserServer)

	if evt.Info.SenderAlt.User != "" && evt.Info.SenderAlt.Server == types.DefaultUserServer {
		pnJID := types.NewJID(evt.Info.SenderAlt.User, types.DefaultUserServer)
		_ = lidResolver.PutLIDMapping(ctx, senderLID, pnJID)
	} else if evt.Info.IsFromMe && ownerPhone != "" {

		pnJID := types.NewJID(ownerPhone, types.DefaultUserServer)
		_ = lidResolver.PutLIDMapping(ctx, senderLID, pnJID)
	}

	if evt.Info.IsFromMe && !evt.Info.IsGroup &&
		evt.Info.Chat.Server == types.HiddenUserServer &&
		evt.Info.RecipientAlt.User != "" && evt.Info.RecipientAlt.Server == types.DefaultUserServer {
		recipLID := types.NewJID(evt.Info.Chat.User, types.HiddenUserServer)
		recipPN := types.NewJID(evt.Info.RecipientAlt.User, types.DefaultUserServer)
		_ = lidResolver.PutLIDMapping(ctx, recipLID, recipPN)
	}
}

func BootstrapOwnerSudoers() {
	if ownerPhone == "" {
		return
	}
	changed := false

	if !BotSettings.IsSudo(ownerPhone) {
		BotSettings.AddSudo(ownerPhone)
		changed = true
	}

	if lid := GetAltID(ownerPhone); lid != "" && !BotSettings.IsSudo(lid) {
		BotSettings.AddSudo(lid)
		changed = true
	}

	if changed {
		_ = SaveSettings()
	}
}

func ResolveTarget(ctx *Context, arg string) (phone, lid string) {

	if arg == "" || strings.EqualFold(arg, "reply") {
		participant := ctx.Msg.RawMessage.GetExtendedTextMessage().GetContextInfo().GetParticipant()
		if participant != "" {
			return resolveJIDString(participant)
		}
		if arg != "" {

			return "", ""
		}
	}

	arg = strings.TrimPrefix(arg, "@")

	return resolveJIDString(arg)
}

// ResolveTargetUser resolves a target user JID from context + args.
//
// Resolution order:
//  1. If args contain a phone/ID → clean it, look up in LID map, build JID
//  2. If the message is a reply → use the quoted message's participant
//  3. If the message has mentions → use the first mentioned JID
//  4. In DM chat → use the other user (chat JID)
//  5. Otherwise → nil (no user found)
func ResolveTargetUser(ctx *Context) *types.JID {
	// 1. Explicit arg (highest priority)
	if len(ctx.Args) > 0 {
		raw := ctx.Args[0]
		// Strip common prefixes/symbols
		raw = strings.TrimPrefix(raw, "@")
		raw = strings.TrimPrefix(raw, "+")
		raw = strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1 // drop non-digits
		}, raw)
		if raw != "" {
			return buildUserJID(raw)
		}
	}

	// 2. Replied message → quoted participant
	if q := ctx.Msg.Quoted(); q != nil && q.Participant != "" {
		return parseUserJID(q.Participant)
	}

	// 3. Mentioned users in the message
	ci := ctx.Msg.RawMessage.GetExtendedTextMessage().GetContextInfo()
	if ci != nil {
		if mentioned := ci.GetMentionedJID(); len(mentioned) > 0 {
			return parseUserJID(mentioned[0])
		}
	}

	// 4. DM chat → the other user
	if !ctx.Msg.IsGroup {
		jid := ctx.Msg.Chat.ToNonAD()
		return &jid
	}

	return nil
}

// buildUserJID takes a raw phone/user string, checks both PN and LID
// directions in the LID map, and returns a JID with the correct server.
func buildUserJID(raw string) *types.JID {
	// Try as phone number first: does this PN have a known LID?
	pnJID := types.NewJID(raw, types.DefaultUserServer)
	if altLID := GetAltID(pnJID.String()); altLID != "" {
		return &pnJID
	}

	// Try as LID: does this LID have a known PN?
	lidJID := types.NewJID(raw, types.HiddenUserServer)
	if altPN := GetAltID(lidJID.String()); altPN != "" {
		return &lidJID
	}

	// Unknown — default to phone number
	return &pnJID
}

// parseUserJID parses a JID string like "12345@s.whatsapp.net" into a types.JID.
func parseUserJID(s string) *types.JID {
	if s == "" {
		return nil
	}
	parsed, err := types.ParseJID(s)
	if err != nil {
		return nil
	}
	parsed.Device = 0
	return &parsed
}

func resolveJIDString(s string) (phone, lid string) {
	if s == "" {
		return "", ""
	}

	var jid types.JID
	if strings.Contains(s, "@") {
		parsed, err := types.ParseJID(s)
		if err != nil {
			return "", ""
		}

		parsed.Device = 0
		jid = parsed
	} else {

		s = strings.TrimPrefix(s, "+")
		jid = types.NewJID(s, types.DefaultUserServer)
	}

	switch jid.Server {
	case types.DefaultUserServer:
		phone = jid.User
		lid = GetAltID(jid.String())
	case types.HiddenUserServer:
		lid = jid.User
		phone = GetAltID(jid.String())
	}
	return
}
