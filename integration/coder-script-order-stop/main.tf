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

variable "output_path" {
  type = string
}

# Every file the test reads is named after the output path, so a copied
# template cannot share a file with another case.
locals {
  case      = trimsuffix(var.output_path, ".json")
  order_log = "${local.case}.log"
}

resource "coder_script" "save_state" {
  agent_id     = coder_agent.main.id
  display_name = "Save state"
  run_on_stop  = true
  script       = <<-EOT
    echo "save_state start" >> ${local.order_log}
    sleep 2
    echo "save_state end" >> ${local.order_log}
  EOT
}

resource "coder_script" "stop_db" {
  agent_id     = coder_agent.main.id
  display_name = "Stop DB"
  run_on_stop  = true
  script       = <<-EOT
    echo "stop_db start" >> ${local.order_log}
    echo "stop_db end" >> ${local.order_log}
  EOT
}

data "coder_script_order" "shutdown" {
  rule {
    run   = ["coder_script.stop_db"]
    after = ["coder_script.save_state"]
    phase = "stop"
  }
}

resource "local_file" "output" {
  filename = var.output_path
  content = jsonencode({
    "workspace.id"        = data.coder_workspace.me.id
    "script_order.id"     = data.coder_script_order.shutdown.id
    "script_order.rule.#" = tostring(length(data.coder_script_order.shutdown.rule))
  })
}

resource "local_sensitive_file" "agent_token" {
  filename = "${local.case}.agent-token"
  content  = coder_agent.main.token
}

resource "local_sensitive_file" "session_token" {
  filename = "${local.case}.session-token"
  content  = data.coder_workspace_owner.me.session_token
}
