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
data "coder_workspace" "me" {}
data "coder_workspace_owner" "me" {}

resource "coder_agent" "main" {
  os   = "linux"
  arch = data.coder_provisioner.me.arch
}

locals {
  order_log = "/tmp/coder-script-order.log"
}

# The sleep gives the unordered script time to start before this one ends.
resource "coder_script" "clone_repo" {
  agent_id     = coder_agent.main.id
  display_name = "Clone repo"
  run_on_start = true
  timeout      = 30
  script       = <<-EOT
    echo "clone_repo start" >> ${local.order_log}
    sleep 3
    echo "clone_repo end" >> ${local.order_log}
  EOT
}

resource "coder_script" "install_tools" {
  agent_id     = coder_agent.main.id
  display_name = "Install tools"
  run_on_start = true
  script       = <<-EOT
    echo "install_tools start" >> ${local.order_log}
    echo "install_tools end" >> ${local.order_log}
  EOT
}

resource "coder_script" "post_setup" {
  agent_id     = coder_agent.main.id
  display_name = "Post setup"
  run_on_start = true
  script       = <<-EOT
    echo "post_setup start" >> ${local.order_log}
    echo "post_setup end" >> ${local.order_log}
  EOT
}

resource "coder_script" "start_code_server" {
  agent_id     = coder_agent.main.id
  display_name = "code-server"
  run_on_start = true
  script       = <<-EOT
    echo "start_code_server start" >> ${local.order_log}
    echo "start_code_server end" >> ${local.order_log}
  EOT
}

data "coder_script_order" "bootstrap" {
  rule {
    run   = ["coder_script.install_tools"]
    after = ["coder_script.clone_repo"]
  }
  rule {
    run      = ["coder_script.post_setup"]
    after    = ["coder_script.clone_repo"]
    requires = "completion"
  }
}

variable "output_path" {
  type = string
}

resource "local_file" "output" {
  filename = var.output_path
  content = jsonencode({
    "workspace.id"        = data.coder_workspace.me.id
    "agent.id"            = coder_agent.main.id
    "script_order.id"     = data.coder_script_order.bootstrap.id
    "script_order.rule.#" = tostring(length(data.coder_script_order.bootstrap.rule))
  })
}

# The test reads these files inside the container. Keeping the tokens out of
# the output JSON keeps them out of the test log.
resource "local_sensitive_file" "agent_token" {
  filename = "/tmp/coder-script-order.agent-token"
  content  = coder_agent.main.token
}

resource "local_sensitive_file" "session_token" {
  filename = "/tmp/coder-script-order.session-token"
  content  = data.coder_workspace_owner.me.session_token
}
