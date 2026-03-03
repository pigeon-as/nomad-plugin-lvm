name      = "test-golden"
type      = "host"
plugin_id = "nomad-plugin-lvm"

capability {
  access_mode     = "single-node-writer"
  attachment_mode = "file-system"
}

capacity_min = "10MiB"
capacity_max = "50MiB"

parameters {
  type       = "persistent"
  filesystem = "ext4"
}
