# hpphub

HPP Hub CLI — connect [OpenClaw](https://openclaw.ai/) to [HPP Hub](https://hub.hpp.io) with a single command.

```bash
hpphub launch openclaw
```

Like `ollama launch openclaw` connects OpenClaw to local models, `hpphub launch openclaw` connects OpenClaw to HPP's cloud models.

## Quick Start

```bash
# Install (macOS/Linux)
curl -fsSL https://hpp.io/install.sh | bash

# Launch OpenClaw with HPP
hpphub launch openclaw
```

The `launch` command handles everything automatically:

1. Detects OpenClaw (installs if needed)
2. Logs in to HPP Hub (Device Code Flow)
3. Creates an API key
4. Configures OpenClaw with HPP as model provider
5. Starts the OpenClaw gateway

## Commands

```bash
hpphub launch openclaw              # Full setup: install → login → configure → start
hpphub launch openclaw --config     # Configure only, don't start gateway
hpphub launch openclaw --model <m>  # Use a specific model

hpphub login                        # Log in to HPP Hub
hpphub logout                       # Log out
hpphub whoami                       # Show current login status

hpphub models                       # List available models with pricing
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

## Build from Source

Requires Go 1.22+.

```bash
git clone https://github.com/hpp-io/hpphub-cli.git
cd hpphub-cli
go build -o hpphub ./cmd/hpphub/
./hpphub --help
```

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

Once configured, use OpenClaw directly:

```bash
# Connect messaging channels
openclaw configure --section channels

# Stop the gateway
openclaw gateway stop

# Reconfigure HPP model
hpphub launch openclaw --config --model anthropic/claude-sonnet-4-6
```

## License

MIT
