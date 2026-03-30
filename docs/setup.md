# scli — Setup Guide

## Prerequisites

- Go 1.26 or later
- GNU Make
- A Slack workspace where you have permission to install apps

---

## Step 1: Create a Slack App

### 1-1. Open the Slack API portal

Go to <https://api.slack.com/apps> and click **Create New App**.
Choose **From scratch**, then enter:

- **App Name**: `scli` (or any name you prefer)
- **Pick a workspace**: select your workspace

### 1-2. Configure OAuth scopes

In the left sidebar, go to **OAuth & Permissions**.

Scroll down to **User Token Scopes** (not Bot Token Scopes) and add:

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
| `files:read` | Download attached files (`channel export --save-dir`) |
| `search:read` | Search messages |
| `users:read` | Resolve user names and profiles |

### 1-3. Set the redirect URL

Still on the **OAuth & Permissions** page, under **Redirect URLs**, add:

```
https://localhost:7777/callback
```

This is the local HTTPS URL that `scli auth login` listens on during the OAuth flow.
`scli` generates a self-signed certificate at runtime — your browser will show a
**certificate warning**; accept it to complete the login.

> **Note**: Slack requires HTTPS for all redirect URIs. `scli` handles this by spinning up
> a local HTTPS server with a short-lived self-signed certificate. No external tooling
> (e.g., mkcert) is required.

### 1-4. Note your credentials

Go to **Basic Information** and note:

- **Client ID**
- **Client Secret**

You will need these in Step 3.

---

## Step 2: Build scli

```bash
# Clone the repository
git clone https://github.com/<your-org>/scli.git
cd scli

# Install Git hooks and verify toolchain
make setup

# Build for your current platform
make build

# (Optional) Cross-compile for all platforms
make build-all
```

The binary is placed in `./dist/`.

---

## Step 3: Configure scli

### Option A-1 — Interactive login (recommended)

```bash
export SLACK_CLIENT_ID=<your-client-id>
export SLACK_CLIENT_SECRET=<your-client-secret>
scli auth login --workspace myteam
```

Opens your browser for the Slack OAuth flow. `scli` starts a local HTTPS server
on port 7777 using a self-signed certificate to receive the callback automatically.

**When the browser shows a certificate warning:**
- Chrome/Edge: click **Advanced** → **Proceed to localhost (unsafe)**
- Firefox: click **Advanced** → **Accept the Risk and Continue**
- Safari: click **Visit Website**

On success, the token is stored securely in your OS keychain.

### Option A-2 — Manual login (headless / browser flow unavailable)

```bash
export SLACK_CLIENT_ID=<your-client-id>
export SLACK_CLIENT_SECRET=<your-client-secret>
scli auth login --workspace myteam --manual
```

`scli` prints the authorization URL. Open it in your browser, authorize the app.
The browser will redirect to `https://localhost:7777/callback?code=xxxxx` — the page
will fail to load (expected). Copy the full URL from the address bar and paste it
at the prompt.

### Option B — Environment variable

```bash
export SLACK_TOKEN=xoxp-...        # default workspace
export SLACK_TOKEN_MYTEAM=xoxp-... # named workspace
```

### Option C — Config file

Edit `~/.config/scli/config.json`:

```json
{
  "default_workspace": "myteam",
  "workspaces": {
    "myteam": {
      "token": "xoxp-...",
      "team_id": "T012AB3C4",
      "user_id": "U012AB3C4"
    }
  }
}
```

> **Security note**: Option A (OS keychain) is strongly recommended.
> Config file and environment variable approaches store the token in plaintext
> and should only be used in environments where the keychain is unavailable
> (e.g., CI pipelines, containers).

---

## Step 4: Verify

```bash
scli workspace list
scli channel list
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `invalid_auth` | Token expired or revoked | Run `scli auth login` again |
| `missing_scope` | Scope not added to the Slack app | Add the scope in the API portal and reinstall the app |
| Browser does not open | Headless environment | Copy the URL printed to stdout and open it manually |
| Keychain prompt appears repeatedly | OS keychain locked | Unlock your keychain (macOS: Keychain Access app) |
| `no token found for workspace` after renaming | Workspace renamed via config.json edit | Run `scli auth login --workspace <new-name>` to re-authenticate, or use `scli workspace rename` for safe renaming |

---

*Japanese translation: `docs/ja/setup.md`*
