GO ?= go
BINARY_NAME := webpaste
BIN_DIR := bin
PREFIX ?= $(HOME)/.local
DESTDIR ?=
SRC := ./...

.PHONY: all build run install clean fmt vet lint test staticcheck

all: build

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -o $(BIN_DIR)/$(BINARY_NAME) .

run: build
	./$(BIN_DIR)/$(BINARY_NAME)

clean:
	@rm -rf $(BIN_DIR)

fmt:
	$(GO) fmt $(SRC)

vet:
	$(GO) vet $(SRC)

staticcheck:
	$(GO) run honnef.co/go/tools/cmd/staticcheck@latest $(SRC)

lint: vet staticcheck

test:
	$(GO) test ./...

install: build
	install -Dm755 $(BIN_DIR)/$(BINARY_NAME) $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)
