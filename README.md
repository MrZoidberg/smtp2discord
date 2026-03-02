SMTP2Discord (email-to-Discordwebhook)
========================
smtp2Discord is a simple smtp server that resends the incoming email to the configured web endpoint (webhook) as a Discord webhook http post request.

Dev 
===
- `go mod vendor`
- `go build`

Dev with Docker
==============
Locally :
- `go mod vendor`
- `docker build -f Dockerfile.dev -t smtp2discord-dev .`
- `docker run -p 25:25 smtp2discord-dev --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api`

Or build it as it comes from the repo :
- `docker build -t smtp2discord .`
- `docker run -p 25:25 smtp2discord --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api`

The `timeout` options are of course optional but make it easier to test in local with `telnet localhost 25`
Here is a telnet example payload : 
```
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

Docker (production)
=====
**Docker images arn't available online for now**
**See "Dev with Docker" above**
- `docker run -p 25:25 smtp2discord --webhook=http://some.hook/api`

Docker Compose
=====
```
version: '3.1'
services:
  smtp2discord:
    container_name: smtp2discord
    image: MrZoidberg/smtp2discord
    command: ' -webhook https://discord.com/api/webhooks/<ID>/<Pass>'
    ports:
      - '25:25'
    restart: unless-stopped
```


Native usage
=====
`smtp2discord --listen=:25 --webhook=http://localhost:8080/api/smtp-hook`
`smtp2discord --listen=:25 --smtp-user=myuser --smtp-pass=mypass --webhook=http://localhost:8080/api/smtp-hook`
`smtp2discord --help`

SMTP AUTH PLAIN
=====
If `--smtp-user` and `--smtp-pass` are provided, clients must authenticate before `MAIL FROM`.

Example session:
```
EHLO localhost
AUTH PLAIN AG15dXNlcgBteXBhc3M=
MAIL FROM:<email@from.com>
RCPT TO:<youremail@example.com>
DATA
your mail content
.
```

Contribution
============
Original repo from @alash3al
Thanks to @aranajuan


