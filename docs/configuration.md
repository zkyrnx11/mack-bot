---
layout: page
title: Configuration
nav_order: 3
---

# Configuration

{: .no_toc }

<details open markdown="block">
  <summary>Table of contents</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## Data directory

Mack-Bot stores all data in `~/Documents/Mack-Bot/` (created automatically). In Docker this maps to `/root/Documents/Mack-Bot/`, which is bind-mounted to `./data` by the provided `docker-compose.yml`.

---

## Runtime settings

Bot settings are stored per-owner in the database and can be changed from chat at any time without restarting.

### Prefix

The character(s) that must precede a command name.

```
.setprefix .
.setprefix . ! empty    # multiple prefixes; "empty" allows no prefix
```

Default: `.`

### Mode

| Value     | Who can use commands |
| --------- | -------------------- |
| `public`  | Everyone (default)   |
| `private` | Sudo users only      |

```
.setmode public
.setmode private
```

### Sudo users

Sudo users bypass all permission checks. The owner is added automatically on first start.

```
.setsudo <phone>    # grant sudo
.delsudo <phone>    # revoke sudo
.getsudo            # list all sudo users
```

### Language

```
.lang en   .lang es   .lang pt   .lang ar   .lang hi
.lang fr   .lang de   .lang ru   .lang tr   .lang sw
```

### Other toggles

```
.shh / .shh off          # disable / re-enable group responses
.antidelete on/off        # forward deleted messages to owner
.ban <phone>              # silently ignore a user
.unban <phone>
.disable <command>        # disable a specific command by name
.enable  <command>
```

---

## Build-time flags

When building from source, these ldflags are injected automatically by `make build` and `make release`:

| Flag                          | Description                         |
| ----------------------------- | ----------------------------------- |
| `-X main.Version=x.y.z`       | Version string shown by `--version` |
| `-X main.Commit=<sha>`        | Short git commit hash               |
| `-X main.BuildDate=<rfc3339>` | Build timestamp                     |
| `-X main.sourceDir=<path>`    | Source directory used by `--update` |

Manual example:

```bash
go build \
  -ldflags="-s -w -X main.Version=1.0.0 -X main.sourceDir=$(pwd)" \
  -trimpath -o mack ./
```
