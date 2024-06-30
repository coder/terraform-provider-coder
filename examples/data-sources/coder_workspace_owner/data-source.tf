provider "coder" {}

data "coder_workspace" "me" {}

data "coder_workspace_owner" "me" {}

resource "coder_agent" "dev" {
  arch = "amd64"
  os   = "linux"
  dir  = local.repo_dir
  env = {
    OIDC_TOKEN : data.coder_workspace_owner.me.oidc_access_token,
  }
}

# Add git credentials from coder_workspace_owner
resource "coder_env" "git_author_name" {
  agent_id = coder_agent.agent_id
  name     = "GIT_AUTHOR_NAME"
  value    = coalesce(data.coder_workspace_owner.me.full_name, data.coder_workspace_owner.me.name)
}

resource "coder_env" "git_author_email" {
  agent_id = var.agent_id
  name     = "GIT_AUTHOR_EMAIL"
  value    = data.coder_workspace_owner.me.email
  count    = data.coder_workspace_owner.me.email != "" ? 1 : 0
}