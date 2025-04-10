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
  agent_id     = coder_agent.dev.id
  slug         = "code-server"
  display_name = "VS Code"
  icon         = "${data.coder_workspace.me.access_url}/icon/code.svg"
  url          = "http://localhost:13337"
  share        = "owner"
  subdomain    = false
  open_in      = "window"
  healthcheck {
    url       = "http://localhost:13337/healthz"
    interval  = 5
    threshold = 6
  }
}

resource "coder_app" "vim" {
  agent_id     = coder_agent.dev.id
  slug         = "vim"
  display_name = "Vim"
  icon         = "${data.coder_workspace.me.access_url}/icon/vim.svg"
  command      = "vim"
}
