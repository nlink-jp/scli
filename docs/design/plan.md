# scli — Development Plan

## Phases Overview

| Phase | Title | Goal |
|-------|-------|------|
| 0 | Scaffolding | Buildable skeleton, toolchain, Git hooks |
| 1 | Auth & Config | OAuth login, token storage, workspace management |
| 2 | Core Channel | Channel list / read / post |
| 3 | DM | Direct message send / read |
| 4 | Unread & Users | Unread summary, user list |
| 5 | Extended Features | Search, threads, file attachment |
| 6 | Polish & Release | Cross-compile, security scan, docs, v1.0.0 |

---

## Phase 0 — Scaffolding

**Goal**: A compilable, testable skeleton with all tooling in place.

### Tasks

- [ ] `go mod init github.com/<org>/scli`
- [ ] Directory structure (`cmd/`, `internal/`, `docs/`, `scripts/hooks/`)
- [ ] `Makefile` with targets: `build`, `build-all`, `test`, `lint`, `check`, `setup`, `clean`
- [ ] Cross-compilation targets: `linux/amd64`, `linux/arm64`, `darwin/arm64`, `windows/amd64` (darwin is arm64-only)
- [ ] `cobra` root command (`cmd/root.go`) with global flags: `--workspace`, `--json`, `--no-color`
- [ ] Git hooks (`scripts/hooks/pre-commit`, `scripts/hooks/pre-push`) running `make check`
- [ ] `make setup` installs hooks automatically
- [ ] `golangci-lint` config (`.golangci.yml`)
- [ ] `CHANGELOG.md` initial entry

### Completion Criteria

`make check` passes (lint + test + build-all) on a clean clone.

---

## Phase 1 — Auth & Config

**Goal**: Users can authenticate with Slack and switch workspaces.

### Tasks

- [ ] `internal/keychain` — OS keychain abstraction (`zalando/go-keyring`)
  - Interface: `Store{Get, Set, Delete}`
  - Unit tests with a mock store
- [ ] `internal/config` — Config file + env var reader
  - Token resolution chain: env var → `.env` → `config.json` → keychain
  - Read/write `~/.config/scli/config.json`
  - Unit tests
- [ ] `internal/auth` — OAuth 2.0 PKCE flow
  - Local HTTP server on `localhost:7777` for callback
  - Browser launch (cross-platform)
  - Token exchange and storage
  - Integration test (mock Slack OAuth endpoint)
- [ ] `cmd/auth.go` — `scli auth login / logout / list`
- [ ] `cmd/workspace.go` — `scli workspace list / use`
- [ ] `docs/setup.md` updated with any changes discovered during implementation

### Completion Criteria

`scli auth login`, `scli auth logout`, `scli workspace list`, `scli workspace use` all work
against a real Slack workspace.

---

## Phase 2 — Core Channel

**Goal**: Users can list channels, read messages, and post.

### Tasks

- [ ] `internal/output` — Formatter
  - Color-formatted text renderer (with ANSI codes)
  - JSON renderer
  - TTY auto-detection (disable color when not a terminal)
  - Unit tests
- [ ] `internal/slack` — Slack API client (initial)
  - HTTP client wrapper with auth header injection
  - `conversations.list` — channel list
  - `conversations.history` — message history
  - `chat.postMessage` — post message
  - Channel/user name resolution (`#name` → ID, `@name` → ID)
  - Unit tests with mocked HTTP responses
- [ ] `cmd/channel.go` — `scli channel list / read`
- [ ] `cmd/post.go` (or as subcommand) — `scli post <channel> <message>`

### Completion Criteria

Can list channels, read last N messages with color output, and post a message.
`--json` flag produces valid JSON on all commands.

---

## Phase 3 — DM

**Goal**: Users can send and read direct messages.

### Tasks

- [ ] `internal/slack` additions
  - `conversations.open` — open/find DM channel
  - `im.list` / `conversations.list` filtered for DMs
  - User lookup by name (`users.list`)
- [ ] `cmd/dm.go` — `scli dm list / read / send`
- [ ] `cmd/user.go` — `scli user list`

### Completion Criteria

Can list DM conversations, read messages, and send a DM to a user by `@name` or user ID.

---

## Phase 4 — Unread & Users

**Goal**: Quick overview of what needs attention.

### Tasks

- [ ] `internal/slack` additions
  - `users.conversations` — channels the user is in
  - Unread count via `conversations.info` (field: `unread_count`)
- [ ] `cmd/unread.go` — `scli unread`
  - Displays channel name + unread count, sorted by count descending
  - Skips channels with zero unread

### Completion Criteria

`scli unread` prints a summary of all channels/DMs with unread messages.

---

## Phase 5 — Extended Features

**Goal**: Thread replies, file attachments, search.

### Tasks

- [ ] Thread support
  - `--thread <timestamp>` flag on `scli post` and `scli dm send`
  - `--thread <timestamp>` flag on `scli channel read` to display a thread
  - `conversations.replies` API call
- [ ] File attachment
  - `--file <path>` flag on `scli post`
  - `files.getUploadURLExternal` + `files.completeUploadExternal` (v2 upload API)
- [ ] Search
  - `cmd/search.go` — `scli search <query> [--in <channel>] [--limit N]`
  - `search.messages` API call

### Completion Criteria

Thread replies work end-to-end. Files upload and appear in Slack.
Search returns results with channel context.

---

## Phase 6 — Polish & Release

**Goal**: Production-ready v1.0.0.

### Tasks

- [ ] `govulncheck ./...` clean
- [ ] All `golangci-lint` warnings resolved
- [ ] Cross-compilation verified on all 5 targets
- [ ] `docs/dependencies.md` completed
- [ ] `docs/setup.md` and `docs/ja/setup.md` final review
- [ ] `docs/design/overview.md` updated to reflect any implementation changes
- [ ] `CHANGELOG.md` v1.0.0 entry
- [ ] Git tag `v1.0.0`

### Completion Criteria

`make check` passes. Binaries for all platforms build cleanly.
All documentation is in sync with the implementation.

---

## Dependency Plan

| Library | Purpose | Phase |
|---------|---------|-------|
| `github.com/spf13/cobra` | CLI framework | 0 |
| `github.com/zalando/go-keyring` | OS keychain abstraction | 1 |
| `github.com/fatih/color` | ANSI color output | 2 |
| `github.com/joho/godotenv` | `.env` file loading | 1 |

All dependencies to be documented in `docs/dependencies.md` before use.

---

*Japanese translation: `docs/ja/design/plan.md`*
