.PHONY: build test lint install clean

build:
	go build -o bin/unifi-network-pp-cli ./cmd/unifi-network-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/unifi-network-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/unifi-network-pp-mcp ./cmd/unifi-network-pp-mcp

install-mcp:
	go install ./cmd/unifi-network-pp-mcp

build-all: build build-mcp
