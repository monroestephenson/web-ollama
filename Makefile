.PHONY: build install test clean run deps

BINARY_NAME=web-ollama
INSTALL_PATH=$(HOME)/bin

build:
	go build -o $(BINARY_NAME) -ldflags="-s -w" .

install: build
	mkdir -p $(INSTALL_PATH)
	cp $(BINARY_NAME) $(INSTALL_PATH)/
	chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"
	@echo "Make sure $(INSTALL_PATH) is in your PATH"

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)
	rm -rf ~/.web-ollama/

run: build
	./$(BINARY_NAME)

deps:
	go mod download
	go mod tidy

# Cross-compilation targets
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 -ldflags="-s -w" .

build-darwin-arm:
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 -ldflags="-s -w" .

build-darwin-amd:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 -ldflags="-s -w" .

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe -ldflags="-s -w" .

build-all: build-linux build-darwin-arm build-darwin-amd build-windows
