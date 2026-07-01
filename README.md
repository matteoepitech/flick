<div align="center">

# ⚡ Flick

<b>Share files in a flick.</b>

Upload a file, get a simple code like `ocean-tiger-42`, share it. That's it.

<img src="docs/assets/banner.png" width="100%" alt="Flick">

<br>

<img alt="GitHub Repo stars" src="https://img.shields.io/github/stars/Flick-Corp/flick?label=Star%20Repo&style=social">

<img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go">
<img src="https://img.shields.io/badge/Next.js-16-black?logo=next.js" alt="Next.js">
<img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL">
<img src="https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white" alt="Docker">

</div>

<br>

## What is Flick?

**Flick** is a sleek and lightweight **file sharing tool** built with **Go** and **Next.js**. It's not here to replace WeTransfer or Dropbox but rather to offer a simple, modern, and hassle-free way to share files.

Use it right away on **[flick.d3l.tech](https://flick.d3l.tech)**. Our hosted server is free for anyone, no account or setup needed.

Or **self-host** it for your organization: private groups, team management, and full control over your data and infrastructure.

You send a file and Flick gives you a short code that is easy to remember or say out loud. The other person enters the code, from the website or the terminal, and gets the file. Files clean themselves up: they expire after a while or after a few downloads.

```
$ flick myfile.pdf
Uploading the file myfile.pdf... (2097152 bytes)
Uploading 100% |████████████████████████████████| (2.1/2.1 MB, 46 MB/s)

Code: ocean-tiger-42 [15m left]
Code copied to clipboard.
```

```
$ flick
Specify the code: ocean-tiger-42
Searching the code ocean-tiger-42...
Downloading 100% |████████████████████████████████| (2.1/2.1 MB, 51 MB/s)
```

### Why Flick?

✅ &nbsp;No accounts, no long links: just a short code, easy to say over the phone<br>
✅ &nbsp;Self-destructing files: expire by time, by download count, or both<br>
✅ &nbsp;Use it instantly on our free hosted server at [flick.d3l.tech](https://flick.d3l.tech)<br>
✅ &nbsp;Self-host for your org: private groups, your own rules, your infrastructure<br>
✅ &nbsp;Open-source: your files never leave your server

### What's in a Name?

A "flick" is a quick, effortless movement of the finger. That's exactly how sharing a file should feel. 😄

## Quick Start

Get Flick running in minutes.

### Docker (recommended)

All you need is [Docker](https://docs.docker.com/get-docker/) installed on your server.

```
git clone https://github.com/Flick-Corp/flick.git
cd flick

# Create your configuration
make setup

# Start everything: database, migrations, API, web app
make up
```

Open the address you set during `make setup` (`http://localhost` by default). 🎉

> [!IMPORTANT]
> Keep your `POSTGRES_PASSWORD` safe. `make setup` generates a strong random one; if
> you write `.env` by hand, set a strong password before starting for the first time.

> [!NOTE]
> Everything goes through the bundled [Caddy](https://caddyserver.com/) reverse proxy.
> To serve Flick on your own domain with automatic HTTPS (Let's Encrypt) and HTTP/3,
> set your domain in `.env`:
>
> ```bash
> FLICK_SITE_ADDRESS=flick.example.com
> ```
>
> No certificate to generate or manage: Caddy takes care of it.

> [!TIP]
> **Already have your own reverse proxy?** (Nginx Proxy Manager, Traefik, Caddy...)
> `make setup` takes care of it: answer "yes" when it asks, then just point your proxy
> at the Flick host on **port 80**. Your proxy keeps handling TLS, and there is nothing
> to edit by hand.

To stop Flick, run `make down`. Your data is kept safe.

### Development mode

Use the dev stack when you want to hack on Flick: the API is rebuilt from source and the web app hot-reloads.

```
make dev        # start the dev stack
make down-dev   # stop and clean up
make help       # see all available commands
```

## The CLI

Prefer the terminal? On Debian/Ubuntu, install `flick` straight from the APT repository:

```bash
curl -fsSL https://apt.d3l.tech/apt/flick.gpg | sudo tee /usr/share/keyrings/flick.gpg > /dev/null
echo "deb [signed-by=/usr/share/keyrings/flick.gpg] https://apt.d3l.tech/apt stable main" | sudo tee /etc/apt/sources.list.d/flick.list
sudo apt update && sudo apt install flick
```

On macOS, install via Homebrew:

```bash
brew install Flick-Corp/flick/flick
```

On other platforms (Windows, other Linux), grab the `flick` binary from
[apt.d3l.tech/releases](https://apt.d3l.tech/releases/) - the CLI keeps itself up to date afterwards.

```bash
# Send a file
flick holiday-photos.zip

# Send with custom rules: expires in 1 hour, max 3 downloads
flick holiday-photos.zip --exp 1h --mdc 3

# Receive a file (it will ask you for the code)
flick

# Set your defaults (server address, expiration...)
flick configure
```

You can also build the binaries yourself:

```bash
make build      # requires Go 1.26+, outputs to build/bin/
```

## Features

Flick combines effortless sharing with full control over your server.

### 📦 &nbsp;Sharing Essentials

- Human-friendly codes in the `word-word-number` format, easy to type and remember
- Share code copied automatically to your clipboard after upload
- Self-destructing files: expiration by time, by download count, or both
- Fast transfers with **HTTP/3 (QUIC)** and automatic HTTPS, served at the edge by Caddy

### 🌐 &nbsp;Web Experience

- **Send**: drag and drop a file, get your code
- **Receive**: enter a code, download the file
- **Dashboard** for admins with stats, users and server settings
- Localized interface with light and dark mode

### 🛡️ &nbsp;Control and Limits

- Configurable max file size, default and max expiration, download counts
- Rate limiting per user, per IP, and global hourly caps
- PostgreSQL persistence with automatic database migrations

## Self-Hosting

Flick is built for easy self-hosting. Docker Compose gets you up in minutes with a database, API, web app, and a Caddy reverse proxy that handles automatic HTTPS.

Perfect for organizations that want private groups, custom limits, and full data sovereignty.

See the [Quick Start](#quick-start) section above to get started.

## Technologies Used

- [Go](https://go.dev/) with [Cobra](https://github.com/spf13/cobra)
- [Caddy](https://caddyserver.com/) for automatic HTTPS and HTTP/3 at the edge
- [Next.js](https://nextjs.org/) with [shadcn/ui](https://ui.shadcn.com/) and [Tailwind CSS](https://tailwindcss.com/)
- [PostgreSQL](https://www.postgresql.org/) with [dbmate](https://github.com/amacneil/dbmate) migrations
- [Docker Compose](https://docs.docker.com/compose/) for one-command deployment

## Contributing

Contributions are welcome! Feel free to open an issue or submit a pull request.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=Flick-Corp/flick&type=Date)](https://star-history.com/#Flick-Corp/flick&Date)

---

<p align="center">
    Made with ❤️ · © 2026 Flick. All rights reserved.
</p>
