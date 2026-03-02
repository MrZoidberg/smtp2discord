PORT ?= 25
HOST ?= localhost
FROM ?= test@example.com
TO ?= webhook@example.com
SUBJECT ?= smtp2discord test
BODY ?= This is a test message sent via make smtp-test

.PHONY: lint smtp-test

lint:
	golangci-lint run

smtp-test:
	@{ \
		printf 'HELO localhost\r\n'; \
		printf 'MAIL FROM:<$(FROM)>\r\n'; \
		printf 'RCPT TO:<$(TO)>\r\n'; \
		printf 'DATA\r\n'; \
		printf 'Subject: $(SUBJECT)\r\n'; \
		printf '\r\n'; \
		printf '$(BODY)\r\n'; \
		printf '.\r\n'; \
		printf 'QUIT\r\n'; \
	} | nc $(HOST) $(PORT)

docker-build:
	docker build -f Dockerfile.dev -t smtp2discord-dev .

docker-run:
	docker run -p 25:25 smtp2discord-dev --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api