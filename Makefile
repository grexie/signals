build-ci:
	GOOS=windows GOARCH=amd64 go build -o bin/signals-windows-amd64.exe .
	GOOS=linux GOARCH=amd64 go build -o bin/signals-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -o bin/signals-linux-arm64 .
	GOOS=darwin GOARCH=arm64 go build -o bin/signals-darwin-arm64 .
	cp .env.example bin/.env.local
	cp README.dist.txt bin/README
	cd bin && tar cvzf signals-windows-amd64.tar.gz signals-windows-amd64.exe .env.local README
	cd bin && tar cvzf signals-linux-amd64.tar.gz signals-linux-amd64 .env.local README
	cd bin && tar cvzf signals-linux-arm64.tar.gz signals-linux-arm64 .env.local README
	cd bin && tar cvzf signals-darwin-arm64.tar.gz signals-darwin-arm64 .env.local README