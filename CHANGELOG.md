# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [1.7.1] - 2026-04-24

### Fixed

- **File download in channel export** — Downloaded files contained HTML instead of actual content. Go's default `http.Client` strips the `Authorization` header on cross-domain redirects; Slack's `url_private_download` redirects to a CDN on a different origin, causing an HTML login page to be saved. Fixed by preserving the header across redirect hops.

---

## [1.7.0] - 2026-04-07

### Added

- **`channel joined`** — New command that lists only the channels the authenticated user is a member of. Uses `users.conversations` instead of `conversations.list`, so the full workspace channel list is never fetched (more efficient). Results are cached separately under `joined_channels.json`.

### Fixed

- **`channel list`** — Now returns all visible channels regardless of membership, as originally intended. Previously the `IsMember` filter incorrectly excluded channels the user had not joined.

---

## [1.6.0] - 2026-04-06

### Added

- **Attachments in channel export** — Legacy rich attachments (URL unfurls, bot cards) are now included in the export JSON as `attachments` array with `fallback`, `color`, `title`, `text`, `fields`, `footer`, and `image_url`.
- **Blocks in channel export** — Block Kit payloads are preserved as raw JSON in the `blocks` field.
- **`docs/EXPORT_FORMAT.md`** — Shared export schema specification covering scat, stail, and scli.

### Fixed

- **`thread_timestamp_unix`** — Now set directly from Slack API's `thread_ts` field, matching stail's behavior. Thread parents include the field (previously omitted).
- **File download error handling** — Download failures during export now log a warning and continue instead of aborting the entire export.

## [1.5.4] - 2026-03-31

### Fixed
- `post --file --blocks` now returns error (mutually exclusive)
- `unwrapBlocksObject` correctly handles `{"blocks": null}` (returns input unchanged)
- PostResult JSON parse edge case with null blocks

### Added
- Tests for PostResult (JSON and text output)
- Tests for unwrapBlocksObject edge cases (null, empty array)

## [1.5.3] - 2026-03-31

### Added
- `post --file` now supports `--thread` for uploading files as thread replies
- `post --file` no longer requires message argument (file-only upload)
- `UploadFile` passes `thread_ts` to `files.completeUploadExternal`

## [1.5.2] - 2026-03-31

### Added
- `post` and `dm send` output `{"ts":"...","channel":"..."}` with `--json` flag
- In text mode, prints `Message posted (ts: ...)` as before
- Enables pipeline capture of ts for thread replies and file attachments

## [1.5.1] - 2026-03-31

### Fixed
- Accept `{"blocks": [...]}` wrapper format (e.g. md-to-slack output) in addition to bare JSON arrays
- Auto-unwrap to extract the blocks array before sending to Slack API

### Changed
- Refactor blocks JSON resolution into shared `loadBlocksJSON()` helper (used by both `post` and `dm send`)

## [1.5.0] - 2026-03-31

### Added
- `dm send` now supports `--blocks` and `--blocks-file` flags for Block Kit messages
- Message argument is optional when blocks are provided (same behavior as `post`)

## [1.4.0] - 2026-03-30

### Added

- **`scli workspace rename <old> <new>`** — Rename a workspace, atomically updating config.json, OS keychain token, cache directory, and default workspace pointer. Prevents keychain data orphaning from manual config edits.
- **Japanese README** (`README.ja.md`) — Full Japanese translation of README.md.


## [1.3.2] - 2026-03-30

### Fixed

- **Version display** — `scli --version` now shows the correct version instead of `dev`. Makefile LDFLAGS module path was still referencing the pre-migration path (`magifd2`).

## [1.3.1] - 2026-03-28

### Internal

- Updated Go module path and repository URLs to `github.com/nlink-jp/scli`.
- Added macOS-specific entries to `.gitignore`.

## [1.3.0] - 2026-03-27

### Added

- **`scli channel export <channel>`**: Export full channel message history as JSON,
  compatible with scat and stail export format.
  - Threads are fully expanded: each reply appears immediately after its parent
    with `is_reply=true` and `thread_timestamp_unix` set.
  - `--output <path>` — write to file (`-` or omit for stdout)
  - `--start <RFC3339>` / `--end <RFC3339>` — time range filtering
  - `--save-dir <path>` — download attached files; `local_path` is set in output JSON
  - New OAuth scope required: `files:read` (for file downloads)

## [1.2.0] - 2026-03-27

### Added

- **HTTP 429 retry**: The Slack API client now automatically retries rate-limited requests,
  honouring the `Retry-After` header (+1 s buffer) with up to 3 attempts before returning an error.
- **Disk cache for channels and users**: `channel list` and `user list` API calls are cached to disk
  with a 1-hour TTL at `~/.config/scli/cache/<workspace>/`. Eliminates repeated full-list fetches
  in large workspaces. Cache is workspace-specific to prevent cross-workspace contamination.
- **In-memory user cache**: `GetUser` additionally caches results in-memory for the duration of
  a single process, avoiding redundant API calls during username resolution.
- **`scli channel info <channel>`**: Shows detailed channel information (topic, purpose, member count,
  creator, creation date, visibility flags).
- **`scli channel search <query>`**: Searches joined channels by name or purpose (uses disk cache).
- **`scli user info <user>`**: Shows detailed user profile (display name, real name, title, email,
  phone, status, timezone, user ID).
- **`scli user search <query>`**: Searches workspace members by name or display name (uses disk cache).
- **`scli cache clear`**: Removes cached channel and user data for the current workspace.

## [1.1.0] - 2026-03-23

### Added

- **Block Kit support for `post` command**: Post rich messages using Slack's Block Kit JSON format.
  - `--blocks <json>` — inline JSON array string
  - `--blocks-file <path>` — read JSON from a file (`-` reads from stdin)
  - `[message]` argument is now optional when blocks are provided (used as notification fallback text)
  - Designed for use with tools like [md-to-slack](https://github.com/magifd2/md-to-slack)

## [1.0.1] - 2026-03-20

### Fixed

- **`.env` priority order**: Current directory `.env` now correctly takes precedence over `~/.config/scli/.env`, matching the documented token resolution chain.
- **No-op parameter assignment**: Removed redundant `params.Set("limit", params.Get("limit"))` in the Slack API client.

## [1.0.0] - 2026-03-20

### Added

**Authentication**
- OAuth 2.0 PKCE flow with local HTTPS callback server (self-signed certificate)
- `--manual` flag for headless environments (prints auth URL, reads redirect URL from stdin)
- Token storage: OS keychain → environment variables → `.env` files → `config.json`
- Multiple workspace support (`--workspace` flag, `scli workspace use`)

**Channel commands**
- `scli channel list` — list channels you are a member of
- `scli channel read <channel>` — read messages with thread expansion
  - `--limit`, `--unread`, `--thread` flags
  - Thread replies automatically nested under parent messages

**Direct message commands**
- `scli dm list` — list open DM conversations (includes bots/apps)
- `scli dm read <user>` — read DM messages
- `scli dm send <user> <message>` — send a DM

**Post command**
- `scli post <channel> <message>` — post a message
  - `--thread` flag for thread replies
  - `--file` flag for file attachments (uses `files.getUploadURLExternal` API)
  - `\n` and `\t` escape sequences in message text

**Search command**
- `scli search <query>` — search messages across the workspace
  - `--limit` and `--asc` flags

**Unread summary**
- `scli unread` — show channels and DMs with unread messages
  - Falls back to `conversations.history` for channels where `unread_count` is inaccurate (bot/webhook-only channels)

**User command**
- `scli user list` — list workspace members

**Output**
- Color-formatted text output with auto TTY detection
- `--json` flag for machine-readable output
- `--no-color` flag

**Infrastructure**
- Cross-compilation for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`
- `make check`: lint (`golangci-lint`), test, build-all, security scan (`govulncheck`)
- Git pre-commit and pre-push hooks
- Design documents and setup guide in English and Japanese

[1.0.0]: https://github.com/nlink-jp/scli/releases/tag/v1.0.0
