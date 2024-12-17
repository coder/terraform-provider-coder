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

resource "coder_app" "window" {
  agent_id = coder_agent.dev.id
  slug     = "window"
  share    = "owner"
  open_in  = "window"
}

resource "coder_app" "slim-window" {
  agent_id = coder_agent.dev.id
  slug     = "slim-window"
  share    = "owner"
  open_in  = "slim-window"
}

resource "coder_app" "defaulted" {
  agent_id = coder_agent.dev.id
  slug     = "defaulted"
  share    = "owner"
}

locals {
  # NOTE: these must all be strings in the output
  output = {
    "coder_app.window.open_in"    = tostring(coder_app.window.open_in)
    "coder_app.slim-window.open_in"   = tostring(coder_app.slim-window.open_in)
    "coder_app.defaulted.open_in" = tostring(coder_app.defaulted.open_in)
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

