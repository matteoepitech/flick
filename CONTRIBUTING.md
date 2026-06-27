# Contributing to Flick

First off, thanks for taking the time to contribute! ⚡

Flick is a lightweight file sharing tool built with **Go** and **Next.js**.
This document explains how to set up your environment, the conventions we follow, and how to get your changes merged.

By participating in this project, you agree to abide by our
[Code of Conduct](CODE_OF_CONDUCT.md).

## Table of Contents

- [Ways to Contribute](#ways-to-contribute)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Coding Guidelines](#coding-guidelines)
- [Commit Messages](#commit-messages)
- [Pull Requests](#pull-requests)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)

## Ways to Contribute

- 🐛 **Report bugs** by opening an issue
- 💡 **Suggest features** or improvements
- 📖 **Improve documentation** (README, this guide, code comments)
- 🔧 **Submit code** via pull requests (bug fixes, features, refactors)

If you are planning a large change, please open an issue first to discuss it.
This avoids duplicated effort and makes sure the change fits the project.

## Development Setup

You'll need [Docker](https://docs.docker.com/get-docker/) and, for the CLI [Go 1.26+](https://go.dev/dl/).

```bash
# Clone your fork
git clone https://github.com/<your-username>/flick.git
cd flick

# Create your configuration
cp .env.example .env

# Start the dev stack: API rebuilt from source, web app hot-reloads
make dev

# Stop and clean up the dev stack
make down-dev

# See all available commands
make help
```

The web app is served through the bundled [Caddy](https://caddyserver.com/) reverse proxy. Open `http://localhost` once the stack is up.

### Building the CLI

```bash
make build      # outputs binaries to build/bin/
```

### Database migrations

```bash
make migrate-new name=<name>   # create a new migration
make migrate-up                # apply pending migrations
make migrate-down              # roll back the last migration
make migrate-status            # show migration status
make sqlc-generate             # regenerate type-safe Go from db/queries
```

## Project Structure

```
cmd/        Go entry points (API server, CLI)
internal/   Go application code (not importable from outside)
db/         SQL queries and dbmate migrations
web/        Next.js front-end (shadcn/ui, Tailwind CSS)
scripts/    Build and helper scripts
docs/       Documentation and assets
```

## Coding Guidelines

- **Go**: format with `gofmt` (run `go fmt ./...`) and keep `go vet ./...`
  clean. Follow the conventions already present in `internal/` and `cmd/`.
- **Web**: follow the existing Next.js / TypeScript style; use the configured
  linter and formatter before committing.
- Keep changes focused. One logical change per pull request.
- Add or update documentation when behavior changes.

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/). The format is:

```
<type>(<scope>): <short description>
```

Common types: `feat`, `fix`, `docs`, `refactor`, `chore`, `test`, `ci`.

Examples (matching the project history):

```
feat(cli): add resumable uploads
fix(explore-cli): fix the explore mode footer sometimes disappears
docs: add contributing guide
```

Keep the subject line concise and in the imperative mood.

## Pull Requests

1. Fork the repository and create a branch from `main`
   (e.g. `feat/my-feature` or `fix/some-bug`).
2. Make your changes, following the guidelines above.
3. Make sure the project builds and existing checks pass.
4. Push to your fork and open a pull request against `main`.
5. Describe **what** you changed and **why**. Link any related issue.

A maintainer will review your PR. Please be responsive to feedback;
small follow-up commits are fine and we squash where appropriate.

## Reporting Bugs

When opening a bug report, please include:

- A clear description of the problem
- Steps to reproduce
- What you expected to happen vs. what actually happened
- Your environment (OS, Flick version / commit, Docker version)
- Relevant logs or screenshots

## Suggesting Features

Open an issue describing:

- The problem you're trying to solve
- Your proposed solution
- Any alternatives you've considered

---

Thanks again for contributing to Flick! 🙌
