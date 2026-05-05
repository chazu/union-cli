BINARY := union
PKG    := ./cmd/union
GOBIN  := $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif
CODESIGN_IDENTITY ?= -
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/chazu/union/internal/cli.Version=$(VERSION)"

.PHONY: build install codesign clean test vet

build:
	go build $(LDFLAGS) -o $(BINARY) $(PKG)
	codesign --force --sign $(CODESIGN_IDENTITY) $(BINARY)

install:
	go install $(LDFLAGS) $(PKG)
	codesign --force --sign $(CODESIGN_IDENTITY) $(GOBIN)/$(BINARY)

codesign:
	codesign --force --sign $(CODESIGN_IDENTITY) $(GOBIN)/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
