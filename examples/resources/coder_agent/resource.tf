data "coder_workspace" "me" {
}

resource "coder_agent" "dev" {
  os   = "linux"
  arch = "amd64"
  dir  = "/workspace"
  display_apps {
    vscode          = true
    vscode_insiders = false
    web_terminal    = true
    ssh_helper      = false
  }
}
