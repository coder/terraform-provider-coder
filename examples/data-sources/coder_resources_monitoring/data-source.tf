provider "coder" {}

data "coder_provisioner" "dev" {}

data "coder_workspace" "dev" {}

resource "coder_agent" "main" {
  arch = data.coder_provisioner.dev.arch
  os   = data.coder_provisioner.dev.os
  dir  = "/workspace"
  resources_monitoring {
    memory {
      enabled   = true
      threshold = 80
    }
    volume {
      path      = "/volume1"
      enabled   = true
      threshold = 80
    }
    volume {
      path      = "/volume2"
      enabled   = true
      threshold = 100
    }
  }
}