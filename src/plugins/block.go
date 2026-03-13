package plugins

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

func init() {
	Register(&Command{
		Pattern:  "block",
		Category: "utility",
		Func: func(ctx *Context) error {
			target := ResolveTargetUser(ctx)
			if target == nil {
				ctx.Reply(T().BlockUsage)
				return nil
			}
			ctx.SendMentionSync(fmt.Sprintf("@%s Blocked", target.User), []string{target.String()})
			if err := blockUser(ctx.Client, *target); err != nil {
				log.Printf("[block] ERROR: %v", err)
				ctx.Reply(fmt.Sprintf("Block failed: %v", err))
				return nil
			}
			return nil
		},
	})

	Register(&Command{
		Pattern:  "unblock",
		Category: "utility",
		Func: func(ctx *Context) error {
			target := ResolveTargetUser(ctx)
			if target == nil {
				ctx.Reply(T().UnblockUsage)
				return nil
			}
			ctx.SendMentionSync(fmt.Sprintf("@%s unblocked", target.User), []string{target.String()})
			if err := unblockUser(ctx.Client, *target); err != nil {
				log.Printf("[unblock] ERROR: %v", err)
				ctx.Reply(fmt.Sprintf("Unblock failed: %v", err))
				return nil
			}
			return nil
		},
	})
}

func blockUser(client *whatsmeow.Client, target types.JID) error {
	lid, pn := resolveBothJIDs(target)

	attrs := waBinary.Attrs{
		"action": "block",
		"jid":    lid,
	}
	if pn != "" {
		attrs["pn_jid"] = pn
	}

	return sendBlocklistIQ(client, attrs)
}

func unblockUser(client *whatsmeow.Client, target types.JID) error {
	lid, _ := resolveBothJIDs(target)

	attrs := waBinary.Attrs{
		"action": "unblock",
		"jid":    lid,
	}

	return sendBlocklistIQ(client, attrs)
}

func sendBlocklistIQ(client *whatsmeow.Client, attrs waBinary.Attrs) error {
	log.Printf("[blocklist] %v", attrs)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := client.DangerousInternals().SendIQ(ctx, whatsmeow.DangerousInfoQuery{
		Namespace: "blocklist",
		Type:      "set",
		To:        types.ServerJID,
		Content:   []waBinary.Node{{Tag: "item", Attrs: attrs}},
	})
	if err != nil {
		return fmt.Errorf("blocklist IQ: %w", err)
	}
	return nil
}

func resolveBothJIDs(target types.JID) (lid, pn string) {
	if target.Server == types.HiddenUserServer {
		lid = target.String()
		if alt := GetAltID(target.String()); alt != "" {
			pn = alt + "@s.whatsapp.net"
		}
		return
	}
	pn = target.String()
	if alt := GetAltID(target.String()); alt != "" {
		lid = alt + "@lid"
	} else {
		lid = target.String()
	}
	return
}
