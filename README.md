<div align="center">
  <h1>Mack-Bot</h1>
  <p>A self-hosted WhatsApp bot written in Go.</p>
  <a href="https://zkyrnx11.github.io/mack-bot/"><strong>Documentation →</strong></a>
  &nbsp;·&nbsp;
  <a href="https://github.com/zkyrnx11/mack-bot/releases/latest">Latest Release</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/zkyrnx11/mack-bot/releases/tag/nightly">Nightly Build</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/zkyrnx11/mack-bot/issues">Report a Bug</a>
  <br/><br/>
  <img src="https://img.shields.io/github/v/release/zkyrnx11/mack-bot?style=flat&label=version" alt="Latest release"/>
  <img src="https://img.shields.io/badge/Docker-ready-2496ED?style=flat&logo=docker" alt="Docker"/>
  <img src="https://img.shields.io/github/license/zkyrnx11/mack-bot?style=flat" alt="License"/>
  <img src="https://img.shields.io/github/stars/zkyrnx11/mack-bot?style=flat" alt="Stars"/>
</div>

---

## Quick Start

**1 — Run with Docker**

```bash
mkdir data
docker compose up -d
```

**2 — Or download the binary** from the [latest release](https://github.com/zkyrnx11/mack-bot/releases/latest) for your platform, then run it directly. Data is stored in `~/Documents/Mack-Bot` automatically.

**3 — Pair your phone** (first run only)

```
mack --phone-number <international-number>
```

WhatsApp → Linked Devices → Link with phone number → enter the printed code.

**4 — Done.** Subsequent starts reconnect automatically.

> See the [full installation guide](https://zkyrnx11.github.io/mack-bot/installation) for Docker details, session management, and updating.

## Documentation

Full command reference at **[here](https://zkyrnx11.github.io/mack-bot/commands)**.

**[Here contains](https://zkyrnx11.github.io/mack-bot/)** — installation, configuration, commands, plugin development.

<details>
<summary><strong>Building from Source</strong></summary>

Requires Go 1.25+ and Git. The `patched/` directory (a minimal whatsmeow fork for pin/unpin support) is included in the repo.

```bash
git clone https://github.com/zkyrnx11/mack-bot.git && cd mack
make setup              # install ffmpeg + yt-dlp
make build              # mack.exe / mack
make release VERSION=x.y.z   # cross-platform archives → dist/
```

Install scripts (also build from source):

```bash
# Linux
sudo bash <(curl -fsSL https://raw.githubusercontent.com/zkyrnx11/mack/master/scripts/install-linux.sh)
# macOS
sudo bash <(curl -fsSL https://raw.githubusercontent.com/zkyrnx11/mack/master/scripts/install-mac.sh)
# Windows (PowerShell as Administrator)
irm https://raw.githubusercontent.com/zkyrnx11/mack/master/scripts/install.ps1 | iex
```

</details>

## Contributing

Contributions are **by invitation only** — [get in touch](mailto:danielpeter0081@gmail.com) if you'd like to help.

## License

[MIT](LICENSE)
