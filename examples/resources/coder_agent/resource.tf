data "coder_workspace" "me" {
}

resource "coder_agent" "dev" {
  os   = "linux"
  arch = "amd64"
  dir  = "/workspace"
}

resource "kubernetes_pod" "dev" {
  count = data.coder_workspace.me.start_count
  spec {
    container {
      command = ["sh", "-c", coder_agent.dev.init_script]
      env {
        name  = "CODER_AGENT_TOKEN"
        value = coder_agent.dev.token
      }
    }
  }
}
