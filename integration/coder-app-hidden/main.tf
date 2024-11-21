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

data "coder_workspace" "me" {}

resource "coder_agent" "dev" {
  os   = "linux"
  arch = "amd64"
  dir  = "/workspace"
}

resource "coder_app" "hidden" {
  agent_id = coder_agent.dev.id
  slug     = "hidden"
  share    = "owner"
  hidden   = true
}

resource "coder_app" "visible" {
  agent_id = coder_agent.dev.id
  slug     = "visible"
  share    = "owner"
  hidden   = false
}

resource "coder_app" "defaulted" {
  agent_id = coder_agent.dev.id
  slug     = "defaulted"
  share    = "owner"
}

locals {
  # NOTE: these must all be strings in the output
  output = {
    "coder_app.hidden.hidden"    = tostring(coder_app.hidden.hidden)
    "coder_app.visible.hidden"   = tostring(coder_app.visible.hidden)
    "coder_app.defaulted.hidden" = tostring(coder_app.defaulted.hidden)
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

