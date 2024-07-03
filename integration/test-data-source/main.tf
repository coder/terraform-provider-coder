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

// TODO: test coder_external_auth and coder_git_auth
// data coder_external_auth "me" {}
// data coder_git_auth "me" {}
data "coder_provisioner" "me" {}
data "coder_workspace" "me" {}
data "coder_workspace_owner" "me" {}

locals {
  # NOTE: these must all be strings in the output
  output = {
    "provisioner.arch" : data.coder_provisioner.me.arch,
    "provisioner.id" : data.coder_provisioner.me.id,
    "provisioner.os" : data.coder_provisioner.me.os,
    "workspace.access_port" : tostring(data.coder_workspace.me.access_port),
    "workspace.access_url" : data.coder_workspace.me.access_url,
    "workspace.id" : data.coder_workspace.me.id,
    "workspace.name" : data.coder_workspace.me.name,
    "workspace.start_count" : tostring(data.coder_workspace.me.start_count),
    "workspace.template_id" : data.coder_workspace.me.template_id,
    "workspace.template_name" : data.coder_workspace.me.template_name,
    "workspace.template_version" : data.coder_workspace.me.template_version,
    "workspace.transition" : data.coder_workspace.me.transition,
  }
}

variable "output_path" {
  type = string
}

resource "local_file" "output" {
  filename = var.output_path
  content  = jsonencode(local.output)
}

output "output" {
  value     = local.output
  sensitive = true
}
