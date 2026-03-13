---
layout: page
title: Plugin Development
nav_order: 5
---

# Plugin Development

{: .no_toc }

Mack-Bot uses a simple registration-based plugin system. Every `.go` file inside the `src/plugins/` package can register one or more commands using `init()` functions — no configuration files, no framework, just Go.

<details open markdown="block">
  <summary>Table of contents</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## Anatomy of a command

```go
package plugins

func init() {
    Register(&Command{
        Pattern:  "hello",             // primary trigger word
        Aliases:  []string{"hi"},      // optional aliases
        Category: "utility",           // shown in .menu grouping
        IsSudo:   false,               // true = sudo users only
        IsAdmin:  false,               // true = requires bot+sender to be admin (groups)
        IsGroup:  false,               // true = only works in group chats
        Func: func(ctx *Context) error {
            ctx.Reply("Hello, " + ctx.Event.Info.PushName + "!")
            return nil
        },
    })
}
```

Place this file anywhere inside `src/src/plugins/` and it will be compiled in automatically.

---

## The Command struct

| Field      | Type                   | Description                                     |
| ---------- | ---------------------- | ----------------------------------------------- |
| `Pattern`  | `string`               | Primary command name (case-insensitive)         |
| `Aliases`  | `[]string`             | Alternative names that trigger the same handler |
| `Category` | `string`               | Menu category label (default: `"general"`)      |
| `IsSudo`   | `bool`                 | Only sudo users may invoke this command         |
| `IsAdmin`  | `bool`                 | Bot and sender must both be group admins        |
| `IsGroup`  | `bool`                 | Command only works inside group chats           |
| `Func`     | `func(*Context) error` | The command handler                             |

---

## The Context object

`*Context` is passed to every command handler and provides everything needed to respond.

```go
type Context struct {
    Client     *whatsmeow.Client   // underlying WhatsApp client
    Event      *events.Message     // the triggering message event
    Args       []string            // words after the command name
    Text       string              // full text after the command name (unsplit)
    Prefix     string              // matched prefix character(s)
    Matched    string              // matched command name (lowercased)
    ReceivedAt time.Time           // when the event was dispatched
}
```

### Replying

```go
// Async reply (non-blocking, background send)
ctx.Reply("message text")

// Sync reply (blocks until server ACK — use when you need the message ID)
resp, err := ctx.ReplySync("message text")

// Edit a previously sent message
ctx.QueueEdit(resp.ID, "updated text")
```

---

## Accessing arguments

```go
// Full text after the command
ctx.Text  // e.g. "hello world"

// Whitespace-split words
ctx.Args  // e.g. ["hello", "world"]
ctx.Args[0] // first word
```

---

## Reading settings

```go
// Check if the sender is sudo
isSudo := BotSettings.IsSudo(ctx.Event.Info.Sender.User)

// Get current language
lang := BotSettings.GetLanguage()

// Check current mode
mode := BotSettings.GetMode() // ModePublic or ModePrivate
```

---

## Sending a file / media

Use the underlying `ctx.Client` directly for operations beyond plain-text replies — the [whatsmeow documentation](https://pkg.go.dev/go.mau.fi/whatsmeow) covers all available methods.

```go
data, _ := os.ReadFile("image.jpg")
uploaded, _ := ctx.Client.Upload(context.Background(), data, whatsmeow.MediaImage)

ctx.Client.SendMessage(context.Background(), ctx.Event.Info.Chat, &waProto.Message{
    ImageMessage: &waProto.ImageMessage{
        Mimetype:      proto.String("image/jpeg"),
        Url:           &uploaded.URL,
        DirectPath:    &uploaded.DirectPath,
        MediaKey:      uploaded.MediaKey,
        FileEncSha256: uploaded.FileEncSHA256,
        FileSha256:    uploaded.FileSHA256,
        FileLength:    proto.Uint64(uint64(len(data))),
    },
})
```

---

## Registering a moderation hook

Moderation hooks run on **every incoming message**, before command dispatch. Use them for passive monitoring (e.g. anti-spam logic).

```go
func init() {
    RegisterModerationHook(func(client *whatsmeow.Client, evt *events.Message) {
        // runs on every message — keep it fast
    })
}
```

---

## Internationalization (i18n)

To add translated strings for your plugin, extend the `Strings` struct and each language block in `src/src/plugins/i18n.go`, then access them via:

```go
ctx.Reply(T().YourNewField)
```

`T()` returns the `Strings` struct for the currently configured language.

---

## Building and testing

```bash
# Build
make build        # produces mack.exe on Windows, mack elsewhere

# Run tests
go test ./...

# Run the bot locally
make run
```
