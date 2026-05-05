BINARY := union
PKG    := ./cmd/union
GOBIN  := $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif
CODESIGN_IDENTITY ?= -

.PHONY: build install codesign clean test vet

build:
	go build -o $(BINARY) $(PKG)
	codesign --force --sign $(CODESIGN_IDENTITY) $(BINARY)

install:
	go install $(PKG)
	codesign --force --sign $(CODESIGN_IDENTITY) $(GOBIN)/$(BINARY)

codesign:
	codesign --force --sign $(CODESIGN_IDENTITY) $(GOBIN)/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
