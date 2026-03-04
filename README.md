# nomad-plugin-lvm

A [Nomad dynamic host volume plugin](https://developer.hashicorp.com/nomad/plugins/author/host-volume) for LVM thin provisioning.

## Install

```bash
make build
cp build/nomad-plugin-lvm /opt/nomad/host_volume_plugins/
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

parameters {
  volume_group = "vg0"
  thin_pool    = "thinpool0"
}
```

```bash
nomad volume create myapp-data.volume.hcl
```

Creates a thin LV with an ext4 filesystem.

## Snapshots

Create a thin snapshot of an existing LV:

```hcl
type      = "host"
name      = "myapp-snap"
plugin_id = "nomad-plugin-lvm"

parameters {
  type         = "snapshot"
  source       = "myapp-base"
  volume_group = "vg0"
  thin_pool    = "thinpool0"
}
```

The source LV must already exist in the same volume group.

## Parameters

All configuration is passed through the volume definition's `parameters {}` block:

| Parameter      | Required | Default              | Description                                          |
|----------------|----------|----------------------|------------------------------------------------------|
| `volume_group` | yes      | —                    | LVM volume group name                                |
| `thin_pool`    | yes      | —                    | Thin pool name                                       |
| `type`         | no       | `persistent`         | `persistent` (new thin LV) or `snapshot` (COW clone) |
| `source`       | snapshot | —                    | Source LV name (required when `type = "snapshot"`)   |
| `filesystem`   | no       | `ext4`               | Filesystem for persistent volumes                    |
| `mode`         | no       | `filesystem`         | `filesystem` or `block`                              |
| `mount_dir`    | no       | `/srv/nomad-volumes` | Volume mount directory                               |

## Requirements

- Nomad 1.10+ with [dynamic host volumes](https://developer.hashicorp.com/nomad/docs/configuration/client#host_volume_plugin_dir)
- LVM2 tools (`lvcreate`, `lvremove`, `lvs`, `lvchange`)
- An existing LVM thin pool