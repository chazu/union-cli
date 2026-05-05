BINARY := union
PKG    := ./cmd/union
GOBIN  := $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif
CODESIGN_IDENTITY ?= -

.PHONY: build install codesign clean

build:
	go build -o $(BINARY) $(PKG)
	codesign --force --sign $(CODESIGN_IDENTITY) $(BINARY)

install:
	go install $(PKG)
	codesign --force --sign $(CODESIGN_IDENTITY) $(GOBIN)/$(BINARY)

codesign:
	codesign --force --sign $(CODESIGN_IDENTITY) $(GOBIN)/$(BINARY)

clean:
	rm -f $(BINARY)
