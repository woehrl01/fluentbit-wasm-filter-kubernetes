.PHONY: build

setup:
	@echo "Setup target..."
	rustup target add wasm32-unknown-unknown
	cargo install -f wasm-bindgen-cli


build: 
	@echo "Building..."
	cargo build --target wasm32-unknown-unknown --release
	wasm-bindgen target/wasm32-unknown-unknown/release/filter_rust.wasm --out-dir pkg

test:
	@echo "Testing..."
	cargo test
