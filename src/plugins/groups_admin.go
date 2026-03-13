package plugins

import (
	"context"
	"fmt"
	"strings"

	db "github.com/zkyrnx11/mack/src/sql"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func findParticipant(participants []types.GroupParticipant, phone, lid string) *types.GroupParticipant {
	for i := range participants {
		p := &participants[i]
		if phone != "" && (p.JID.User == phone || p.PhoneNumber.User == phone) {
			return p
		}
		if lid != "" && (p.JID.User == lid || p.LID.User == lid) {
			return p
		}
	}
	return nil
}

func botIsAdmin(participants []types.GroupParticipant, phone, user string) bool {
	p := findParticipant(participants, phone, user)
	if p == nil {
		return false
	}
	return p.IsAdmin || p.IsSuperAdmin
}

func init() {
	Register(&Command{
		Pattern:  "promote",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			arg := ctx.Text
			if arg == "" {
				ctx.Reply(menuHeader("promote") + T().PromoteUsage)
				return nil
			}
			phone, lid := ResolveTarget(ctx, arg)
			if phone == "" && lid == "" {
				ctx.Reply(T().UserResolveFail)
				return nil
			}
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Msg.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			p := findParticipant(group.Participants, phone, lid)
			if p == nil {
				ctx.Reply(T().UserNotFound)
				return nil
			}
			if p.IsAdmin || p.IsSuperAdmin {
				ctx.Reply(T().PromoteAlreadyAdmin)
				return nil
			}
			targetJID := p.JID.ToNonAD()
			_, err = ctx.Client.UpdateGroupParticipants(context.Background(), ctx.Msg.Chat,
				[]types.JID{targetJID}, whatsmeow.ParticipantChangePromote)
			if err != nil {
				return err
			}
			ctx.SendMention(fmt.Sprintf("@%s promoted", targetJID.User), []string{targetJID.String()})
			return nil
		},
	})

	Register(&Command{
		Pattern:  "demote",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			arg := ctx.Text
			if arg == "" {
				ctx.Reply(menuHeader("demote") + T().DemoteUsage)
				return nil
			}
			phone, lid := ResolveTarget(ctx, arg)
			if phone == "" && lid == "" {
				ctx.Reply(T().UserResolveFail)
				return nil
			}
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Msg.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			p := findParticipant(group.Participants, phone, lid)
			if p == nil {
				ctx.Reply(T().UserNotFound)
				return nil
			}
			if p.IsSuperAdmin {
				ctx.Reply(T().DemoteSuperAdmin)
				return nil
			}
			if !p.IsAdmin {
				ctx.Reply(T().DemoteNotAdmin)
				return nil
			}
			targetJID := p.JID.ToNonAD()
			_, err = ctx.Client.UpdateGroupParticipants(context.Background(), ctx.Msg.Chat,
				[]types.JID{targetJID}, whatsmeow.ParticipantChangeDemote)
			if err != nil {
				return err
			}
			ctx.SendMention(fmt.Sprintf("@%s demoted", targetJID.User), []string{targetJID.String()})
			return nil
		},
	})

	Register(&Command{
		Pattern:  "kick",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			arg := ctx.Text
			if arg == "" {
				ctx.Reply(menuHeader("kick") + T().KickUsage)
				return nil
			}
			phone, lid := ResolveTarget(ctx, arg)
			if phone == "" && lid == "" {
				ctx.Reply(T().UserResolveFail)
				return nil
			}
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Msg.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			p := findParticipant(group.Participants, phone, lid)
			if p == nil {
				ctx.Reply(T().UserNotFound)
				return nil
			}
			if p.IsSuperAdmin {
				ctx.Reply(T().KickSuperAdmin)
				return nil
			}
			targetJID := p.JID.ToNonAD()
			ctx.SendMentionSync(fmt.Sprintf("@%s kicked", targetJID.User), []string{targetJID.String()})
			_, err = ctx.Client.UpdateGroupParticipants(context.Background(), ctx.Msg.Chat,
				[]types.JID{targetJID}, whatsmeow.ParticipantChangeRemove)
			if err != nil {
				return err
			}
			return nil
		},
	})

	Register(&Command{
		Pattern:  "kickall",
		IsGroup:  true,
		IsAdmin:  true,
		IsSudo:   true,
		Category: "group",
		Func: func(ctx *Context) error {
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Msg.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			var toKick []types.JID
			for _, p := range group.Participants {
				if !p.IsSuperAdmin {
					toKick = append(toKick, p.JID.ToNonAD())
				}
			}
			ctx.Reply(fmt.Sprintf(T().KickAllStart, len(toKick)))
			for i := 0; i < len(toKick); i += 20 {
				end := i + 20
				if end > len(toKick) {
					end = len(toKick)
				}
				ctx.Client.UpdateGroupParticipants(context.Background(), ctx.Msg.Chat,
					toKick[i:end], whatsmeow.ParticipantChangeRemove)
			}
			ctx.Reply(T().KickAllDone)
			ctx.Client.LeaveGroup(context.Background(), ctx.Msg.Chat)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "mute",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Msg.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			if group.IsAnnounce {
				ctx.Reply(T().MuteAlready)
				return nil
			}
			if err := ctx.Client.SetGroupAnnounce(context.Background(), ctx.Msg.Chat, true); err != nil {
				return err
			}
			ctx.Reply(T().MuteOK)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "unmute",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Msg.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}
			if !group.IsAnnounce {
				ctx.Reply(T().UnmuteNotMuted)
				return nil
			}
			if err := ctx.Client.SetGroupAnnounce(context.Background(), ctx.Msg.Chat, false); err != nil {
				return err
			}
			ctx.Reply(T().UnmuteOK)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "messages",
		Category: "group",
		Func: func(ctx *Context) error {
			chats, err := db.GetTopChats()
			if err != nil {
				return err
			}

			var sb strings.Builder
			sb.WriteString(T().MessagesHeader)
			n := 0
			for _, chat := range chats {
				jidStr := chat.JID
				cnt := chat.Count
				if strings.HasSuffix(jidStr, "@bot") {
					continue
				}
				var name string
				if strings.HasSuffix(jidStr, "@g.us") {
					parsed, err := types.ParseJID(jidStr)
					if err == nil {
						if gi, err := ctx.Client.GetGroupInfo(context.Background(), parsed); err == nil {
							name = gi.Name
						}
					}
				} else if strings.HasSuffix(jidStr, "@s.whatsapp.net") {
					if pushName := db.GetContactName(jidStr); pushName != "" {
						name = pushName
					}
				} else if strings.HasSuffix(jidStr, "@lid") {
					userPart := strings.TrimSuffix(jidStr, "@lid")
					if pushName := db.GetContactNameByLID(userPart); pushName != "" {
						name = pushName
					}
				}
				if name == "" {
					continue
				}
				n++
				fmt.Fprintf(&sb, "%d. %s — %d msgs\n", n, name, cnt)
			}
			if n == 0 {
				ctx.Reply(T().MessagesEmpty)
				return nil
			}
			ctx.Reply(strings.TrimRight(sb.String(), "\n"))
			return nil
		},
	})

	Register(&Command{
		Pattern:  "active",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			chatJID := ctx.Msg.Chat.String()
			senders, err := db.GetActiveSenders(chatJID)
			if err != nil {
				return err
			}

			var sb strings.Builder
			sb.WriteString(T().ActiveHeader)
			var mentions []string
			n := 0
			for _, s := range senders {
				n++
				senderJID := s.SenderJID
				cnt := s.Count

				userPart := senderJID
				if idx := strings.Index(senderJID, "@"); idx != -1 {
					userPart = senderJID[:idx]
				}
				sb.WriteString(fmt.Sprintf("%d. @%s — %d msgs\n", n, userPart, cnt))
				mentions = append(mentions, senderJID)
			}
			if n == 0 {
				ctx.Reply(T().ActiveEmpty)
				return nil
			}
			sendMention(ctx, strings.TrimRight(sb.String(), "\n"), mentions)
			return nil
		},
	})

	Register(&Command{
		Pattern:  "inactive",
		IsGroup:  true,
		IsAdmin:  true,
		Category: "group",
		Func: func(ctx *Context) error {
			group, err := ctx.Client.GetGroupInfo(context.Background(), ctx.Msg.Chat)
			if err != nil {
				ctx.Reply(fmt.Sprintf(T().GroupInfoFailed, err.Error()))
				return nil
			}

			chatJID := ctx.Msg.Chat.String()
			allSenders, err := db.GetAllSenderCounts(chatJID)
			if err != nil {
				return err
			}

			msgCounts := map[string]int{}
			for _, s := range allSenders {
				senderJID := s.SenderJID
				userPart := senderJID
				if idx := strings.Index(senderJID, "@"); idx != -1 {
					userPart = senderJID[:idx]
				}
				msgCounts[userPart] = s.Count
			}

			getMsgCount := func(p types.GroupParticipant) int {
				if cnt, ok := msgCounts[p.JID.User]; ok {
					return cnt
				}
				if p.LID.User != "" {
					if cnt, ok := msgCounts[p.LID.User]; ok {
						return cnt
					}
				}
				if p.PhoneNumber.User != "" {
					if cnt, ok := msgCounts[p.PhoneNumber.User]; ok {
						return cnt
					}
				}
				return 0
			}

			type entry struct {
				jid types.GroupParticipant
				cnt int
			}
			var inactive []entry
			for _, p := range group.Participants {
				cnt := getMsgCount(p)
				if cnt == 0 {
					inactive = append(inactive, entry{p, 0})
				}
			}

			if len(inactive) == 0 {
				ctx.Reply(T().InactiveEmpty)
				return nil
			}
			if len(inactive) > 20 {
				inactive = inactive[:20]
			}

			var sb strings.Builder
			sb.WriteString(T().InactiveHeader)
			var mentions []string
			for i, e := range inactive {

				displayUser := e.jid.PhoneNumber.User
				if displayUser == "" {
					displayUser = e.jid.JID.User
				}
				fullJID := displayUser + "@s.whatsapp.net"
				sb.WriteString(fmt.Sprintf("%d. @%s — 0 msgs\n", i+1, displayUser))
				mentions = append(mentions, fullJID)
			}
			sendMention(ctx, strings.TrimRight(sb.String(), "\n"), mentions)
			return nil
		},
	})
}
