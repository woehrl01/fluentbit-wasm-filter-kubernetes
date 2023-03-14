.PHONY: build

setup:
	@echo "Setup target..."
	rustup target add wasm32-unknown-unknown
	rustup target add wasm32-wasi
	cargo install -f wasm-bindgen-cli


build: 
	@echo "Building..."
	rm -rf ./target
	rm -rf ./pkg
	cargo build --target wasm32-unknown-unknown --release
	cargo build --target wasm32-wasi --release 

test:
	@echo "Testing..."
	cargo test
