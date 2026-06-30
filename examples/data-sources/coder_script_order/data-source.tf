provider "coder" {}

resource "coder_agent" "main" {
  os   = "linux"
  arch = "amd64"
}

resource "coder_script" "clone" {
  agent_id     = coder_agent.main.id
  display_name = "Clone repository"
  run_on_start = true
  script       = "git clone https://github.com/coder/coder ~/coder"
}

resource "coder_script" "install" {
  agent_id     = coder_agent.main.id
  display_name = "Install dependencies"
  run_on_start = true
  script       = "cd ~/coder && make install"
}

data "coder_script_order" "startup" {
  rule {
    run   = ["coder_script.install"]
    after = ["coder_script.clone"]
  }
}
