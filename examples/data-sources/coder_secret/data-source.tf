data "coder_secret" "github_token" {
  env          = "GITHUB_TOKEN"
  help_message = "Create a GitHub personal access token and add it as a secret with env=GITHUB_TOKEN"
}

data "coder_secret" "aws_credentials" {
  file         = "~/.aws/credentials"
  help_message = "Add your AWS credentials file as a secret with file=~/.aws/credentials"
}

# Use the secret value in an agent startup script.
resource "coder_script" "setup" {
  agent_id = coder_agent.main.id
  script   = "echo ${data.coder_secret.github_token.value} | gh auth login --with-token"
}
