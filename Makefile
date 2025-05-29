.PHONY: build clean test release-local install dev

# Variables
BINARY_NAME=transcriber
VERSION?=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"
DIST_DIR=dist

# Default target
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(DIST_DIR)

# Run tests
test:
	go test -v ./...

# Build for all platforms (local release)
release-local: clean
	mkdir -p $(DIST_DIR)
	
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	
	# Create archives
	cd $(DIST_DIR) && \
	tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
	tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
	tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
	tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
	zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe

# Install locally
install: build
	sudo mv $(BINARY_NAME) /usr/local/bin/

# Development build with race detection
dev:
	go build -race $(LDFLAGS) -o $(BINARY_NAME) .

newtag:
	@echo "Creating new tag...Last tag: $(VERSION)"
	@read -p "Enter new version tag (e.g., v1.0.0): " new_tag; \
	if [ -z "$$new_tag" ]; then \
		echo "Tag cannot be empty"; \
		exit 1; \
	fi; \
	git tag "$$new_tag"; \
	git push origin "$$new_tag"; \
	echo "Tag $$new_tag created and pushed."