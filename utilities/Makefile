BINARY_NAME=mirgo-utils
OUT_DIR=..

.PHONY: all build clean

all: build

build:
	cargo build --release
	cp target/release/$(BINARY_NAME) $(OUT_DIR)/$(BINARY_NAME)

clean:
	cargo clean
	rm -f $(OUT_DIR)/$(BINARY_NAME)
