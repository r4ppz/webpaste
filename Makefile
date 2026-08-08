BINARY_NAME := web-paste
BIN_DIR := ./bin
SRC := ./...

.PHONY: all build run clean fmt vet lint test

# Default target
all: build

# Compile the binary
build: $(BIN_DIR)/$(BINARY_NAME)

$(BIN_DIR)/$(BINARY_NAME): $(shell find . -name '*.go')
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(SRC)

# Run the binary directly (builds if needed)
run: build
	$(BIN_DIR)/$(BINARY_NAME)

# Remove generated artifacts
clean:
	@rm -rf $(BIN_DIR)

# Format source code
fmt:
	go fmt $(SRC)

# Run go vet
vet:
	go vet $(SRC)

# Lint using staticcheck (optional, requires installation)
lint:
	staticcheck $(SRC)

# Run tests (if any)
test:
	go test ./...
