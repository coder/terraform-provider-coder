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

// TODO: test coder_external_auth
// data coder_external_auth "me" {}
data "coder_provisioner" "me" {}
data "coder_workspace" "me" {}
data "coder_workspace_owner" "me" {}
data "coder_parameter" "param" {
  name        = "param"
  description = "param description"
  icon        = "param icon"
}
data "coder_workspace_preset" "preset" {
  name        = "preset"
  description = "preset description"
  icon        = "preset icon"
  default     = true
  parameters = {
    (data.coder_parameter.param.name) = "preset param value"
  }

  prebuilds {
    instances = 1
    expiration_policy {
      ttl = 86400
    }
    scheduling {
      timezone = "UTC"
      schedule {
        cron      = "* 8-18 * * 1-5"
        instances = 3
      }
      schedule {
        cron      = "* 8-14 * * 6"
        instances = 1
      }
    }
  }
}

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
    "workspace_parameter.name" : data.coder_parameter.param.name,
    "workspace_parameter.description" : data.coder_parameter.param.description,
    "workspace_parameter.value" : data.coder_parameter.param.value,
    "workspace_parameter.icon" : data.coder_parameter.param.icon,
    "workspace_preset.name" : data.coder_workspace_preset.preset.name,
    "workspace_preset.description" : data.coder_workspace_preset.preset.description,
    "workspace_preset.icon" : data.coder_workspace_preset.preset.icon,
    "workspace_preset.default" : tostring(data.coder_workspace_preset.preset.default),
    "workspace_preset.parameters.param" : data.coder_workspace_preset.preset.parameters.param,
    "workspace_preset.prebuilds.instances" : tostring(one(data.coder_workspace_preset.preset.prebuilds).instances),
    "workspace_preset.prebuilds.expiration_policy.ttl" : tostring(one(one(data.coder_workspace_preset.preset.prebuilds).expiration_policy).ttl),
    "workspace_preset.prebuilds.scheduling.timezone" : tostring(one(one(data.coder_workspace_preset.preset.prebuilds).scheduling).timezone),
    "workspace_preset.prebuilds.scheduling.schedule0.cron" : tostring(one(one(data.coder_workspace_preset.preset.prebuilds).scheduling).schedule[0].cron),
    "workspace_preset.prebuilds.scheduling.schedule0.instances" : tostring(one(one(data.coder_workspace_preset.preset.prebuilds).scheduling).schedule[0].instances),
    "workspace_preset.prebuilds.scheduling.schedule1.cron" : tostring(one(one(data.coder_workspace_preset.preset.prebuilds).scheduling).schedule[1].cron),
    "workspace_preset.prebuilds.scheduling.schedule1.instances" : tostring(one(one(data.coder_workspace_preset.preset.prebuilds).scheduling).schedule[1].instances),
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
