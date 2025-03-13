build-ci:
	GOOS=windows GOARCH=amd64 go build -o bin/signals-windows-amd64.exe .
	GOOS=linux GOARCH=amd64 go build -o bin/signals-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o bin/signals-linux-arm64 .
	GOOS=darwin GOARCH=arm64 go build -o bin/signals-darwin-arm64 .
	cp .env.example bin/.env.local