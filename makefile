.PHONY: lint smtp-test smtps-test smpts-test show-auth-token

lint:
	golangci-lint run

docker-build:
	docker build -f Dockerfile.dev -t smtp2discord-dev .

docker-run:
	docker run -p 25:25 smtp2discord-dev --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api