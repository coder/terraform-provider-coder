provider "coder" {}

data "coder_provisioner" "dev" {}

data "coder_workspace" "dev" {}

resource "coder_agent" "main" {
  arch = data.coder_provisioner.dev.arch
  os   = data.coder_provisioner.dev.os
  dir  = "/workspace"
  display_apps {
    vscode          = true
    vscode_insiders = false
    web_terminal    = true
    ssh_helper      = false
  }
}