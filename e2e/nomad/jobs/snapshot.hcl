# Test job: creates a snapshot volume from a golden source.
job "test-snapshot" {
  type = "batch"

  group "test" {
    volume "data" {
      type   = "host"
      source = "test-snapshot"
    }

    task "verify" {
      driver = "exec"

      volume_mount {
        volume      = "data"
        destination = "/data"
      }

      config {
        command = "/bin/sh"
        args    = ["-c", "echo snapshot-ok > /data/check.txt && cat /data/check.txt"]
      }

      resources {
        cpu    = 100
        memory = 64
      }
    }
  }
}
