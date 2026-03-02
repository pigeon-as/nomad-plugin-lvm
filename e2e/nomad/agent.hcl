# Nomad dev agent config for e2e testing of the LVM host volume plugin.
# Usage: nomad agent -dev -plugin-dir=<build-dir> -config=e2e/nomad/agent.hcl

client {
  host_volume_plugin "lvm" {
    # The plugin binary is discovered from -plugin-dir.
  }
}
