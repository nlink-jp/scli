# scli

A terminal-based Slack client that operates as a user (not a bot).
Read channels, send messages, search, and manage DMs — all without leaving your terminal.

## Features

- Read channel and DM messages with thread expansion
- Post messages and reply to threads (supports `\n` for newlines)
- Post [Block Kit](https://api.slack.com/block-kit) messages from JSON (file, stdin, or inline)
- Upload files to channels
- List unread channels and DMs
- Search messages across the workspace
- Multiple workspace support
- Color output with `--json` and `--no-color` flags
- Token storage via OS keychain, environment variables, or `.env` files

## Installation

### Build from source

```sh
git clone https://github.com/nlink-jp/scli.git
cd scli
make build          # builds for the current platform → dist/scli
make build-all      # cross-compiles for all target platforms
```

Requirements: Go 1.26+, `make`

### First-time setup

See [docs/setup.md](docs/setup.md) for step-by-step instructions on creating a Slack app and authenticating.

```sh
scli auth login
```

## Commands

| Command | Description |
|---------|-------------|
| `scli auth login` | Authenticate with Slack (OAuth 2.0 PKCE) |
| `scli auth logout` | Remove stored credentials |
| `scli auth list` | Show authenticated workspaces |
| `scli channel list` | List channels you are a member of |
| `scli channel read <channel>` | Read messages from a channel |
| `scli dm list` | List open DM conversations |
| `scli dm read <user>` | Read DM messages |
| `scli dm send <user> <message>` | Send a direct message |
| `scli post <channel> [message]` | Post a message to a channel |
| `scli search <query>` | Search messages in the workspace |
| `scli unread` | Show channels and DMs with unread messages |
| `scli user list` | List workspace members |
| `scli workspace list` | List configured workspaces |
| `scli workspace use <name>` | Switch default workspace |
| `scli workspace rename <old> <new>` | Rename a workspace (config + keychain + cache) |

### Common flags

```
--workspace, -w   Workspace name (default: "default")
--json            Output raw JSON
--no-color        Disable ANSI color codes
```

### channel read / dm read options

```
-n, --limit N     Number of messages to fetch (default: 20)
--unread          Show only messages since last read
--thread <ts>     Show a specific thread by message timestamp
```

### post options

```
--file <path>         Attach a file to the message
--thread <ts>         Reply in a thread
--blocks <json>       Block Kit JSON array (inline string)
--blocks-file <path>  Block Kit JSON from a file ("-" reads from stdin)
```

When `--blocks` or `--blocks-file` is used, `[message]` becomes the notification fallback text
and may be omitted. The two flags are mutually exclusive.

#### Block Kit examples

```sh
# Inline JSON
scli post '#general' 'Hello' --blocks '[{"type":"section","text":{"type":"mrkdwn","text":"*Hello*"}}]'

# From a file
scli post '#general' 'Hello' --blocks-file blocks.json

# From stdin (e.g. piped from md-to-slack)
md-to-slack input.md | scli post '#general' 'Hello' --blocks-file -

# Without fallback text (blocks only)
md-to-slack input.md | scli post '#general' --blocks-file -
```

### search options

```
-n, --limit N     Maximum number of results (default: 20)
--asc             Sort results oldest first (default: newest first)
```

## Multiple workspaces

```sh
scli auth login --workspace personal
scli auth login --workspace work
scli workspace use work
scli channel read #general --workspace personal
```

### Renaming a workspace

```sh
scli workspace rename personal my-personal
```

> **Warning:** Do not rename workspaces by editing `config.json` directly.
> The authentication token is stored in the OS keychain under the workspace
> name. If you change the name in the config file without updating the
> keychain, scli will be unable to find the token and authentication will
> break. Always use `scli workspace rename` to keep config, keychain, and
> cache in sync.

## License

MIT — see [LICENSE](LICENSE)
