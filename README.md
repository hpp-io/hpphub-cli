# hpphub

HPP Hub CLI — connect [OpenClaw](https://openclaw.ai/) to [HPP Hub](https://hub.hpp.io) with a single command.

```bash
hpphub launch openclaw
```

Like `ollama launch openclaw` connects OpenClaw to local models, `hpphub launch openclaw` connects OpenClaw to HPP's cloud models.

## Install

### One-line install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/hpp-io/hpphub-cli/main/install.sh | sudo bash
```

This detects your OS and architecture, downloads the latest binary from [GitHub Releases](https://github.com/hpp-io/hpphub-cli/releases), and installs it to `/usr/local/bin/hpphub`.

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/hpp-io/hpphub-cli/main/install.ps1 | iex
```

> **Note:** If you see an Execution Policy error, run this first:
> ```powershell
> Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned
> ```
> After installation, **restart the terminal** for PATH to take effect.

On Windows, the OpenClaw gateway runs in foreground mode. Keep the terminal open while using Telegram/WhatsApp channels.

### Windows (WSL2) — Recommended

WSL2 provides the best experience on Windows (background gateway, full compatibility):

```bash
curl -fsSL https://raw.githubusercontent.com/hpp-io/hpphub-cli/main/install.sh | sudo bash
```

### Build from source (Go 1.24+)

```bash
git clone https://github.com/hpp-io/hpphub-cli.git
cd hpphub-cli
go build -o hpphub ./cmd/hpphub/
```

## Getting Started

### For New Users (no HPP Hub account)

If you don't have an HPP Hub account yet, `hpphub` will guide you through the entire process:

```bash
$ hpphub launch openclaw
```

**Step 1 — OpenClaw Installation**

```
Checking OpenClaw installation...
  ✗ OpenClaw not found
  Install OpenClaw? (Y/n): Y
  Installing OpenClaw...
  ✓ OpenClaw detected
```

If OpenClaw is already installed, this step is skipped automatically.

> OpenClaw requires Node.js 22+. If Node.js is not installed, the OpenClaw installer will handle it.

**Step 2 — HPP Hub Login**

```
Not logged in. Starting login flow...
  Your code: ABCD-1234
  Browser opened. Enter the code and authorize.
  Waiting for approval...
```

A browser window opens to `hub.hpp.io`. Since you don't have an account:

1. Click **Sign in with Google** (or your preferred login method)
2. Complete the sign-up process:
   - Agree to Terms of Service
   - A wallet and API credentials are automatically created for you
3. After sign-up, you are redirected to the **Authorize Device** page
4. Enter the code shown in your terminal (e.g., `ABCD-1234`)
5. Click **Authorize**

![Authorize Device](https://github.com/user-attachments/assets/d7134077-5e1b-4cb9-b599-bca35dcd97f6)

```
  ✓ Logged in as you@example.com
  ✓ API key saved: ...xxxx
```

**Step 3 — Model Selection**

```
  Available models:
   1. anthropic/claude-sonnet-4-6          ($3.00/$15.00 per M tokens)
   2. openai/gpt-5-mini                    ($0.25/$2.00 per M tokens)
   ...
  Select model (number): 2
```

Or skip with `--model`:

```bash
hpphub launch openclaw --model openai/gpt-5-mini
```

**Step 4 — Done**

```
  ✓ HPP provider configured in OpenClaw
  ✓ OpenClaw gateway running

You're all set! Send a message via Telegram, WhatsApp, or other connected channels.
```

### For Existing Users (already have an HPP Hub account)

```bash
$ hpphub launch openclaw
```

The flow is the same, but faster — no sign-up needed:

1. Browser opens → log in with your existing account
2. Enter the code → Authorize
3. Your existing API key is reused (no duplicate keys)
4. Select a model → OpenClaw configured and running

If you've already run `hpphub launch openclaw` before, your login and API key are cached in `~/.hpphub/config.json`. Running the command again will skip login and go straight to configuration.

## Commands

```bash
hpphub launch openclaw              # Full setup: install → login → configure → start
hpphub launch openclaw --config     # Configure only, don't start gateway
hpphub launch openclaw --model <m>  # Use a specific model

hpphub login                        # Log in to HPP Hub
hpphub logout                       # Log out
hpphub whoami                       # Show current login status

hpphub models                       # List available models with pricing

hpphub setup telegram               # Set up Telegram bot connection

hpphub uninstall                    # Remove hpphub and its configuration
```

## Available Models

After login, `hpphub models` shows all available models:

```
PROVIDER     MODEL                                           INPUT     OUTPUT
──────────────────────────────────────────────────────────────────────────────
anthropic    anthropic/claude-sonnet-4-6                   $3.00/M   $15.00/M
openai       openai/gpt-5-mini                             $0.15/M    $0.60/M
openai       openai/gpt-5                                  $1.25/M   $10.00/M
...
```

## How It Works

```
hpphub launch openclaw
       │
       ├─ OpenClaw installed? → detect or auto-install
       ├─ Logged in? → Device Code Flow (browser-based)
       ├─ API key? → auto-create on login
       ├─ Model? → interactive selection or --model flag
       ├─ Configure OpenClaw → inject HPP provider into ~/.openclaw/openclaw.json
       └─ Start gateway → openclaw gateway start
```

After setup, OpenClaw routes messages through HPP:

```
Telegram/WhatsApp/Slack → OpenClaw → HPP Hub (router.hpp.io) → LLM → response
```

## Authentication

`hpphub login` uses [Device Code Flow (RFC 8628)](https://datatracker.ietf.org/doc/html/rfc8628):

```bash
$ hpphub login
  Your code: ABCD-1234
  Browser opened. Enter the code and authorize.
  Waiting for approval...
  ✓ Logged in as user@example.com
  ✓ API key saved: ...a3f2
```

Works in all environments including WSL and SSH sessions.

Credentials are stored in `~/.hpphub/config.json`.

## Configuration

### CLI config

Stored at `~/.hpphub/config.json` after login:

```json
{
  "api_key": "hpph_...",
  "base_url": "https://router.hpp.io/llm/v1",
  "email": "user@example.com"
}
```

### OpenClaw config

`hpphub launch openclaw` adds HPP as a provider in `~/.openclaw/openclaw.json`:

```json
{
  "models": {
    "mode": "merge",
    "providers": {
      "hpp": {
        "baseUrl": "https://router.hpp.io/llm/v1",
        "apiKey": "hpph_...",
        "api": "openai-completions",
        "models": [...]
      }
    }
  }
}
```

## After Setup

### Connect Telegram

```bash
$ hpphub setup telegram

  To create a Telegram bot:

  1. Open Telegram and talk to @BotFather
  2. Send /newbot and follow the steps
  3. Copy the bot token

  Paste your Telegram bot token: 123456789:ABCdefGHI...
  ✓ Bot token saved

  Your Telegram user ID (or press Enter to skip): 123456789
  ✓ Access restricted to your account

  ✓ Gateway restarted
  ✓ Telegram bot connected!
```

Send a message to your bot in Telegram — it should respond using HPP models.

### Connect Other Channels

```bash
# Interactive setup for WhatsApp, Discord, Slack, Signal, etc.
openclaw configure --section channels
```

### Change Model

```bash
hpphub launch openclaw --config --model anthropic/claude-sonnet-4-6
```

### Manage

```bash
# Check gateway status
openclaw health

# Stop the gateway
openclaw gateway stop

# View logs
tail -f ~/.openclaw/logs/gateway.log

# Re-login with a different account
hpphub logout
hpphub login
```

## License

MIT
