# Google Integration — Setup Guide

This document explains how tocli connects to your Google account and how developers can build the binary with embedded credentials.

---

## For end users

If you received a pre-built binary of tocli, **no manual setup is required**.

On the first run, tocli will:

1. Print a short banner and open your **default browser** automatically.
2. Show a Google sign-in page asking for permission to access your **Calendar** (read-only) and **Tasks** (read + write).
3. After you approve, the browser shows a success page — you can close it and return to the terminal.
4. tocli starts immediately with your real data.

Your authorization token is saved at `~/.config/tocli/token.json` (permissions `0600`).  
Subsequent runs are fully silent — the token is refreshed automatically in the background.

### Revoking access

To disconnect tocli from your Google account:

```bash
# Remove the local token
rm ~/.config/tocli/token.json

# Optional: revoke the authorization on Google's side
# https://myaccount.google.com/permissions
```

### Flags

| Flag | Description |
|------|-------------|
| `./tocli -offline` | Skip Google entirely, use demo mock data |
| `./tocli -sync` | Test Google auth and exit (no TUI) |

---

## For developers

To build tocli with working Google integration you need OAuth 2.0 client credentials from the Google Cloud Console.

### 1. Create a Google Cloud project

1. Go to [console.cloud.google.com](https://console.cloud.google.com) and create a new project (or select an existing one).
2. Navigate to **APIs & Services → Library** and enable:
   - **Google Calendar API**
   - **Tasks API**

### 2. Create OAuth credentials

1. Go to **APIs & Services → Credentials → Create Credentials → OAuth client ID**.
2. Select **Desktop app** as the application type.
3. Give it any name (e.g. `tocli-dev`).
4. Click **Create**. Note the **Client ID** and **Client Secret** shown.

> **Note:** For a Desktop app OAuth client, the client secret is embedded in the binary.  
> This is expected and accepted by Google — the secret is not truly secret for installed apps.

### 3. Configure the OAuth consent screen

1. Go to **APIs & Services → OAuth consent screen**.
2. Set user type to **External** (or Internal if this is a Workspace org app).
3. Fill in App name, user support email, and developer contact email.
4. Under **Scopes** add:
   - `https://www.googleapis.com/auth/calendar.readonly`
   - `https://www.googleapis.com/auth/tasks`
5. Under **Test users**, add your own Google account (while the app is in testing mode).
6. Save.

### 4. Build with embedded credentials

```bash
go build \
  -ldflags "-X 'tocli/internal/adapter/google.clientID=YOUR_CLIENT_ID' \
            -X 'tocli/internal/adapter/google.clientSecret=YOUR_CLIENT_SECRET'" \
  -o tocli .
```

Replace `YOUR_CLIENT_ID` and `YOUR_CLIENT_SECRET` with the values from step 2.

You can store them in environment variables to avoid pasting in the shell history:

```bash
export TOCLI_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export TOCLI_CLIENT_SECRET="your-client-secret"

go build \
  -ldflags "-X 'tocli/internal/adapter/google.clientID=${TOCLI_CLIENT_ID}' \
            -X 'tocli/internal/adapter/google.clientSecret=${TOCLI_CLIENT_SECRET}'" \
  -o tocli .
```

### 5. Development without ldflags (credentials.json fallback)

If you don't want to use `-ldflags` during active development, you can also place a `credentials.json` file at `~/.config/tocli/credentials.json`.

To get this file:
1. On the Credentials page of your project, click the download icon next to your OAuth client.
2. Save the file to `~/.config/tocli/credentials.json`.

```bash
mkdir -p ~/.config/tocli
mv ~/Downloads/client_secret_*.json ~/.config/tocli/credentials.json
```

Then `go run .` will pick it up automatically.

---

## How the OAuth flow works

```
tocli start
    │
    ├─ token.json exists and valid?
    │       └─ YES → start TUI immediately (silent)
    │
    ├─ token.json exists but expired?
    │       └─ has refresh_token → auto-refresh → start TUI (silent)
    │
    └─ no token / refresh failed
            │
            ├─ start local HTTP server on 127.0.0.1:8085
            ├─ build Google OAuth URL (offline access + force consent)
            ├─ open browser automatically (xdg-open / open / rundll32)
            ├─ user signs in and approves
            ├─ Google redirects to http://127.0.0.1:8085/callback?code=...
            ├─ exchange code for access + refresh tokens
            ├─ save to ~/.config/tocli/token.json (chmod 0600)
            └─ start TUI
```

## File locations

| File | Path | Purpose |
|------|------|---------|
| Token | `~/.config/tocli/token.json` | OAuth access + refresh token (auto-managed) |
| Credentials (dev only) | `~/.config/tocli/credentials.json` | OAuth client ID/secret (only needed without ldflags) |

Override the credentials path with the `TOC_GOOGLE_CREDENTIALS` environment variable.  
Override the config directory base with `XDG_CONFIG_HOME`.

## Scopes requested

| Scope | Reason |
|-------|--------|
| `calendar.readonly` | Read today's events from your primary calendar |
| `tasks` | List, create, complete and reopen tasks |

tocli never writes to your calendar.
