.PHONY: build

build:
	tinygo build -target=wasi -o filter.wasm filter.go

integration:
	docker compose up

test:
	go test -v ./...
