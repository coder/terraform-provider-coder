terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
    local = {
      source = "hashicorp/local"
    }
  }
}

data "coder_provisioner" "me" {}

resource "coder_agent" "main" {
  os   = "linux"
  arch = data.coder_provisioner.me.arch
}

resource "coder_script" "clone_repo" {
  agent_id     = coder_agent.main.id
  display_name = "Clone repo"
  run_on_start = true
  script       = "true"
}

resource "coder_script" "install_tools" {
  agent_id     = coder_agent.main.id
  display_name = "Install tools"
  run_on_start = true
  script       = "true"
}

data "coder_script_order" "bootstrap" {
  rule {
    run   = ["coder_script.install_tools"]
    after = ["coder_script.clone_repo"]
  }
}

variable "output_path" {
  type = string
}

# Coder does not count coder_script as a resource that references the
# agent, so without the agent ID here the agent is dropped.
resource "local_file" "output" {
  filename = var.output_path
  content = jsonencode({
    "agent.id"                     = coder_agent.main.id
    "script_order.id"              = data.coder_script_order.bootstrap.id
    "script_order.rule.0.run.0"    = data.coder_script_order.bootstrap.rule[0].run[0]
    "script_order.rule.0.after.0"  = data.coder_script_order.bootstrap.rule[0].after[0]
    "script_order.rule.0.requires" = data.coder_script_order.bootstrap.rule[0].requires
    "script_order.rule.0.phase"    = data.coder_script_order.bootstrap.rule[0].phase
  })
}
