BINARY_NAME=test3d
CMD_PATH=./cmd/test3d

.PHONY: all build run clean

all: build

build:
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)

run:
	go run $(CMD_PATH)

clean:
	rm -rf bin/

# Cross-compilation is tricky with raylib due to CGO
# These targets require the appropriate cross-compilers and libs installed

build-windows:
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 \
	go build -o bin/$(BINARY_NAME).exe $(CMD_PATH)

build-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
	go build -o bin/$(BINARY_NAME)-linux $(CMD_PATH)
