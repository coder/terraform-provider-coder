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