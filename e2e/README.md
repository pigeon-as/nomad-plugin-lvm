# E2E Tests

Two test layers exercise the plugin at different levels.

## Plugin Tests (`e2e/plugin/`)

Tests the plugin binary directly against a real LVM thin pool on a loopback
device. No Nomad agent required.

### Requirements

- Linux
- Root privileges
- `lvm2` (provides `lvcreate`, `lvremove`, `vgcreate`, etc.)
- `e2fsprogs` (provides `mkfs.ext4`, `blkid`)

### Usage

```sh
sudo make e2e-plugin
```

Or manually:

```sh
make build
sudo go test -tags=e2e -v -count=1 ./e2e/plugin
```

The tests create a 200MB loopback thin pool, run all plugin operations, and
tear everything down on exit.

## Nomad Tests (`e2e/nomad/`)

Tests the full lifecycle through a running Nomad dev agent. Submits job specs
that reference dynamic host volumes and verifies they are created, mounted, and
cleaned up correctly.

### Requirements

- Linux
- Nomad binary on `$PATH`
- The plugin built and available in the plugin dir
- Root privileges (for the Nomad agent)

### Usage

In one terminal, start the dev agent:

```sh
make dev
```

In another terminal, run the tests:

```sh
make e2e-nomad
```
