---
layout: page
title: Command Reference
nav_order: 4
---

# Command Reference

{: .no_toc }

All commands use the configured prefix (default `.`). **Sudo** - sender must be a sudo user or the owner. **Admin** - bot and sender must both be group admins. **Group** - only works in group chats.

<details open markdown="block">
  <summary>Table of contents</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## CLI flags

These are passed to the `mack` binary, not sent in chat.

| Flag                        | Description                             |
| --------------------------- | --------------------------------------- |
| `--phone-number <number>`   | Pair or identify a device               |
| `--update`                  | Pull latest source and rebuild in-place |
| `--list-sessions`           | List all paired sessions                |
| `--delete-session <number>` | Permanently delete a session            |
| `--reset-session <number>`  | Reset a session for re-pairing          |
| `--version`                 | Print version, commit, and build date   |
| `-h / --help`               | Show help                               |

---

## Utility

### menu / help

Show the interactive command menu grouped by category.

```
.menu
.help
```

### ping

Reply with `pong` and the round-trip latency.

```
.ping
```

### lang

Change the language used for all bot responses.

```
.lang <code>
```

Supported codes: `en` `es` `pt` `ar` `hi` `fr` `de` `ru` `tr` `sw`

### update

Pull the latest source from GitHub and rebuild the binary in-place. **Sudo only.**

```
.update
```

---

## Settings - sudo only

### setprefix

Change the command prefix(es). Separate multiple with spaces. Use `empty` for no prefix.

```
.setprefix .
.setprefix . ! empty
```

### setmode / getmode

```
.setmode public
.setmode private
```

### setsudo / delsudo / getsudo

```
.setsudo <phone>
.delsudo <phone>
.getsudo
```

### ban / unban

Silently ignore (or unignore) a user.

```
.ban <phone>
.unban <phone>
```

### disable / enable

Disable or re-enable a command by name.

```
.disable <command>
.enable  <command>
```

---

## Moderation

### antidelete

Forward deleted messages to the bot owner.

```
.antidelete on
.antidelete off
```

### antilink - admin, group

Remove messages containing WhatsApp group invite links.

```
.antilink on
.antilink off
```

### antispam - admin, group

Auto-mute members who send messages too rapidly.

```
.antispam on
.antispam off
```

### anticall

Reject or block incoming calls.

```
.anticall on
.anticall off
.anticall reject
.anticall block
```

### antistatus

Block automatic status-update read receipts.

```
.antistatus on
.antistatus off
```

### antiword - admin, group

Remove messages containing specified words and warn the sender.

```
.antiword add <word>
.antiword remove <word>
.antiword list
.antiword on
.antiword off
```

### antivv

Auto-open view-once messages and forward them back to the sender.

```
.antivv on
.antivv off
```

Manually open a specific view-once (reply to it):

```
.vv
```

### warn - admin, group

Issue a warning (reply to a message). Three warnings trigger an auto-kick.

```
.warn
```

### report

Report a message to the bot owner (reply to it).

```
.report
```

---

## Group administration - admin, group

### promote / demote

```
.promote @user
.demote  @user
```

### kick / kickall

```
.kick @user
.kickall
```

### mute / unmute

Restrict all members from sending messages.

```
.mute
.mute off
```

### shh

Toggle whether Mack-Bot responds to commands in groups.

```
.shh
.shh off
```

### newgc

Create a new group with participants taken from mentions or a reply.

```
.newgc <group name>
```

### filter / gfilter / dfilter

Auto-reply to matching keywords.

```
.filter  <trigger> | <response>
.gfilter <trigger> | <response>
.dfilter <trigger>
```

---

## Chat utilities

### del

Delete a message (reply to it). Bot must be admin in groups.

```
.del
```

### star / unstar

```
.star
.unstar
```

### pin / unpin - admin, group

```
.pin
.unpin
```

### archive

Archive a chat.

```
.archive
```

### block / unblock

```
.block
.unblock
```

### afk

Mark yourself as away. Mack-Bot auto-replies while you are AFK.

```
.afk
.afk <reason>
.afk off
```

---

## Media

### mp3

Extract audio from a video as an MP3 (reply to the video).

```
.mp3
```

### trim

Trim a video to a time range (reply to the video).

```
.trim <start> <end>
```

Example: `.trim 0:05 0:30`

### black

Remove black borders from a video (reply to it).

```
.black
```

---

## Downloads

### yt

Download a YouTube video (returns a stream link).

```
.yt <url>
```

### ytaudio

Download audio from a YouTube video.

```
.ytaudio <url>
```

### ytsearch

Search YouTube and return the top 5 results.

```
.ytsearch <query>
```

### spotify

Get Spotify track/album info.

```
.spotify <url>
```

### tweet

Get Twitter/X post info.

```
.tweet <url>
```

### reddit

Get Reddit post info.

```
.reddit <url>
```

---

## Status

### autosavestatus

Automatically save all contact status updates.

```
.autosavestatus on
.autosavestatus off
```

### autolikestatus

Automatically react with a heart emoji to every contact status update.

```
.autolikestatus on
.autolikestatus off
```

---

## AI

### meta

Send a prompt to Meta AI and receive a reply in the chat.

```
.meta <prompt>
```

Example: `.meta What is the capital of France?`
