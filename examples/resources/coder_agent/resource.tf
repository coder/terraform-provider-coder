data "coder_agent_script" "dev" {
  os   = "linux"
  arch = "amd64"
}

resource "coder_agent" "dev" {
  startup_script = "code-server"
}

resource "google_compute_instance" "dev" {
  spec {
    container {
      command = ["sh", "-c", data.coder_agent_script.dev.value]
      env {
        name  = "CODER_TOKEN"
        value = coder_agent.dev.token
      }
    }
  }
}
