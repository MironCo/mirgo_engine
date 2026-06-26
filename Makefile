BINARY_NAME=mirgo_engine
CMD_PATH=./cmd/editor
UTILS_PATH=./cmd/mirgo_utils

.PHONY: all build run build-game run-game clean gen-scripts test utils

all: build

build: gen-scripts
	go build -o $(BINARY_NAME) $(CMD_PATH)

run: gen-scripts
	go run $(CMD_PATH)

gen-scripts:
	@go run ./cmd/mirgo_utils genscripts

utils:
	go build -o mirgo_utils $(UTILS_PATH)

build-game:
	go build -tags game -o $(BINARY_NAME) $(CMD_PATH)

run-game:
	go run -tags game $(CMD_PATH)

test:
	@echo "Checking formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ Code not formatted. Run 'gofmt -w .'"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "✓ Formatting OK"
	@echo "Running linter (warnings only)..."
	@golangci-lint run --timeout 5m || echo "⚠️  Linter found issues (non-blocking)"
	@go test ./engine/... ./cmd/gen-scripts/...

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)-linux mirgo_utils

# Cross-compilation is tricky with raylib due to CGO
# These targets require the appropriate cross-compilers and libs installed

build-windows:
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 \
	go build -o $(BINARY_NAME).exe $(CMD_PATH)

build-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
	go build -o $(BINARY_NAME)-linux $(CMD_PATH)
