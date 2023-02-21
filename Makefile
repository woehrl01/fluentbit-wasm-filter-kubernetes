.PHONY: build

toolchain:
	@echo "Downloading toolchain..."
	rustup target add wasm32-unknown-unknown


build:
	@echo "Building..."
	cargo build --target wasm32-unknown-unknown --release
