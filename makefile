PORT ?= 2025
SMTPS_PORT ?= 645
HOST ?= smtp.internal.mmerk.online
SMTPS_SERVERNAME ?= smtp.mmerk.online
FROM ?= test@example.com
TO ?= webhook@example.com
SUBJECT ?= smtp2discord test
BODY ?= This is a test message sent via make smtp-test
SMTP_USER ?= smtp
SMTP_PASS ?= 4Pw3oB8fmWttcMXB9Hrd

# Force a POSIX shell (important if running `make` from fish).
SHELL := /bin/sh

.PHONY: lint smtp-test smtps-test smpts-test

lint:
	golangci-lint run

smtp-test:
	@command -v curl >/dev/null 2>&1 || { echo "Error: need 'curl' for smtp-test" 1>&2; exit 1; }; \
	printf 'Subject: $(SUBJECT)\r\n\r\n$(BODY)\r\n' | \
		curl -v --show-error --fail \
			--user "$(SMTP_USER):$(SMTP_PASS)" \
			--url "smtp://$(HOST):$(PORT)" \
			--mail-from "$(FROM)" \
			--mail-rcpt "$(TO)" \
			--upload-file -

smtps-test:
	@command -v curl >/dev/null 2>&1 || { echo "Error: need 'curl' for smtps-test" 1>&2; exit 1; }; \
	printf 'Subject: $(SUBJECT)\r\n\r\n$(BODY)\r\n' | \
		curl -v --show-error --fail \
			--user "$(SMTP_USER):$(SMTP_PASS)" \
			--url "smtps://$(SMTPS_SERVERNAME):$(SMTPS_PORT)" \
			--mail-from "$(FROM)" \
			--mail-rcpt "$(TO)" \
			--upload-file -

smpts-test: smtps-test

docker-build:
	docker build -f Dockerfile.dev -t smtp2discord-dev .

docker-run:
	docker run -p 25:25 smtp2discord-dev --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api