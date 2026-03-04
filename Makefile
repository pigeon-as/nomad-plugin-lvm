PLUGIN_NAME := nomad-plugin-lvm
PLUGIN_DIR  := /tmp/nomad-plugins

.PHONY: build test e2e dev clean

build:
	go build -o build/$(PLUGIN_NAME) ./cmd/nomad-plugin-lvm

test:
	go test -v ./internal/...

e2e:
	sudo go test -tags=e2e -v -count=1 ./e2e

dev: build
	install -D build/$(PLUGIN_NAME) $(PLUGIN_DIR)/$(PLUGIN_NAME)
	sudo nomad agent -dev -config=$(abspath e2e/agent.hcl)

clean:
	rm -rf build/
