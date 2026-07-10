resource "coder_agent" "dev" {
  os   = "linux"
  arch = "amd64"
}

resource "coder_script" "clone" {
  agent_id     = coder_agent.dev.id
  display_name = "clone"
  script       = "git clone ..."
  run_on_start = true
}

resource "coder_script" "install" {
  agent_id     = coder_agent.dev.id
  display_name = "install"
  script       = "make install"
  run_on_start = true
}

# install waits until clone completes.
resource "coder_script_order" "startup" {
  rule {
    run   = [coder_script.install.id]
    after = [coder_script.clone.id]
    state = "completes"
  }
}
