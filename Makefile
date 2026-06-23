.PHONY: build install build-all clean

LDFLAGS=-ldflags "-s -w"

build:
	go build $(LDFLAGS) -o bin/rdl-tool ./cmd/rdl-tool

install:
	go install $(LDFLAGS) ./cmd/rdl-tool

build-all:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/rdl-tool-linux-amd64 ./cmd/rdl-tool
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/rdl-tool-linux-arm64 ./cmd/rdl-tool
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/rdl-tool-windows-amd64.exe ./cmd/rdl-tool
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/rdl-tool-darwin-amd64 ./cmd/rdl-tool
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/rdl-tool-darwin-arm64 ./cmd/rdl-tool

clean:
	rm -rf bin/ .release/
