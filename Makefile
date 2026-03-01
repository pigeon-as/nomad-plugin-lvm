PLUGIN_NAME := nomad-plugin-lvm

.PHONY: build test clean

build:
	go build -o $(PLUGIN_NAME) .

test:
	go test -v ./...

clean:
	rm -f $(PLUGIN_NAME)
