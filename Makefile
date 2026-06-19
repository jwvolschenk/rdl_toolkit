.PHONY: build install build-all

build:
	go build -o bin/rdl-tool ./cmd/rdl-tool

install:
	go install ./cmd/rdl-tool

build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/rdl-tool-linux-amd64 ./cmd/rdl-tool
	GOOS=windows GOARCH=amd64 go build -o bin/rdl-tool-windows-amd64.exe ./cmd/rdl-tool
	GOOS=darwin GOARCH=amd64 go build -o bin/rdl-tool-darwin-amd64 ./cmd/rdl-tool
	GOOS=darwin GOARCH=arm64 go build -o bin/rdl-tool-darwin-arm64 ./cmd/rdl-tool
