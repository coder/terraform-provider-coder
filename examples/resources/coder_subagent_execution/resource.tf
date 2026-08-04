resource "coder_agent" "dev" {
  os   = "linux"
  arch = "amd64"
}

resource "coder_subagent_execution" "sandbox" {
  agent_id          = coder_agent.dev.id
  name              = "sandbox"
  driver            = "example-driver-configuration"
  shared_host_path  = "/workspace"
  shared_child_path = "/workspace"
}

resource "coder_app" "sandbox_terminal" {
  agent_id = coder_subagent_execution.sandbox.subagent_id
  slug     = "sandbox-terminal"
  command  = "bash"
}

resource "coder_script" "sandbox_setup" {
  agent_id     = coder_subagent_execution.sandbox.subagent_id
  display_name = "Sandbox setup"
  run_on_start = true
  script       = "echo sandbox ready"
}

resource "coder_env" "sandbox_name" {
  agent_id = coder_subagent_execution.sandbox.subagent_id
  name     = "SANDBOX_NAME"
  value    = coder_subagent_execution.sandbox.name
}
