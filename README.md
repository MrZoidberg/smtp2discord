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
  - `docker run -p 25:25 ghcr.io/MrZoidberg/smtp2discord:latest --smtp-user=myuser --smtp-pass=mypass --webhook=https://discord.com/api/webhooks/<ID>/<TOKEN>`

Required/validation rules:

- `--webhook` is required.
- `--smtp-user` and `--smtp-pass` must be provided together (or both omitted).

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
  --smtp-pass=mypass
  --webhook=https://discord.com/api/webhooks/<ID>/<TOKEN>
```

## Native usage

- `smtp2discord --listen=:25 --webhook=http://localhost:8080/api/smtp-hook`
- `smtp2discord --listen=:25 --smtp-user=myuser --smtp-pass=mypass --webhook=http://localhost:8080/api/smtp-hook`
- `smtp2discord --help`

## SMTP AUTH PLAIN

If `--smtp-user` and `--smtp-pass` are provided, clients must authenticate before `MAIL FROM`.

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

Original repo from @alash3al.
Thanks to @aranajuan.


