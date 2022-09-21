data "coder_workspace" "me" {}

resource "coder_agent" "dev" {
  os             = "linux"
  arch           = "amd64"
  dir            = "/workspace"
  startup_script = <<EOF
curl -fsSL https://code-server.dev/install.sh | sh
code-server --auth none --port 13337
EOF
}

resource "coder_app" "code-server" {
  agent_id      = coder_agent.dev.id
  name          = "VS Code"
  icon          = data.coder_workspace.me.access_url + "/icons/vscode.svg"
  url           = "http://localhost:13337"
  relative_path = true
}

resource "coder_app" "vim" {
  agent_id = coder_agent.dev.id
  name     = "Vim"
  icon     = data.coder_workspace.me.access_url + "/icons/vim.svg"
  command  = "vim"
}

resource "coder_app" "intellij" {
  agent_id = coder_agent.dev.id
  icon     = data.coder_workspace.me.access_url + "/icons/intellij.svg"
  name     = "JetBrains IntelliJ"
  command  = "projector run"
}
