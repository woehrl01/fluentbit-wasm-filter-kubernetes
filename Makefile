.PHONY: build

setup:
	@echo "Setup target..."
	rustup target add wasm32-unknown-unknown

build: 
	@echo "Building..."
	rm -rf ./target
	rm -rf ./pkg
	cargo build --target wasm32-unknown-unknown --release

test:
	@echo "Testing..."
	cargo test
