build-ci:
	go build -o bin/signals-$(SUFFIX) .
	cp .env.example bin/.env.local