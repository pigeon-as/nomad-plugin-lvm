# Test job: creates a persistent dynamic host volume and writes to it.
job "test-persistent" {
  type = "batch"

  group "test" {
    volume "data" {
      type   = "host"
      source = "test-persistent"
    }

    task "write" {
      driver = "exec"

      volume_mount {
        volume      = "data"
        destination = "/data"
      }

      config {
        command = "/bin/sh"
        args    = ["-c", "echo hello > /data/test.txt && cat /data/test.txt"]
      }

      resources {
        cpu    = 100
        memory = 64
      }
    }
  }
}
