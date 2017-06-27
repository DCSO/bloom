## simple makefile to log workflow
.PHONY: all test clean build install

GOFLAGS ?= $(GOFLAGS:)

all: install test

build:
	@go build $(GOFLAGS) ./...

install:
	@go get $(GOFLAGS) ./...

test: install
	@go test -cover $(GOFLAGS) ./...

bench: install
	@go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i ./...

release:
	@go get $(GOFLAGS) ./...
	@go build -v -o bloom_linux_amd64.bin bloom/*
	GOOS=windows GOARCH=amd64 go build -v -o bloom_windows_amd64.exe bloom/*

## EOF
