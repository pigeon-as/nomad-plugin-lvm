# Nomad dev agent config for e2e testing of the LVM host volume plugin.
# Run via: make dev
#
# The plugin binary is built into build/ by the Makefile.
# Nomad discovers it automatically by filename (nomad-plugin-lvm).

client {
  host_volume_plugin_dir = "/tmp/nomad-plugins"
}

plugin "raw_exec" {
  config {
    enabled = true
  }
}
