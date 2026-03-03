# E2E Tests

Tests the full volume lifecycle (create, snapshot, delete) through a running
Nomad dev agent and a real LVM thin pool on a loopback device.

## Requirements

- Linux (WSL2 works)
- Root privileges
- `lvm2` (provides `lvcreate`, `lvremove`, `vgcreate`, etc.)
- `e2fsprogs` (provides `mkfs.ext4`)
- Nomad binary on `$PATH`

## Usage

In one terminal, start the dev agent:

```sh
make dev
```

In another terminal, run the tests:

```sh
make e2e
```
