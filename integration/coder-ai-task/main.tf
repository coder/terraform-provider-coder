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

resource "coder_app" "ai_interface" {
  agent_id = coder_agent.dev.id
  slug     = "ai-chat"
  share    = "owner"
  url      = "http://localhost:8080"
}

data "coder_parameter" "ai_prompt" {
  type        = "string"
  name        = "AI Prompt"
  default     = ""
  description = "Write a prompt for Claude Code"
  mutable     = true
}

data "coder_task" "me" {}

resource "coder_ai_task" "task" {
  sidebar_app {
    id = coder_app.ai_interface.id
  }
}

locals {
  # NOTE: these must all be strings in the output
  output = {
    "ai_task.id"      = coder_ai_task.task.id
    "ai_task.app_id"  = coder_ai_task.task.app_id
    "ai_task.prompt"  = coder_ai_task.task.prompt
    "ai_task.enabled" = tostring(coder_ai_task.task.enabled)
    "app.id"          = coder_app.ai_interface.id

    "task.id"      = data.coder_task.me.id
    "task.prompt"  = data.coder_task.me.prompt
    "task.enabled" = tostring(data.coder_task.me.enabled)
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
