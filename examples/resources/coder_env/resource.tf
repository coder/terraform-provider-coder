data "coder_workspace" "me" {}

resource "coder_agent" "dev" {
  os   = "linux"
  arch = "amd64"
  dir  = "/workspace"
}

resource "coder_env" "welcome_message" {
  agent_id = coder_agent.dev.id
  name     = "WELCOME_MESSAGE"
  value    = "Welcome to your Coder workspace!"
}

resource "coder_env" "internal_api_url" {
  agent_id = coder_agent.dev.id
  name     = "INTERNAL_API_URL"
  value    = "https://api.internal.company.com/v1"
}

# Append to PATH without losing the directories already set by the
# workspace's image. Only one coder_env resource needs to reference
# $PATH; the others can append plain values.
resource "coder_env" "path_cuda" {
  agent_id       = coder_agent.dev.id
  name           = "PATH"
  value          = "$PATH:/usr/local/cuda/bin"
  merge_strategy = "append"
}

resource "coder_env" "path_go" {
  agent_id       = coder_agent.dev.id
  name           = "PATH"
  value          = "/usr/local/go/bin"
  merge_strategy = "append"
}
