# nomad-plugin-lvm

A [Nomad dynamic host volume plugin](https://developer.hashicorp.com/nomad/plugins/author/host-volume) for LVM thin provisioning.

## Install

```bash
make build
cp nomad-plugin-lvm /opt/nomad/host_volume_plugins/
```

Create `nomad-plugin-lvm.json` next to the binary:

```json
{
  "volume_group": "vg0",
  "thin_pool": "thinpool0"
}
```

Reload the Nomad client (`SIGHUP` or `systemctl reload nomad`).

## Persistent volumes

```hcl
type      = "host"
name      = "myapp-data"
plugin_id = "nomad-plugin-lvm"

capacity_min = "1G"
capacity_max = "10G"

capability {
  access_mode     = "single-node-single-writer"
  attachment_mode = "file-system"
}
```

```bash
nomad volume create myapp-data.volume.hcl
```

Creates a thin LV with an ext4 filesystem. Override with `parameters { filesystem = "xfs" }`.

## Snapshots

Create a thin snapshot of an existing LV:

```hcl
type      = "host"
name      = "myapp-snap"
plugin_id = "nomad-plugin-lvm"

parameters {
  type   = "snapshot"
  source = "myapp-base"
}
```

The source LV must already exist in the same volume group.

## Parameters

Per-volume parameters in the `parameters {}` block:

| Parameter    | Default      | Description                                          |
|--------------|--------------|------------------------------------------------------|
| `type`       | `persistent` | `persistent` (new thin LV) or `snapshot` (COW clone) |
| `source`     | —            | Source LV name (required when `type = "snapshot"`)   |
| `filesystem` | `ext4`       | Filesystem for persistent volumes                    |

## Requirements

- Nomad 1.10+ with [dynamic host volumes](https://developer.hashicorp.com/nomad/docs/configuration/client#host_volume_plugin_dir)
- LVM2 tools (`lvcreate`, `lvremove`, `lvs`, `lvchange`)
- An existing LVM thin pool