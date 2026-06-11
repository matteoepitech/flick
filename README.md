<div align="center">

# ⚡ Flick

**Share files in a flick.**

Upload a file, get a simple code like `ocean-tiger-42`, share it. That's it.

<!-- Add a screenshot or banner here once available:
<img src="docs/assets/banner.png" width="100%" alt="Flick in action">
-->

<img alt="GitHub Repo stars" src="https://img.shields.io/github/stars/matteoepitech/flick?label=Star%20Repo&style=social">

<img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go">
<img src="https://img.shields.io/badge/Next.js-16-black?logo=next.js" alt="Next.js">
<img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL">
<img src="https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white" alt="Docker">

<br><br>

<picture>
    <source srcset="https://fonts.gstatic.com/s/e/notoemoji/latest/26a1/512.webp" type="image/webp">
    <img src="https://fonts.gstatic.com/s/e/notoemoji/latest/26a1/512.gif" alt="⚡" width="32" height="32">
</picture>
<picture>
    <source srcset="https://fonts.gstatic.com/s/e/notoemoji/latest/1f680/512.webp" type="image/webp">
    <img src="https://fonts.gstatic.com/s/e/notoemoji/latest/1f680/512.gif" alt="🚀" width="32" height="32">
</picture>
<picture>
    <source srcset="https://fonts.gstatic.com/s/e/notoemoji/latest/1f389/512.webp" type="image/webp">
    <img src="https://fonts.gstatic.com/s/e/notoemoji/latest/1f389/512.gif" alt="🎉" width="32" height="32">
</picture>

</div>

<div align="center">

<br>

<a href="https://flick.d3l.tech"><kbd> &nbsp;🌍 Live Server&nbsp; </kbd></a>&nbsp;
<a href="#quick-start"><kbd> &nbsp;🎉 Quick Start&nbsp; </kbd></a>&nbsp;
<a href="#the-cli"><kbd> &nbsp;💻 The CLI&nbsp; </kbd></a>&nbsp;
<a href="#configuration"><kbd> &nbsp;⚙️ Configuration&nbsp; </kbd></a>

</div>

## What is Flick?

**Flick** is a sleek and lightweight **self-hosted file sharing tool** built with **Go** and **Next.js**. It's not here to replace WeTransfer or Dropbox but rather to offer a simple, modern, and hassle-free way to share files from your own server with minimal effort.

You send a file and Flick gives you a short code that is easy to remember or say out loud. The other person enters the code, from the website or the terminal, and gets the file. Files clean themselves up: they expire after a while or after a few downloads.

```console
$ flick-cli myfile.pdf
Uploading the file myfile.pdf... (2097152 bytes)
Uploading 100% |████████████████████████████████| (2.1/2.1 MB, 46 MB/s)

Code: ocean-tiger-42 [15m left]
Code copied to clipboard.
```

```console
$ flick-cli
Specify the code: ocean-tiger-42
Searching the code ocean-tiger-42...
Downloading 100% |████████████████████████████████| (2.1/2.1 MB, 51 MB/s)
```

### Why Flick?

✅ &nbsp;No accounts, no long links: just a short code, easy to say over the phone<br>
✅ &nbsp;Self-destructing files: expire by time, by download count, or both<br>
✅ &nbsp;Your server, your rules: max file size, duration, rate limits, all configurable<br>
✅ &nbsp;Open-source and self-hosted: your files never leave your server

### What's in a Name?

A "flick" is a quick, effortless movement of the finger. That's exactly how sharing a file should feel. 😄

## Quick Start

Get Flick running in minutes.

### Docker (recommended)

All you need is [Docker](https://docs.docker.com/get-docker/) installed on your server.

```bash
git clone https://github.com/matteoepitech/flick.git
cd flick

# Create your configuration
cp .env.example .env

# Start everything: database, migrations, API, web app
make up
```

Open `http://localhost:3000`. 🎉

> [!IMPORTANT]
> Set a strong `POSTGRES_PASSWORD` in `.env` before starting for the first time.

> [!NOTE]
> Flick runs on HTTPS only. Put your `cert.pem` and `key.pem` in the `certificates/` folder.
> For a quick local test, generate a self-signed certificate:
>
> ```bash
> openssl req -x509 -newkey rsa:4096 -nodes \
>   -keyout certificates/key.pem -out certificates/cert.pem \
>   -days 365 -subj "/CN=localhost"
> ```

| Service | URL |
| ------- | --- |
| 🌐 Web app | http://localhost:3000 |
| ⚙️ API | https://localhost:15702 |

To stop Flick, run `make down`. Your data is kept safe.

### Development mode

Use the dev stack when you want to hack on Flick: the API is rebuilt from source and the web app hot-reloads.

```bash
make dev        # start the dev stack
make down-dev   # stop and clean up
make help       # see all available commands
```

## The CLI

Prefer the terminal? Grab the `flick-cli` binary for your platform (Linux, macOS or Windows):

```bash
# Send a file
flick-cli holiday-photos.zip

# Send with custom rules: expires in 1 hour, max 3 downloads
flick-cli holiday-photos.zip --exp 1h --mdc 3

# Receive a file (it will ask you for the code)
flick-cli

# Set your defaults (server address, expiration...)
flick-cli configure
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
- Fast transfers built on **HTTP/3 (QUIC)** with HTTP/2 fallback

### 🌐 &nbsp;Web Experience

- **Send**: drag and drop a file, get your code
- **Receive**: enter a code, download the file
- **Dashboard** for admins with stats, users and server settings
- Localized interface with light and dark mode

### 🛡️ &nbsp;Control and Limits

- Configurable max file size, default and max expiration, download counts
- Rate limiting per user, per IP, and global hourly caps
- PostgreSQL persistence with automatic database migrations

## Configuration

The server creates a default configuration on first start. The main knobs:

| Setting | Default | What it does |
| ------- | ------- | ------------ |
| `max_file_size_mb` | `1000` | Biggest file allowed |
| `default_expiration` | `15m` | How long files live by default |
| `max_expiration` | `4h` | Longest a file can live |
| `default_download_count` | `1` | Downloads allowed by default |
| `max_download_count` | `5` | Most downloads allowed per file |
| `activate_rate_limit` | `true` | Protect your server from abuse |

## Technologies Used

- [Go](https://go.dev/) with [quic-go](https://github.com/quic-go/quic-go) (HTTP/3) and [Cobra](https://github.com/spf13/cobra)
- [Next.js](https://nextjs.org/) with [shadcn/ui](https://ui.shadcn.com/) and [Tailwind CSS](https://tailwindcss.com/)
- [PostgreSQL](https://www.postgresql.org/) with [dbmate](https://github.com/amacneil/dbmate) migrations
- [Docker Compose](https://docs.docker.com/compose/) for one-command deployment

## Contributing

Contributions are welcome! Feel free to open an issue or submit a pull request.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=matteoepitech/flick&type=Date)](https://star-history.com/#matteoepitech/flick&Date)

---

<p align="center">
    Made with ❤️ · © 2026 Flick. All rights reserved.
</p>
