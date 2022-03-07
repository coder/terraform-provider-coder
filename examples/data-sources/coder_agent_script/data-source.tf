data "coder_agent_script" "dev" {
  os   = "darwin"
  arch = "amd64"
}

resource "kubernetes_pod" "dev" {
  spec {
    container {
      command = ["sh", "-c", data.coder_agent_script.dev.value]
    }
  }
}
