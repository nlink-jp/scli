# scli — Design Overview

## 1. Purpose

`scli` is a command-line Slack client that operates as a **user** (not a bot).
The goal is to let users post and read Slack messages without leaving the terminal,
eliminating the context switch to a GUI client.

---

## 2. Design Principles

- **Security first** — tokens are stored in the OS keychain by default; plaintext fallbacks are opt-in.
- **Small and focused** — each command does one thing; no background daemon required.
- **Separation of concerns** — CLI layer, API layer, auth layer, config layer, cache layer, and output layer are independent.
- **Testable** — all layers are connected via interfaces; I/O and business logic are never mixed.

---

## 3. System Context

```
User (terminal)
     │  scli <command>
     ▼
┌─────────────────┐
│   scli (CLI)    │  ← cobra-based command dispatcher
└────────┬────────┘
         │
  ┌──────┴──────┐
  │ Auth Layer  │  ← token resolution (env → config → keychain)
  └──────┬──────┘
         │
  ┌──────┴──────┐
  │ Cache Layer │  ← disk TTL cache + in-memory cache
  └──────┬──────┘
         │
  ┌──────┴──────┐
  │ Slack API   │  ← Slack Web API (HTTPS)
  │   Client    │
  └──────┬──────┘
         │
  ┌──────┴──────┐
  │  Output     │  ← color-formatted text (default) or JSON (--json)
  └─────────────┘
```

---

## 4. Layer Breakdown

### 4.1 cmd/

Cobra command definitions. Each subcommand delegates immediately to an internal service;
no business logic lives in this layer.

```
cmd/
  root.go          # global flags: --workspace, --json, --no-color
  auth.go          # auth login / logout / list
  workspace.go     # workspace list / use
  channel.go       # channel list / joined / read / info / search
  post.go          # post (with --blocks / --blocks-file support)
  dm.go            # dm list / send / read
  unread.go        # unread
  search.go        # search
  user.go          # user list / info / search
  cache.go         # cache clear
```

### 4.2 internal/auth/

OAuth 2.0 PKCE flow against Slack's identity provider.
Responsibilities:
- Spin up a local HTTP server to receive the OAuth callback.
- Exchange the authorization code for a user token (`xoxp-`).
- Hand the token to the keychain/config layer for storage.

### 4.3 internal/config/

Token and workspace profile management.

**Token resolution order (per workspace):**

```
1. Environment variable  SLACK_TOKEN_<WORKSPACE>  (or SLACK_TOKEN for default)
2. .env file             (current directory, or ~/.config/scli/.env)
3. Config file           ~/.config/scli/config.json
4. OS keychain           (see internal/keychain/)
```

Config file schema (`config.json`):

```json
{
  "default_workspace": "myteam",
  "workspaces": {
    "myteam": {
      "token": "",           // leave empty to use keychain
      "team_id": "T012AB3C4",
      "user_id": "U012AB3C4"
    }
  }
}
```

### 4.4 internal/keychain/

Thin abstraction over OS secret storage:

| Platform | Backend                     |
|----------|-----------------------------|
| macOS    | Keychain (via Security.framework) |
| Linux    | libsecret / `secret-tool`   |
| Windows  | Windows Credential Manager  |

Go library: [`zalando/go-keyring`](https://github.com/zalando/go-keyring)

Interface:

```go
type Store interface {
    Get(workspace string) (token string, err error)
    Set(workspace string, token string) error
    Delete(workspace string) error
}
```

### 4.5 internal/slack/

Slack Web API client. Each method maps to one API endpoint.

**Caching** (for large-workspace performance):

- `ListChannels`, `ListJoinedChannels`, and `ListUsers` cache results to disk with a 1-hour TTL.
  Cache location: `~/.config/scli/cache/<workspace>/` (workspace-specific to prevent cross-workspace contamination).
- `GetUser` additionally maintains an in-memory `map[string]User` for repeated lookups within a single process.
- `cache clear` removes the workspace cache directory.

**Rate limiting**:

The HTTP client automatically retries on HTTP 429 responses, honouring the `Retry-After` header
(+1 s buffer) with up to 3 attempts before returning an error.

Required OAuth scopes (user token):

| Scope | Purpose |
|-------|---------|
| `channels:read` | List public channels |
| `groups:read` | List private channels |
| `im:read` | List DMs |
| `im:write` | Open DM conversations |
| `mpim:read` | List group DMs |
| `channels:history` | Read public channel messages |
| `groups:history` | Read private channel messages |
| `im:history` | Read DM messages |
| `mpim:history` | Read group DM messages |
| `chat:write` | Post messages |
| `files:write` | Upload files |
| `search:read` | Search messages |
| `users:read` | Resolve user names and profiles |

### 4.6 internal/output/

Renders results to stdout.

- Default: ANSI color-formatted, human-readable.
- `--json` flag: raw JSON (suitable for piping to `jq`).
- `--no-color` flag: plain text (for non-TTY environments).

Auto-detects TTY; disables color automatically when stdout is not a terminal.

---

## 5. Command Reference

### Auth

```
scli auth login  [--workspace <name>]   Opens browser for OAuth; saves token
scli auth logout [--workspace <name>]   Removes token from storage
scli auth list                          Lists authenticated workspaces
```

### Workspace

```
scli workspace list                     Lists configured workspaces
scli workspace use <name>               Sets default workspace
```

### Channel

```
scli channel list                       Lists all visible channels (public and private)
scli channel joined                     Lists channels the user is a member of
scli channel read <channel>             Reads recent messages
  [--limit N]                           Number of messages (default: 20)
  [--unread]                            Only show messages since last read
  [--thread <timestamp>]                Show a specific thread
scli channel info <channel>             Shows detailed channel information
scli channel search <query>             Searches channels by name or purpose
scli channel export <channel>           Exports full message history as JSON
  [--output <path>]                     Output file ("-" or omit for stdout)
  [--start <RFC3339>]                   Export messages after this time
  [--end <RFC3339>]                     Export messages before this time
  [--save-dir <path>]                   Download attached files to this directory
```

### Post

```
scli post <channel> [message]           Posts a message
  [--thread <timestamp>]                Reply in a thread
  [--file <path>]                       Attach a file
  [--blocks <json>]                     Post using Block Kit JSON (inline string)
  [--blocks-file <path>]                Post using Block Kit JSON from a file (- for stdin)
```

Note: `[message]` is optional when `--blocks` or `--blocks-file` is provided
(used as the notification fallback text).

### DM

```
scli dm list                            Lists DM conversations
scli dm read <user>                     Reads recent DMs with a user
  [--limit N]
scli dm send <user> <message>           Sends a DM
  [--thread <timestamp>]
```

### Unread

```
scli unread                             Shows unread message counts across all channels and DMs
  [--workspace <name>]
```

### Search

```
scli search <query>                     Searches messages
  [--in <channel>]                      Scope to a channel
  [--limit N]
```

### User

```
scli user list                          Lists workspace members
scli user info <user>                   Shows detailed profile information for a user
scli user search <query>                Searches users by name or display name
```

### Cache

```
scli cache clear                        Removes cached channel and user data for the current workspace
```

---

## 6. Channel / User Resolution

When a command accepts `<channel>` or `<user>`, `scli` resolves as follows:

- If the argument starts with `C`, `G`, `D`, or `U` (Slack ID prefix) → used as-is.
- If the argument starts with `#` → strip `#` and look up by channel name.
- If the argument starts with `@` → strip `@` and look up by username/display name.
- Otherwise → attempt name lookup; error if ambiguous or not found.

---

## 7. Directory Layout

```
scli/
  cmd/                  CLI entry points (cobra)
  internal/
    auth/               OAuth flow
    config/             Configuration & token resolution
    keychain/           OS keychain abstraction
    slack/              Slack API client (includes disk/memory cache)
    output/             Formatter (color / JSON)
  docs/
    design/             Design documents (English)
    ja/                 Japanese translations
  scripts/
    hooks/              Git hooks (pre-commit, pre-push)
  Makefile
  go.mod
  go.sum
  CHANGELOG.md
  CLAUDE.md
```

---

## 8. Error Handling

- API errors are surfaced with the Slack error code and a human-readable message.
- Network errors suggest checking connectivity or token validity.
- HTTP 429 (rate limit) errors are retried automatically; an error is returned only after all retries are exhausted.
- All errors exit with a non-zero status code (suitable for scripting).

---

## 9. Out of Scope (v1)

- Reactions
- Real-time event streaming (WebSocket / Events API)
- Message editing / deletion
- Slash commands
- Interactive components (modals, buttons)

---

*Japanese translation: `docs/ja/design/overview.md`*
