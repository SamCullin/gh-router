SHELL := /bin/sh

BINARY := gh-router
VERSION ?= dev
LDFLAGS := -s -w -X github.com/SamCullin/gh-router/internal/version.Version=$(VERSION)

.PHONY: build test vet fmt install clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/gh-router

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal

install:
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/gh-router

clean:
	rm -rf bin
