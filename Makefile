PLUGIN_NAME := nomad-plugin-lvm

.PHONY: build test e2e-plugin e2e-nomad e2e dev clean

build:
	go build -o build/$(PLUGIN_NAME) ./cmd/nomad-plugin-lvm

test:
	go test -v ./internal/...

e2e-plugin: build
	sudo go test -tags=e2e -v -count=1 ./e2e/plugin

e2e-nomad:
	go test -tags=e2e -v -count=1 ./e2e/nomad

e2e: e2e-plugin e2e-nomad

dev: build
	sudo nomad agent -dev -plugin-dir=$$(pwd)/build -config=e2e/nomad/agent.hcl

clean:
	rm -rf build/
