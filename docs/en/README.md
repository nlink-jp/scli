# scli

A Slack client that runs in your terminal. It operates as a user, not a bot.
Read and write channels, send messages, search, and manage DMs from the terminal.

## Features

- Read channel and DM messages (with thread expansion)
- Post messages and reply to threads (`\n` inserts a newline)
- Post rich messages with [Block Kit](https://api.slack.com/block-kit) JSON (file, stdin, or inline)
- Upload files to channels
- List unread channels and DMs
- Search messages across the whole workspace
- Multiple workspace support
- Color output / `--json` / `--no-color` flags
- Tokens managed via the OS keychain, environment variables, or a `.env` file

## Installation

### Build from source

```sh
git clone https://github.com/nlink-jp/scli.git
cd scli
make build          # build for the current platform → dist/scli
make build-all      # cross-compile for all target platforms
```

Requirements: Go 1.26 or later, `make`

### First-time setup

See [docs/en/setup.md](setup.md) for how to create a Slack app and authenticate.

```sh
scli auth login
```

## Commands

| Command | Description |
|---------|------|
| `scli auth login` | Authenticate with Slack (OAuth 2.0 PKCE) |
| `scli auth logout` | Delete stored credentials |
| `scli auth list` | List authenticated workspaces |
| `scli channel list` | List visible channels (whether joined or not) |
| `scli channel joined` | List channels you have joined |
| `scli channel read <channel>` | Read messages from a channel |
| `scli dm list` | List open DM conversations |
| `scli dm read <user>` | Read a DM |
| `scli dm send <user> <message>` | Send a DM |
| `scli post <channel> [message]` | Post a message to a channel |
| `scli search <query>` | Search messages |
| `scli unread` | Show unread channels and DMs |
| `scli user list` | List workspace members |
| `scli workspace list` | List configured workspaces |
| `scli workspace use <name>` | Switch the default workspace |

### Common flags

```
--workspace, -w   Workspace name (default: "default")
--json            Output in JSON format
--no-color        Disable ANSI color codes
```

### channel read / dm read options

```
-n, --limit N     Number of messages to fetch (default: 20)
--unread          Show only messages since last read
--thread <ts>     Show the thread at the given timestamp
```

### post options

```
--file <path>         Attach a file
--thread <ts>         Reply in a thread
--blocks <json>       Block Kit JSON array (inline string)
--blocks-file <path>  Block Kit JSON read from a file ("-" means stdin)
```

When `--blocks` or `--blocks-file` is used, `[message]` becomes the notification fallback text and may be omitted.
The two flags cannot be used together.

#### Block Kit examples

```sh
# Inline JSON
scli post '#general' 'Hello' --blocks '[{"type":"section","text":{"type":"mrkdwn","text":"*Hello*"}}]'

# Read from a file
scli post '#general' 'Hello' --blocks-file blocks.json

# Read from stdin (piped from md-to-slack or similar)
md-to-slack input.md | scli post '#general' 'Hello' --blocks-file -

# Without fallback text (blocks only)
md-to-slack input.md | scli post '#general' --blocks-file -
```

### search options

```
-n, --limit N     Maximum number of results (default: 20)
--asc             Sort oldest first (default: newest first)
```

## Multiple workspaces

```sh
scli auth login --workspace personal
scli auth login --workspace work
scli workspace use work
scli channel read #general --workspace personal
```

## License

MIT — see [LICENSE](../../LICENSE)
