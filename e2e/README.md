# E2E Tests

Exercises the plugin binary against a real LVM thin pool on a loopback device.

## Requirements

- Linux
- Root privileges
- `lvm2` (provides `lvcreate`, `lvremove`, `vgcreate`, etc.)
- `e2fsprogs` (provides `mkfs.ext4`, `blkid`)

## Usage

Build the plugin and run the tests:

```sh
sudo make e2e
```

Or manually:

```sh
make build
sudo go test -tags=e2e -v -count=1 ./e2e
```

The tests create a 200MB loopback thin pool, run all plugin operations, and tear everything down on exit.
