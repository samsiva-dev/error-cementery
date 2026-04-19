BINARY := cemetery
BIN_DIR := bin

.PHONY: build install clean test

build:
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY) ./cmd/cemetery

install: build
	cp $(BIN_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -rf $(BIN_DIR)/

test:
	go test ./...
