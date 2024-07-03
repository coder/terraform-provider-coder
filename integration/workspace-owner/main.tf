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
    "workspace.owner" : data.coder_workspace.me.owner,
    "workspace.owner_email" : data.coder_workspace.me.owner_email,
    "workspace.owner_groups" : jsonencode(data.coder_workspace.me.owner_groups),
    "workspace.owner_id" : data.coder_workspace.me.owner_id,
    "workspace.owner_name" : data.coder_workspace.me.owner_name,
    "workspace.owner_oidc_access_token" : data.coder_workspace.me.owner_oidc_access_token,
    "workspace.owner_session_token" : data.coder_workspace.me.owner_session_token,
    "workspace.start_count" : tostring(data.coder_workspace.me.start_count),
    "workspace.template_id" : data.coder_workspace.me.template_id,
    "workspace.template_name" : data.coder_workspace.me.template_name,
    "workspace.template_version" : data.coder_workspace.me.template_version,
    "workspace.transition" : data.coder_workspace.me.transition,
    "workspace_owner.email" : data.coder_workspace_owner.me.email,
    "workspace_owner.full_name" : data.coder_workspace_owner.me.full_name,
    "workspace_owner.groups" : jsonencode(data.coder_workspace_owner.me.groups),
    "workspace_owner.id" : data.coder_workspace_owner.me.id,
    "workspace_owner.name" : data.coder_workspace_owner.me.name,
    "workspace_owner.oidc_access_token" : data.coder_workspace_owner.me.oidc_access_token,
    "workspace_owner.session_token" : data.coder_workspace_owner.me.session_token,
    "workspace_owner.ssh_private_key" : data.coder_workspace_owner.me.ssh_private_key,
    "workspace_owner.ssh_public_key" : data.coder_workspace_owner.me.ssh_public_key,
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
