# SMTP2Discord (email-to-Discord webhook)

smtp2discord is a simple SMTP server that resends incoming email to the configured web endpoint (webhook) as a Discord webhook HTTP POST request.

Forwarded message format in Discord:

- Default format is:

```text
<from>: <subject>
<body>
```

- If `From` header is missing, `<from>` falls back to SMTP envelope sender (`MAIL FROM`).
- If one field is missing, the template omits the unnecessary separator automatically.

## Installation

### Quick install (all supported OS)

The installer detects your OS, downloads the correct package, installs the service, and prompts for your Discord webhook URL.

```sh
curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | sudo sh
```

> **Alpine Linux** ships with `doas` instead of `sudo` — replace `sudo` with `doas` in all commands below.
>
> ```sh
> curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | doas sh
> ```

Supply the webhook URL non-interactively (useful for automated provisioning):

```sh
curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | sudo sh -s -- --webhook https://discord.com/api/webhooks/<ID>/<TOKEN>
```

Upgrade an existing installation (config is preserved):

```sh
curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | sudo sh -s -- --upgrade
```

**Supported operating systems:** Ubuntu, Debian, Alpine, Fedora, Amazon Linux 2023 (x86_64 and arm64).

---

### Manual package install

Download the package for your OS and architecture from the [latest release](https://github.com/MrZoidberg/smtp2discord/releases/latest).

#### Ubuntu / Debian

```sh
# Download the .deb (replace version and arch as needed)
curl -LO https://github.com/MrZoidberg/smtp2discord/releases/latest/download/smtp2discord_<version>_linux_amd64.deb

# Install
sudo apt-get install -y ./smtp2discord_*.deb
```

#### Fedora

```sh
curl -LO https://github.com/MrZoidberg/smtp2discord/releases/latest/download/smtp2discord_<version>_linux_amd64.rpm
sudo dnf install -y ./smtp2discord_*.rpm
```

#### Amazon Linux 2023

```sh
curl -LO https://github.com/MrZoidberg/smtp2discord/releases/latest/download/smtp2discord_<version>_linux_amd64.rpm
sudo dnf install -y ./smtp2discord_*.rpm
```

#### Alpine

> **Note:** Alpine Linux ships with `doas` instead of `sudo`. Run commands as root or prefix them with `doas`.

```sh
curl -LO https://github.com/MrZoidberg/smtp2discord/releases/latest/download/smtp2discord_<version>_linux_amd64.apk
doas apk add --allow-untrusted ./smtp2discord_*.apk
```

---

### Service configuration

After installing, open `/etc/default/smtp2discord` in your editor and at minimum set the required webhook URL:

```sh
sudo $EDITOR /etc/default/smtp2discord
```

```sh
# /etc/default/smtp2discord — required
SMTP2DISCORD_WEBHOOK=https://discord.com/api/webhooks/<ID>/<TOKEN>

# Optional overrides (all have built-in defaults):
# SMTP2DISCORD_LISTEN=:smtp          # listen address (default: :smtp  →  port 25)
# SMTP2DISCORD_NAME=smtp2discord     # SMTP banner name
# SMTP2DISCORD_SMTP_USER=myuser      # AUTH PLAIN credentials (both or neither)
# SMTP2DISCORD_SMTP_PASS_HASH='$2y$10$...'  # bcrypt hash (NOT plaintext)
# SMTP2DISCORD_AUTHOR=               # Discord message username
# SMTP2DISCORD_AVATAR_URL=           # Discord avatar URL
# SMTP2DISCORD_MSG_LIMIT=2097152     # max message size in bytes
# SMTP2DISCORD_TIMEOUT_READ=5        # read timeout (seconds)
# SMTP2DISCORD_TIMEOUT_WRITE=5       # write timeout (seconds)
# SMTP2DISCORD_MESSAGE_TEMPLATE_FILE= # path to custom Go template
```

Then start (or restart) the service:

| Init system | Start | Restart | Logs |
|-------------|-------|---------|------|
| systemd (Ubuntu, Debian, Fedora, Amazon Linux) | `sudo systemctl start smtp2discord` | `sudo systemctl restart smtp2discord` | `journalctl -u smtp2discord -f` |
| OpenRC (Alpine) | `doas rc-service smtp2discord start` | `doas rc-service smtp2discord restart` | `doas logread \| grep smtp2discord` |

Enable auto-start on boot (if not already done by the installer):

```sh
# systemd
sudo systemctl enable smtp2discord

# OpenRC (Alpine)
doas rc-update add smtp2discord default
```

---

## Custom message template

You can override message formatting with `--message-template-file`.

Template data fields available in Go template files:

- `{{ .From }}`
- `{{ .Subject }}`
- `{{ .Body }}`

Run with custom template file:

- `smtp2discord --webhook=https://discord.com/api/webhooks/<ID>/<TOKEN> --message-template-file=./my-template.tmpl`

Docker example:

- `docker run -p 25:25 -v $(pwd)/my-template.tmpl:/templates/my-template.tmpl ghcr.io/MrZoidberg/smtp2discord:latest --webhook=https://discord.com/api/webhooks/<ID>/<TOKEN> --message-template-file=/templates/my-template.tmpl`

Example custom template file:

```gotemplate
[FROM] {{ .From }}
[SUBJECT] {{ .Subject }}

{{ .Body }}
```

## Dev

- `go build`

## Dev with Docker

Locally:

- `docker build -f Dockerfile.dev -t smtp2discord-dev .`
- `docker run -p 25:25 smtp2discord-dev --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api`

Note: `Dockerfile` is optimized for GoReleaser `dockers_v2` and copies prebuilt binaries from `$TARGETPLATFORM/...`.

The `timeout` options are optional but make local testing easier with `telnet localhost 25`.

Here is a telnet example payload:

```text
HELO zeus
# smtp answer

MAIL FROM:<email@from.com>
# smtp answer

RCPT TO:<youremail@example.com>
# smtp answer

DATA
your mail content
.
```

## Docker (production)

Docker images are published to GitHub Container Registry (GHCR):

- `docker pull ghcr.io/MrZoidberg/smtp2discord:latest`
- Minimal required args:
  - `docker run -p 25:25 ghcr.io/MrZoidberg/smtp2discord:latest --webhook=https://discord.com/api/webhooks/<ID>/<TOKEN>`
- Full common args:
  - `docker run -p 25:25 ghcr.io/MrZoidberg/smtp2discord:latest --name=smtp2discord --listen=:25 --msglimit=2097152 --timeout.read=5 --timeout.write=5 --author="SMTP Bridge" --avatar-url="https://example.com/bot.png" --webhook=https://discord.com/api/webhooks/<ID>/<TOKEN>`
- With SMTP AUTH PLAIN enabled (both required together):
  - `docker run -p 25:25 ghcr.io/MrZoidberg/smtp2discord:latest --smtp-user=myuser --smtp-pass-hash='$2y$10$...' --webhook=https://discord.com/api/webhooks/<ID>/<TOKEN>`

Required/validation rules:

- `--webhook` is required.
- `--smtp-user` and `--smtp-pass-hash` must be provided together (or both omitted).

## Docker Compose

```yaml
services:
  smtp2discord:
    container_name: smtp2discord
    image: ghcr.io/MrZoidberg/smtp2discord:latest
    command: >-
      --name=smtp2discord
      --listen=:25
      --msglimit=2097152
      --timeout.read=5
      --timeout.write=5
      --webhook=https://discord.com/api/webhooks/<ID>/<TOKEN>
    ports:
      - '25:25'
    restart: unless-stopped
```

SMTP AUTH PLAIN in Compose:

```yaml
command: >-
  --listen=:25
  --smtp-user=myuser
  --smtp-pass-hash=$$2y$$10$$...
  --webhook=https://discord.com/api/webhooks/<ID>/<TOKEN>
```

## Native usage

- `smtp2discord --listen=:25 --webhook=http://localhost:8080/api/smtp-hook`
- `smtp2discord --listen=:25 --smtp-user=myuser --smtp-pass-hash='$2y$10$...' --webhook=http://localhost:8080/api/smtp-hook`
- `smtp2discord --help`

## SMTP AUTH PLAIN

If `--smtp-user` and `--smtp-pass-hash` are provided, clients must authenticate before `MAIL FROM`.

### Generate password + bcrypt hash

1) Generate a strong password (pick one):

```sh
# OpenSSL
openssl rand -base64 32 | tr -d '\n'
```

```sh
# Python (no external deps)
python3 -c 'import secrets; print(secrets.token_urlsafe(32))'
```

2) Generate a bcrypt hash for `--smtp-pass-hash` / `SMTP2DISCORD_SMTP_PASS_HASH` (example, cost 10):

```sh
htpasswd -bnBC 10 "" "mypass" | tr -d ':\n'
```

If you don’t have `htpasswd` installed, you can generate the same hash using Docker:

```sh
docker run --rm httpd:2.4-alpine htpasswd -bnBC 10 "" "mypass" | tr -d ':\n'
```

Set it in your service config (recommended quoting because bcrypt hashes contain `$`):

```sh
SMTP2DISCORD_SMTP_USER=myuser
SMTP2DISCORD_SMTP_PASS_HASH='$2y$10$...'
```

Example session:

```text
EHLO localhost
AUTH PLAIN AG15dXNlcgBteXBhc3M=
MAIL FROM:<email@from.com>
RCPT TO:<youremail@example.com>
DATA
your mail content
.
```

## Contribution

Original repos from @alash3al and @donserdal


