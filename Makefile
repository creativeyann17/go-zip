.PHONY: all build build-all clean test check fmt run tidy dev install install-hooks

BINARY_NAME=gozip
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# -s: strip symbol table, -w: strip DWARF debug info
LDFLAGS="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

all: build

install:
	go mod tidy

build: install
	go build -trimpath -ldflags=$(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/gozip

# Build all common platforms (used by release workflow)
build-all: install
	mkdir -p dist/linux-amd64 dist/linux-arm64 dist/darwin-amd64 dist/darwin-arm64 dist/windows-amd64
	GOOS=linux   GOARCH=amd64    go build -trimpath -ldflags=$(LDFLAGS) -o dist/linux-amd64/$(BINARY_NAME)      ./cmd/gozip
	GOOS=linux   GOARCH=arm64    go build -trimpath -ldflags=$(LDFLAGS) -o dist/linux-arm64/$(BINARY_NAME)      ./cmd/gozip
	GOOS=darwin  GOARCH=amd64    go build -trimpath -ldflags=$(LDFLAGS) -o dist/darwin-amd64/$(BINARY_NAME)     ./cmd/gozip
	GOOS=darwin  GOARCH=arm64    go build -trimpath -ldflags=$(LDFLAGS) -o dist/darwin-arm64/$(BINARY_NAME)     ./cmd/gozip
	GOOS=windows GOARCH=amd64    go build -trimpath -ldflags=$(LDFLAGS) -o dist/windows-amd64/$(BINARY_NAME).exe ./cmd/gozip
	@echo "✓ Binaries built successfully in dist/"
	@echo "  Creating compressed archives..."
	@cd dist && tar -czf $(BINARY_NAME)-linux-amd64.tar.gz   -C linux-amd64   $(BINARY_NAME)     && echo "  - $(BINARY_NAME)-linux-amd64.tar.gz"
	@cd dist && tar -czf $(BINARY_NAME)-linux-arm64.tar.gz   -C linux-arm64   $(BINARY_NAME)     && echo "  - $(BINARY_NAME)-linux-arm64.tar.gz"
	@cd dist && tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz  -C darwin-amd64  $(BINARY_NAME)     && echo "  - $(BINARY_NAME)-darwin-amd64.tar.gz"
	@cd dist && tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz  -C darwin-arm64  $(BINARY_NAME)     && echo "  - $(BINARY_NAME)-darwin-arm64.tar.gz"
	@cd dist && zip -q $(BINARY_NAME)-windows-amd64.zip      -j windows-amd64/$(BINARY_NAME).exe && echo "  - $(BINARY_NAME)-windows-amd64.zip"
	@rm -rf dist/linux-amd64 dist/linux-arm64 dist/darwin-amd64 dist/darwin-arm64 dist/windows-amd64
	@cd dist && sha256sum *.tar.gz *.zip > checksums.txt && echo "  - checksums.txt"
	@echo "✓ Compressed archives created"

clean:
	rm -rf bin/ dist/

test: install
	go test ./... -count=1

# vet + race-detector tests: closest vanilla-Go gets to Rust's compile-time safety.
check:
	@fmtout=$$(gofmt -l .); if [ -n "$$fmtout" ]; then echo "✗ needs gofmt:"; echo "$$fmtout"; exit 1; fi
	go vet ./...
	go test -race ./... -count=1

fmt:
	gofmt -w .

run: build
	./bin/$(BINARY_NAME) --version

dev: install
	go run ./cmd/gozip --version

install-hooks:
	cp hooks/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
