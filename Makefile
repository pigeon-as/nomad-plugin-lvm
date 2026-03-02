PLUGIN_NAME := nomad-plugin-lvm

.PHONY: build test e2e clean

build:
	go build -o build/$(PLUGIN_NAME) ./cmd/nomad-plugin-lvm

test:
	go test -v ./cmd/...

e2e: build
	go test -tags=e2e -v -count=1 ./e2e

clean:
	rm -rf build/
