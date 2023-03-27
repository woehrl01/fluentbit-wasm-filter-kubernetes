.PHONY: build

# gc=leaking is allowed as the filter is loaded for each line of the log

build:
	tinygo build -target=wasi -o filter.wasm -gc=leaking -no-debug -opt=2  filter.go
	wasm-opt -Oz -o filter.wasm filter.wasm

build-debug:
	tinygo build -target=wasi -o filter.wasm filter.go

integration:
	docker compose up

test:
	go test -v ./...
