.PHONY: build test clean install

BIN_DIR := bin

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/hd-health ./cmd/hd-health
	go build -o $(BIN_DIR)/hd-health-agent ./cmd/hd-health-agent

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)

install: build
	@./deploy/macos/install.sh 2>/dev/null || ./deploy/fedora/install.sh 2>/dev/null || echo "Run platform install script manually"
